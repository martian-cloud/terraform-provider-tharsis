// Package provider contains the Tharsis provider configuration.
package provider

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	svchost "github.com/hashicorp/terraform-svchost"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/client"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/client/token"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ provider.Provider = (*tharsisProvider)(nil)

// Default scheme/protocol if user supplies only a host name.
const scheme string = "https://"

// New creates a new instance of the Tharsis provider
func New() provider.Provider {
	return &tharsisProvider{
		version: Version,
	}
}

// tharsisProvider satisfies the provider.Provider interface and usually is included
// with all Resource and DataSource implementations.
type tharsisProvider struct {
	// client is the combined Tharsis client for gRPC and REST API calls.
	client *client.GRPCClient
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
	// configured is set to true at the end of the Configure method.
	// This can be used in Resource and DataSource implementations to verify
	// that the provider was previously configured.
	configured bool
}

func (p *tharsisProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "tharsis"
}

func (p *tharsisProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	description := "Tharsis Terraform Provider is used to interact with a Tharsis instance using HCL."
	resp.Schema = schema.Schema{
		Description:         description,
		MarkdownDescription: description,
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Description:         "Host name of the Tharsis API (e.g. https://tharsis.example.com)",
				MarkdownDescription: "This is the hostname for the Tharsis API (e.g. https://tharsis.example.com).",
				Optional:            true,
			},
			"static_token": schema.StringAttribute{
				Description:         "Static token to authenticate with the Tharsis API",
				MarkdownDescription: "A static token to use to authenticate with the Tharsis API.",
				Optional:            true,
			},
			"service_account_path": schema.StringAttribute{
				Description:         "Service account path to use for authenticating with the Tharsis API",
				MarkdownDescription: "A Service account path to use for authenticating with the Tharsis API.",
				Optional:            true,
				DeprecationMessage:  "Use service_account_id instead. The path will be converted to a TRN automatically.",
			},
			"service_account_id": schema.StringAttribute{
				Description:         "Service account ID (TRN or GID) to use for authenticating with the Tharsis API",
				MarkdownDescription: "A Service account ID (TRN or GID) to use for authenticating with the Tharsis API.",
				Optional:            true,
			},
			"service_account_token": schema.StringAttribute{
				Description:         "Service account token to use for authenticating with the Tharsis API",
				MarkdownDescription: "A Service account token to use for authenticating with the Tharsis API.",
				Optional:            true,
			},
			"tls_skip_verify": schema.BoolAttribute{
				Description:         "Skip TLS certificate verification when connecting to the Tharsis API",
				MarkdownDescription: "Skip TLS certificate verification when connecting to the Tharsis API. For development use only.",
				Optional:            true,
			},
		},
	}
}

// providerData can be used to store data from the Terraform configuration.
type providerData struct {
	Host                types.String `tfsdk:"host"`
	StaticToken         types.String `tfsdk:"static_token"`
	ServiceAccountPath  types.String `tfsdk:"service_account_path"`
	ServiceAccountID    types.String `tfsdk:"service_account_id"`
	ServiceAccountToken types.String `tfsdk:"service_account_token"`
	TLSSkipVerify       types.Bool   `tfsdk:"tls_skip_verify"`
}

// checkUnknowns validates that no field is unknown during configuration
func (pd *providerData) checkUnknowns() diag.Diagnostics {
	var diags diag.Diagnostics
	if pd.Host.IsUnknown() {
		// Cannot connect to client with an unknown value
		diags = append(diags,
			diag.NewErrorDiagnostic(
				"Unknown host name",
				"Cannot use an unknown value as host",
			),
		)
	}

	if pd.StaticToken.IsUnknown() {
		diags = append(diags,
			diag.NewErrorDiagnostic(
				"Unknown static token",
				"Cannot use an unknown value as static token",
			),
		)
	}

	if pd.ServiceAccountPath.IsUnknown() {
		diags = append(diags,
			diag.NewErrorDiagnostic(
				"Unknown service account path",
				"Cannot use an unknown value as service account path",
			),
		)
	}

	if pd.ServiceAccountID.IsUnknown() {
		diags = append(diags,
			diag.NewErrorDiagnostic(
				"Unknown service account ID",
				"Cannot use an unknown value as service account ID",
			),
		)
	}

	if pd.ServiceAccountToken.IsUnknown() {
		diags = append(diags,
			diag.NewErrorDiagnostic(
				"Unknown service account token",
				"Cannot use an unknown value as service account token",
			),
		)
	}

	return diags
}

func (p *tharsisProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data providerData

	diags := req.Config.Get(ctx, &data)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	// No field in the provider can be unknown
	diags = data.checkUnknowns()
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	tClient, err := newTharsisClient(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error configuring the Tharsis client",
			fmt.Sprintf("Error configuring the Tharsis client, this is an error in the provider.\n%s\n", err),
		)
		return
	}

	p.client = tClient
	p.configured = true

	// Make the Tharsis client available during DataSource and Resource
	// type Configure methods.
	resp.DataSourceData = tClient
	resp.ResourceData = tClient

	tflog.Info(ctx, "Configured Tharsis client", map[string]any{"success": true})
}

func (p *tharsisProvider) Resources(context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewGPGKeyResource,
		NewGroupResource,
		NewManagedIdentityResource,
		NewManagedIdentityAliasResource,
		NewManagedIdentityAccessRuleResource,
		NewServiceAccountResource,
		NewTerraformModuleResource,
		NewTerraformProviderResource,
		NewVariableResource,
		NewVCSProviderResource,
		NewWorkspaceResource,
		NewApplyModuleResource,
		NewWorkspaceVCSProviderLinkResource,
		NewAssignedManagedIdentityResource,
	}
}

