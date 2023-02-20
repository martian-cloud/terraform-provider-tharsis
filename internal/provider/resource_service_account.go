package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	tharsis "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg"
	ttypes "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

// OIDCTrustPolicyModel is the model for a trust policy.
type OIDCTrustPolicyModel struct {
	Issuer      types.String            `tfsdk:"issuer"`
	BoundClaims map[string]types.String `tfsdk:"bound_claims"`
}

// ServiceAccountModel is the model for a service account.
// Fields intentionally omitted: NamespaceMemberships and ActivityEvents.
type ServiceAccountModel struct {
	ID                types.String           `tfsdk:"id"`
	ResourcePath      types.String           `tfsdk:"resource_path"`
	Name              types.String           `tfsdk:"name"`
	Description       types.String           `tfsdk:"description"`
	GroupPath         types.String           `tfsdk:"group_path"`
	OIDCTrustPolicies []OIDCTrustPolicyModel `tfsdk:"oidc_trust_policies"`
}

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ resource.Resource                = (*serviceAccountResource)(nil)
	_ resource.ResourceWithConfigure   = (*serviceAccountResource)(nil)
	_ resource.ResourceWithImportState = (*serviceAccountResource)(nil)
)

// NewServiceAccountResource is a helper function to simplify the provider implementation.
func NewServiceAccountResource() resource.Resource {
	return &serviceAccountResource{}
}

type serviceAccountResource struct {
	client *tharsis.Client
}

// Metadata returns the full name of the resource, including prefix, underscore, instance name.
func (t *serviceAccountResource) Metadata(ctx context.Context,
	req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "tharsis_service_account"
}

func (t *serviceAccountResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	description := "Defines and manages a service account."

	resp.Schema = schema.Schema{
		Version:             1,
		MarkdownDescription: description,
		Description:         description,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "String identifier of the service account.",
				Description:         "String identifier of the service account.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"resource_path": schema.StringAttribute{
				MarkdownDescription: "The path of the parent namespace plus the name of the service account.",
				Description:         "The path of the parent namespace plus the name of the service account.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the service account.",
				Description:         "The name of the service account.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A description of the service account.",
				Description:         "A description of the service account.",
				Required:            true,
			},
			"group_path": schema.StringAttribute{
				MarkdownDescription: "Path of the parent group.",
				Description:         "Path of the parent group.",
				Required:            true,
			},
			"oidc_trust_policies": schema.ListNestedAttribute{
				MarkdownDescription: "OIDC trust policies for this service account.",
				Description:         "OIDC trust policies for this service account.",
				Required:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"bound_claims": schema.MapAttribute{
							ElementType:         types.StringType,
							MarkdownDescription: "Bound claims for this trust policy.",
							Description:         "Bound claims for this trust policy.",
							Required:            true,
						},
						"issuer": schema.StringAttribute{
							MarkdownDescription: "Issuer for this trust policy.",
							Description:         "Issuer for this trust policy.",
							Required:            true,
						},
					},
				},
			},
		},
	}
}

// Configure lets the provider implement the ResourceWithConfigure interface.
func (t *serviceAccountResource) Configure(_ context.Context,
	req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	t.client = req.ProviderData.(*tharsis.Client)
}

func (t *serviceAccountResource) Create(ctx context.Context,
	req resource.CreateRequest, resp *resource.CreateResponse) {

	// Retrieve values from service account.
	var serviceAccount ServiceAccountModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &serviceAccount)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create the service account.
	created, err := t.client.ServiceAccount.CreateServiceAccount(ctx,
		&ttypes.CreateServiceAccountInput{
			Name:              serviceAccount.Name.ValueString(),
			Description:       serviceAccount.Description.ValueString(),
			GroupPath:         serviceAccount.GroupPath.ValueString(),
			OIDCTrustPolicies: t.copyTrustPoliciesToInput(serviceAccount.OIDCTrustPolicies),
		})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating service account",
			err.Error(),
		)
		return
	}

	// Map the response body to the schema and update the plan with the computed attribute values.
	// Because the schema uses the Set type rather than the List type, make sure to set all fields.
	t.copyServiceAccount(*created, &serviceAccount)

	// Set the response state to the fully-populated plan, whether or not there is an error.
	resp.Diagnostics.Append(resp.State.Set(ctx, serviceAccount)...)
}

