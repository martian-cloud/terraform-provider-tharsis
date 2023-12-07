package provider

import (
	"context"
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

// WorkspaceModel is the model for a workspace.
// Fields intentionally omitted: AssignedManagedIdentities, ManagedIdentities, ServiceAccounts,
// StateVersions, Memberships, Variables, ActivityEvents.
// Also for now, omitting DirtyState, Locked, CurrentStateVersionID, and CurrentJobID.
type WorkspaceModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	Description        types.String `tfsdk:"description"`
	FullPath           types.String `tfsdk:"full_path"`
	GroupPath          types.String `tfsdk:"group_path"`
	TerraformVersion   types.String `tfsdk:"terraform_version"`
	LastUpdated        types.String `tfsdk:"last_updated"`
	MaxJobDuration     types.Int64  `tfsdk:"max_job_duration"`
	PreventDestroyPlan types.Bool   `tfsdk:"prevent_destroy_plan"`
}

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ resource.Resource                = (*workspaceResource)(nil)
	_ resource.ResourceWithConfigure   = (*workspaceResource)(nil)
	_ resource.ResourceWithImportState = (*workspaceResource)(nil)
)

// NewWorkspaceResource is a helper function to simplify the provider implementation.
func NewWorkspaceResource() resource.Resource {
	return &workspaceResource{}
}

type workspaceResource struct {
	client *tharsis.Client
}

// Metadata returns the full name of the resource, including prefix, underscore, instance name.
func (t *workspaceResource) Metadata(_ context.Context,
	_ resource.MetadataRequest, resp *resource.MetadataResponse,
) {
	resp.TypeName = "tharsis_workspace"
}

func (t *workspaceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	description := "Defines and manages a workspace."

	resp.Schema = schema.Schema{
		Version:             1,
		MarkdownDescription: description,
		Description:         description,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "String identifier of the workspace.",
				Description:         "String identifier of the workspace.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the workspace.",
				Description:         "The name of the workspace.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A description of the workspace.",
				Description:         "A description of the workspace.",
				Required:            true,
				// Can be updated in place, so no RequiresReplace plan modifier.
			},
			"full_path": schema.StringAttribute{
				MarkdownDescription: "The path of the parent namespace plus the name of the workspace.",
				Description:         "The path of the parent namespace plus the name of the workspace.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"group_path": schema.StringAttribute{
				MarkdownDescription: "Path of the parent group.",
				Description:         "Path of the parent group.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"max_job_duration": schema.Int64Attribute{
				MarkdownDescription: "Maximum job duration in minutes.",
				Description:         "Maximum job duration in minutes.",
				Optional:            true,
				Computed:            true, // API sets a default value if not specified.
				// Can be updated in place, so no RequiresReplace plan modifier.
			},
			"terraform_version": schema.StringAttribute{
				MarkdownDescription: "Terraform version for this workspace.",
				Description:         "Terraform version for this workspace.",
				Optional:            true,
				Computed:            true, // API sets a default value if not specified.
				// Can be updated in place, so no RequiresReplace plan modifier.
			},
			"prevent_destroy_plan": schema.BoolAttribute{
				MarkdownDescription: "Whether a destroy plan would be prevented.",
				Description:         "Whether a destroy plan would be prevented.",
				Optional:            true,
				Computed:            true, // API sets a (arguably trivial) default value if not specified.
				// Can be updated in place, so no RequiresReplace plan modifier.
			},
			"last_updated": schema.StringAttribute{
				MarkdownDescription: "Timestamp when this workspace was most recently updated.",
				Description:         "Timestamp when this workspace was most recently updated.",
				Computed:            true,
			},
		},
	}
}

// Configure lets the provider implement the ResourceWithConfigure interface.
func (t *workspaceResource) Configure(_ context.Context,
	req resource.ConfigureRequest, _ *resource.ConfigureResponse,
) {
	if req.ProviderData == nil {
		return
	}
	t.client = req.ProviderData.(*tharsis.Client)
}

func (t *workspaceResource) Create(ctx context.Context,
	req resource.CreateRequest, resp *resource.CreateResponse,
) {
	// Retrieve values from workspace.
	var workspace WorkspaceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &workspace)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create the workspace.
	var maxJobDuration *int32
	if workspace.MaxJobDuration.ValueInt64() != 0 {
		maxJobDuration = ptr.Int32(int32(workspace.MaxJobDuration.ValueInt64()))
	}
	var terraformVersion *string
	if workspace.TerraformVersion.ValueString() != "" {
		terraformVersion = ptr.String(workspace.TerraformVersion.ValueString())
	}
	var preventDestroyPlan *bool
	if !(workspace.PreventDestroyPlan.IsUnknown() || workspace.PreventDestroyPlan.IsNull()) {
		preventDestroyPlan = ptr.Bool(workspace.PreventDestroyPlan.ValueBool())
	}
	created, err := t.client.Workspaces.CreateWorkspace(ctx,
		&ttypes.CreateWorkspaceInput{
			Name:               workspace.Name.ValueString(),
			Description:        workspace.Description.ValueString(),
			GroupPath:          workspace.GroupPath.ValueString(),
			MaxJobDuration:     maxJobDuration,
			TerraformVersion:   terraformVersion,
			PreventDestroyPlan: preventDestroyPlan,
		})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating workspace",
			err.Error(),
		)
		return
	}

	// Map the response body to the schema and update the plan with the computed attribute values.
	// Because the schema uses the Set type rather than the List type, make sure to set all fields.
	t.copyWorkspace(*created, &workspace)

	// Set the response state to the fully-populated plan, whether or not there is an error.
	resp.Diagnostics.Append(resp.State.Set(ctx, workspace)...)
}