func (p *tharsisProvider) DataSources(context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{

		// tharsis_workspace_outputs, no JSON
		func() datasource.DataSource {
			return workspaceOutputsDataSource{
				provider:      *p,
				isJSONEncoded: false,
			}
		},

		// tharsis_workspace_outputs, with JSON
		func() datasource.DataSource {
			return workspaceOutputsDataSource{
				provider:      *p,
				isJSONEncoded: true,
			}
		},
	}
}

func newTharsisClient(ctx context.Context, pd *providerData) (*client.GRPCClient, error) {
	host := pd.Host.ValueString()
	if host == "" {
		host = os.Getenv("THARSIS_ENDPOINT")
	}

	if host == "" {
		return nil, fmt.Errorf("host cannot be an empty string")
	}

	// Prepend scheme if not already present, then validate.
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		host = scheme + host
	}

	if _, err := url.ParseRequestURI(host); err != nil {
		return nil, fmt.Errorf("invalid host URL %q: %w", host, err)
	}

	// Build the TokenConfig from provider data. Env vars (THARSIS_STATIC_TOKEN,
	// THARSIS_SERVICE_ACCOUNT_ID, etc.) are loaded automatically by TokenConfig.Resolve.
	tokenCfg := &token.Config{
		StaticToken: getTFTokenForHost(host), // TF_TOKEN_<host> is lowest priority.
	}

	// Provider config values seed the struct; env vars override in Resolve.
	if v := pd.StaticToken.ValueString(); v != "" {
		tokenCfg.StaticToken = v
	}

	if v := pd.ServiceAccountID.ValueString(); v != "" {
		tokenCfg.ServiceAccountID = v
	} else if v := pd.ServiceAccountPath.ValueString(); v != "" {
		tokenCfg.ServiceAccountPath = v
	}

	if v := pd.ServiceAccountToken.ValueString(); v != "" {
		tokenCfg.ServiceAccountToken = v
	}

	userAgent := client.BuildUserAgent("terraform-provider-tharsis", Version)
	tlsSkipVerify := pd.TLSSkipVerify.ValueBool()
	logger := &tflogAdapter{ctx: ctx}

	// Resolve the token.
	resolver, err := tokenCfg.Resolve(ctx, host, nil,
		token.WithTLSSkipVerify(tlsSkipVerify),
		token.WithLogger(logger),
		token.WithUserAgent(userAgent),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve authentication: %w", err)
	}

	// Create the gRPC client.
	grpcClient, err := client.NewGRPCClient(ctx, &client.GRPCClientConfig{
		HTTPEndpoint:  host,
		TokenResolver: resolver,
		TLSSkipVerify: tlsSkipVerify,
		UserAgent:     userAgent,
		Logger:        logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC client: %w", err)
	}

	return grpcClient, nil
}

func getTFTokenForHost(host string) string {
	if host == "" {
		// undefined host doesn't have a token
		return ""
	}

	uri, err := url.Parse(host)
	if err != nil {
		// can't provide a token if host can't be parsed
		return ""
	}

	hostname, err := svchost.ForComparison(uri.Host)
	if err != nil {
		// return an empty string if we can't compare
		return ""
	}

	return tfTokenEnvironmentVariables()[hostname]
}

// tfTokenEnvironmentVariables returns a map of valid hostnames and their token based on the `TF_TOKEN_` prefixed environment variables.
// This was copied from github.com/hashicorp/terraform-provider-tfe/tfe/credentials.go:collectCredentialsFromEnv with a license of MPL-2.0
func tfTokenEnvironmentVariables() map[svchost.Hostname]string {
	const prefix = "TF_TOKEN_"

	ret := make(map[svchost.Hostname]string)
	for _, ev := range os.Environ() {
		eqIdx := strings.Index(ev, "=")
		if eqIdx < 0 {
			continue
		}
		name := ev[:eqIdx]
		value := ev[eqIdx+1:]
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		rawHost := name[len(prefix):]

		// We accept double underscores in place of hyphens because hyphens are not valid
		// identifiers in most shells and are therefore hard to set.
		// This is unambiguous with replacing single underscores below because
		// hyphens are not allowed at the beginning or end of a label and therefore
		// odd numbers of underscores will not appear together in a valid variable name.
		rawHost = strings.ReplaceAll(rawHost, "__", "-")

		// We accept underscores in place of dots because dots are not valid
		// identifiers in most shells and are therefore hard to set.
		// Underscores are not valid in hostnames, so this is unambiguous for
		// valid hostnames.
		rawHost = strings.ReplaceAll(rawHost, "_", ".")

		// Because environment variables are often set indirectly by OS
		// libraries that might interfere with how they are encoded, we'll
		// be tolerant of them being given either directly as UTF-8 IDNs
		// or in Punycode form, normalizing to Punycode form here because
		// that is what the Terraform credentials helper protocol will
		// use in its requests.
		//
		// Using ForDisplay first here makes this more liberal than Terraform
		// itself would usually be in that it will tolerate pre-punycoded
		// hostnames that Terraform normally rejects in other contexts in order
		// to ensure stored hostnames are human-readable.
		dispHost := svchost.ForDisplay(rawHost)
		hostname, err := svchost.ForComparison(dispHost)
		if err != nil {
			// Ignore invalid hostnames
			continue
		}

		ret[hostname] = value
	}

	return ret
}