func (t *serviceAccountResource) Read(ctx context.Context,
	req resource.ReadRequest, resp *resource.ReadResponse) {

	// Get the current state.
	var state ServiceAccountModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the service account from Tharsis.
	found, err := t.client.ServiceAccount.GetServiceAccount(ctx, &ttypes.GetServiceAccountInput{
		ID: state.ID.ValueString(),
	})
	if err != nil {
		if tharsis.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error reading service account",
			err.Error(),
		)
		return
	}

	// Copy the from-Tharsis struct to the state.
	t.copyServiceAccount(*found, &state)

	// Set the refreshed state, whether or not there is an error.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (t *serviceAccountResource) Update(ctx context.Context,
	req resource.UpdateRequest, resp *resource.UpdateResponse) {

	// Retrieve values from plan.
	var plan ServiceAccountModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update the service account via Tharsis.
	// The ID is used to find the record to update.
	// The description is modified.
	updated, err := t.client.ServiceAccount.UpdateServiceAccount(ctx,
		&ttypes.UpdateServiceAccountInput{
			ID:                plan.ID.ValueString(),
			Description:       plan.Description.ValueString(),
			OIDCTrustPolicies: t.copyTrustPoliciesToInput(plan.OIDCTrustPolicies),
		})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating service account",
			err.Error(),
		)
		return
	}

	// Copy all fields returned by Tharsis back into the plan.
	t.copyServiceAccount(*updated, &plan)

	// Set the response state to the fully-populated plan, with or without error.
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (t *serviceAccountResource) Delete(ctx context.Context,
	req resource.DeleteRequest, resp *resource.DeleteResponse) {

	// Get the current state.
	var state ServiceAccountModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the service account via Tharsis.
	err := t.client.ServiceAccount.DeleteServiceAccount(ctx,
		&ttypes.DeleteServiceAccountInput{
			ID: state.ID.ValueString(),
		})
	if err != nil {

		// Handle the case that the service account no longer exists.
		if tharsis.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error deleting service account",
			err.Error(),
		)
	}
}

// ImportState helps the provider implement the ResourceWithImportState interface.
func (t *serviceAccountResource) ImportState(ctx context.Context,
	req resource.ImportStateRequest, resp *resource.ImportStateResponse) {

	// Retrieve import ID and save to id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// copyServiceAccount copies the contents of a service account.
// It is intended to copy from a struct returned by Tharsis to a Terraform plan or state.
func (t *serviceAccountResource) copyServiceAccount(src ttypes.ServiceAccount, dest *ServiceAccountModel) {
	dest.ID = types.StringValue(src.Metadata.ID)
	dest.ResourcePath = types.StringValue(src.ResourcePath)
	dest.Name = types.StringValue(src.Name)
	dest.Description = types.StringValue(src.Description)
	dest.GroupPath = types.StringValue(t.getParentPath(src.ResourcePath))

	newPolicies := []OIDCTrustPolicyModel{}
	for _, trustPolicy := range src.OIDCTrustPolicies {
		newPolicy := OIDCTrustPolicyModel{
			BoundClaims: make(map[string]types.String),
			Issuer:      types.StringValue(trustPolicy.Issuer),
		}
		for boundClaimKey, boundClaimValue := range trustPolicy.BoundClaims {
			newPolicy.BoundClaims[boundClaimKey] = types.StringValue(boundClaimValue)
		}
		newPolicies = append(newPolicies, newPolicy)
	}
	dest.OIDCTrustPolicies = newPolicies
}

// copyTrustPoliciesToInput copies a slice of OIDCTrustPolicyModel to a slice of ttypes.OIDCTrustPolicyInput.
func (t *serviceAccountResource) copyTrustPoliciesToInput(models []OIDCTrustPolicyModel) []ttypes.OIDCTrustPolicy {
	result := []ttypes.OIDCTrustPolicy{}

	for _, model := range models {
		boundClaims := map[string]string{}
		for k, v := range model.BoundClaims {
			boundClaims[k] = v.ValueString()
		}
		result = append(result, ttypes.OIDCTrustPolicy{
			Issuer:      model.Issuer.ValueString(),
			BoundClaims: boundClaims,
		})
	}

	// Terraform generally wants to see nil rather than an empty list.
	// However, this is likely to be moot, because Tharsis does not allow a service account with zero trust policies.
	if len(result) == 0 {
		result = nil
	}

	return result
}

// getParentPath returns the parent path
func (t *serviceAccountResource) getParentPath(fullPath string) string {
	return fullPath[:strings.LastIndex(fullPath, "/")]
}

// The End.