func (t *workspaceResource) Read(ctx context.Context,
	req resource.ReadRequest, resp *resource.ReadResponse,
) {
	// Get the current state.
	var state WorkspaceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the workspace from Tharsis.
	found, err := t.client.Workspaces.GetWorkspace(ctx, &ttypes.GetWorkspaceInput{
		ID: ptr.String(state.ID.ValueString()),
	})
	if err != nil {
		if tharsis.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error reading workspace",
			err.Error(),
		)
		return
	}

	// Copy the from-Tharsis struct to the state.
	t.copyWorkspace(*found, &state)

	// Set the refreshed state, whether or not there is an error.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (t *workspaceResource) Update(ctx context.Context,
	req resource.UpdateRequest, resp *resource.UpdateResponse,
) {
	// Retrieve values from plan.
	var plan WorkspaceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update the workspace via Tharsis.
	// The ID is used to find the record to update.
	// The other fields are modified.
	var maxJobDuration *int32
	if plan.MaxJobDuration.ValueInt64() != 0 {
		maxJobDuration = ptr.Int32(int32(plan.MaxJobDuration.ValueInt64()))
	}
	var terraformVersion *string
	if plan.TerraformVersion.ValueString() != "" {
		terraformVersion = ptr.String(plan.TerraformVersion.ValueString())
	}
	var preventDestroyPlan *bool
	if !(plan.PreventDestroyPlan.IsUnknown() || plan.PreventDestroyPlan.IsNull()) {
		preventDestroyPlan = ptr.Bool(plan.PreventDestroyPlan.ValueBool())
	}
	updated, err := t.client.Workspaces.UpdateWorkspace(ctx,
		&ttypes.UpdateWorkspaceInput{
			ID:                 ptr.String(plan.ID.ValueString()),
			Description:        plan.Description.ValueString(),
			MaxJobDuration:     maxJobDuration,
			TerraformVersion:   terraformVersion,
			PreventDestroyPlan: preventDestroyPlan,
		})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating workspace",
			err.Error(),
		)
		return
	}

	// Copy all fields returned by Tharsis back into the plan.
	t.copyWorkspace(*updated, &plan)

	// Set the response state to the fully-populated plan, with or without error.
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (t *workspaceResource) Delete(ctx context.Context,
	req resource.DeleteRequest, resp *resource.DeleteResponse,
) {
	// Get the current state.
	var state WorkspaceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the workspace via Tharsis.
	err := t.client.Workspaces.DeleteWorkspace(ctx,
		&ttypes.DeleteWorkspaceInput{
			ID: ptr.String(state.ID.ValueString()),
		})
	if err != nil {

		// Handle the case that the workspace no longer exists.
		if tharsis.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error deleting workspace",
			err.Error(),
		)
	}
}

// ImportState helps the provider implement the ResourceWithImportState interface.
func (t *workspaceResource) ImportState(ctx context.Context,
	req resource.ImportStateRequest, resp *resource.ImportStateResponse,
) {
	// Get the workspace by full path from Tharsis.
	found, err := t.client.Workspaces.GetWorkspace(ctx, &ttypes.GetWorkspaceInput{
		Path: &req.ID,
	})
	if err != nil {
		if tharsis.IsNotFoundError(err) {
			resp.Diagnostics.AddError(
				"Import workspace not found: "+req.ID,
				"",
			)
			return
		}

		resp.Diagnostics.AddError(
			"Import workspace not found: "+req.ID,
			err.Error(),
		)
		return
	}

	// Import by full path.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), found.Metadata.ID)...)
}

// copyWorkspace copies the contents of a workspace.
// It is intended to copy from a struct returned by Tharsis to a Terraform plan or state.
func (t *workspaceResource) copyWorkspace(src ttypes.Workspace, dest *WorkspaceModel) {
	dest.ID = types.StringValue(src.Metadata.ID)
	dest.Name = types.StringValue(src.Name)
	dest.Description = types.StringValue(src.Description)
	dest.FullPath = types.StringValue(src.FullPath)
	dest.GroupPath = types.StringValue(src.GroupPath)
	dest.MaxJobDuration = types.Int64Value(int64(src.MaxJobDuration))
	dest.TerraformVersion = types.StringValue(src.TerraformVersion)
	dest.PreventDestroyPlan = types.BoolValue(src.PreventDestroyPlan)

	// Must use time value from SDK/API.  Using time.Now() is not reliable.
	dest.LastUpdated = types.StringValue(src.Metadata.LastUpdatedTimestamp.Format(time.RFC850))
}
