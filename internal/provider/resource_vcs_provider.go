package provider

import (
	"context"
	"strings"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	tharsis "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg"
	ttypes "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

// VCSProviderModel is the model for a VCS provider.
type VCSProviderModel struct {
	ID                 types.String `tfsdk:"id"`
	LastUpdated        types.String `tfsdk:"last_updated"`
	Name               types.String `tfsdk:"name"`
	CreatedBy          types.String `tfsdk:"created_by"`
	Description        types.String `tfsdk:"description"`
	Hostname           types.String `tfsdk:"hostname"`
	GroupPath          types.String `tfsdk:"group_path"`
	ResourcePath       types.String `tfsdk:"resource_path"`
	Type               types.String `tfsdk:"type"`
	AutoCreateWebhooks types.Bool   `tfsdk:"auto_create_webhooks"`
}

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ resource.Resource                = (*vcsProviderResource)(nil)
	_ resource.ResourceWithConfigure   = (*vcsProviderResource)(nil)
	_ resource.ResourceWithImportState = (*vcsProviderResource)(nil)
)

// NewVCSProviderResource is a helper function to simplify the provider implementation.
func NewVCSProviderResource() resource.Resource {
	return &vcsProviderResource{}
}

type vcsProviderResource struct {
	client *tharsis.Client
}

// Metadata returns the full name of the resource, including prefix, underscore, instance name.
func (t *vcsProviderResource) Metadata(ctx context.Context,
	req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "tharsis_vcs_provider"
}

func (t *vcsProviderResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	description := "Defines and manages a VCS provider."

	resp.Schema = schema.Schema{
		Version:             1,
		MarkdownDescription: description,
		Description:         description,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "String identifier of the VCS provider.",
				Description:         "String identifier of the VCS provider.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the VCS provider.",
				Description:         "The name of the VCS provider.",
				Required:            true,
			},
			"created_by": schema.StringAttribute{
				MarkdownDescription: "The email address of the user or account that created this VCS provider.",
				Description:         "The email address of the user or account that created this VCS provider.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A description of the VCS provider.",
				Description:         "A description of the VCS provider.",
				Required:            true,
			},
			"hostname": schema.StringAttribute{
				MarkdownDescription: "Hostname for this VCS provider.",
				Description:         "Hostname for this VCS provider.",
				Optional:            true,
				Computed:            true, // API sets a default value if not specified.
			},
			"group_path": schema.StringAttribute{
				MarkdownDescription: "The path of the group where this VCS provider resides.",
				Description:         "The path of the group where this VCS provider resides.",
				Required:            true,
			},
			"resource_path": schema.StringAttribute{
				MarkdownDescription: "The path within the Tharsis group hierarchy to this VCS provider.",
				Description:         "The path within the Tharsis group hierarchy to this VCS provider.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The type of this VCS provider: gitlab, github, etc.",
				Description:         "The type of this VCS provider: gitlab, github, etc.",
				Required:            true,
			},
			"auto_create_webhooks": schema.BoolAttribute{
				MarkdownDescription: "Whether to automatically create webhooks.",
				Description:         "Whether to automatically create webhooks.",
				Required:            true,
			},
			"last_updated": schema.StringAttribute{
				MarkdownDescription: "Timestamp when this VCS provider was most recently updated.",
				Description:         "Timestamp when this VCS provider was most recently updated.",
				Computed:            true,
			},
		},
	}
}

// Configure lets the provider implement the ResourceWithConfigure interface.
func (t *vcsProviderResource) Configure(_ context.Context,
	req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	t.client = req.ProviderData.(*tharsis.Client)
}

func (t *vcsProviderResource) Create(ctx context.Context,
	req resource.CreateRequest, resp *resource.CreateResponse) {

	// Retrieve values from VCS provider.
	var vcsProvider VCSProviderModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &vcsProvider)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// FIXME: OAuthClient{ID,Secret} are required inputs to the GraphQL mutation.
	// However, they are not returned by the GraphQL mutation.
	// Where/how should this method get them?

	// Create the VCS provider.
	created, err := t.client.VCSProvider.CreateProvider(ctx,
		&ttypes.CreateVCSProviderInput{
			Name:               vcsProvider.Name.ValueString(),
			Description:        vcsProvider.Description.ValueString(),
			GroupPath:          vcsProvider.GroupPath.ValueString(),
			Hostname:           ptr.String(vcsProvider.Hostname.ValueString()),
			OAuthClientID:      "?????", // FIXME: vcsProvider.something.ValueString(),
			OAuthClientSecret:  "?????", // FIXME: vcsProvider.something.ValueString(),
			Type:               ttypes.VCSProviderType(vcsProvider.Type.ValueString()),
			AutoCreateWebhooks: vcsProvider.AutoCreateWebhooks.ValueBool(),
		})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating VCS provider",
			err.Error(),
		)
		return
	}

	// Map the response body to the schema and update the plan with the computed attribute values.
	t.copyVCSProvider(*created, &vcsProvider)

	// Set the response state to the fully-populated plan, whether or not there is an error.
	resp.Diagnostics.Append(resp.State.Set(ctx, vcsProvider)...)
}

func (t *vcsProviderResource) Read(ctx context.Context,
	req resource.ReadRequest, resp *resource.ReadResponse) {

	// Get the current state.
	var state VCSProviderModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the VCS provider from Tharsis.
	found, err := t.client.VCSProvider.GetProvider(ctx, &ttypes.GetVCSProviderInput{
		ID: state.ID.ValueString(),
	})
	if err != nil {
		if tharsis.NotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error reading VCS provider",
			err.Error(),
		)
		return
	}

	// Copy the from-Tharsis struct to the state.
	t.copyVCSProvider(*found, &state)

	// Set the refreshed state, whether or not there is an error.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (t *vcsProviderResource) Update(ctx context.Context,
	req resource.UpdateRequest, resp *resource.UpdateResponse) {

	// Retrieve values from plan.
	var plan VCSProviderModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// FIXME: OAuthClient{ID,Secret} are optional inputs to the GraphQL mutation.
	// However, they are not returned by the GraphQL mutation.
	// Where/how should this method get them?

	// Update the VCS provider via Tharsis.
	// The ID is used to find the record to update.
	updated, err := t.client.VCSProvider.UpdateProvider(ctx,
		&ttypes.UpdateVCSProviderInput{
			ID:                plan.ID.ValueString(),
			Description:       ptr.String(plan.Description.ValueString()),
			OAuthClientID:     ptr.String("?????"), // FIXME: plan.something.ValueString(),
			OAuthClientSecret: ptr.String("?????"), // FIXME: plan.something.ValueString(),
		})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating VCS provider",
			err.Error(),
		)
		return
	}

	// Copy all fields returned by Tharsis back into the plan.
	t.copyVCSProvider(*updated, &plan)

	// Set the response state to the fully-populated plan, with or without error.
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (t *vcsProviderResource) Delete(ctx context.Context,
	req resource.DeleteRequest, resp *resource.DeleteResponse) {

	// Get the current state.
	var state VCSProviderModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the VCS provider via Tharsis.
	_, err := t.client.VCSProvider.DeleteProvider(ctx,
		&ttypes.DeleteVCSProviderInput{
			ID: state.ID.ValueString(),
		})
	if err != nil {

		// Handle the case that the VCS provider no longer exists.
		if tharsis.NotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error deleting VCS provider",
			err.Error(),
		)
	}
}

// ImportState helps the provider implement the ResourceWithImportState interface.
func (t *vcsProviderResource) ImportState(ctx context.Context,
	req resource.ImportStateRequest, resp *resource.ImportStateResponse) {

	// Get the VCS provider by full path from Tharsis.
	found, err := t.client.VCSProvider.GetProvider(ctx, &ttypes.GetVCSProviderInput{
		ID: req.ID,
	})
	if err != nil {
		if tharsis.NotFoundError(err) {
			resp.Diagnostics.AddError(
				"Import VCS provider not found: "+req.ID,
				"",
			)
			return
		}

		resp.Diagnostics.AddError(
			"Import VCS provider not found: "+req.ID,
			err.Error(),
		)
		return
	}

	// Import by full path.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), found.Metadata.ID)...)
}

// copyVCSProvider copies the contents of a VCS provider.
// It is intended to copy from a struct returned by Tharsis to a Terraform plan or state.
func (t *vcsProviderResource) copyVCSProvider(src ttypes.VCSProvider, dest *VCSProviderModel) {
	dest.ID = types.StringValue(src.Metadata.ID)
	dest.Name = types.StringValue(src.Name)
	dest.CreatedBy = types.StringValue(src.CreatedBy)
	dest.Description = types.StringValue(src.Description)
	dest.Hostname = types.StringValue(src.Hostname)
	dest.GroupPath = types.StringValue(t.getParentPath(src.ResourcePath))
	dest.ResourcePath = types.StringValue(src.ResourcePath)
	dest.Type = types.StringValue(string(src.Type))
	dest.AutoCreateWebhooks = types.BoolValue(src.AutoCreateWebhooks)

	// Must use time value from SDK/API.  Using time.Now() is not reliable.
	dest.LastUpdated = types.StringValue(src.Metadata.LastUpdatedTimestamp.Format(time.RFC850))
}

// getParentPath returns the parent path
func (t *vcsProviderResource) getParentPath(fullPath string) string {
	return fullPath[:strings.LastIndex(fullPath, "/")]
}

// The End.
