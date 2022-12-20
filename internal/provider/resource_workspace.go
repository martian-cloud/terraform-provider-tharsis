package provider

import (
	"context"
	"strings"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
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
	MaxJobDuration     types.Int64  `tfsdk:"max_job_duration"`
	TerraformVersion   types.String `tfsdk:"terraform_version"`
	PreventDestroyPlan types.Bool   `tfsdk:"prevent_destroy_plan"`
	LastUpdated        types.String `tfsdk:"last_updated"`
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
func (t *workspaceResource) Metadata(ctx context.Context,
	req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "tharsis_workspace"
}

// The diagnostics return value is required by the interface even though this function returns only nil.
func (t *workspaceResource) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	description := "Defines and manages a workspace."

	return tfsdk.Schema{
		Version: 1,

		MarkdownDescription: description,
		Description:         description,

		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:                types.StringType,
				MarkdownDescription: "String identifier of the workspace.",
				Description:         "String identifier of the workspace.",
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					resource.UseStateForUnknown(),
				},
			},
			"name": {
				Type:                types.StringType,
				MarkdownDescription: "The name of the workspace.",
				Description:         "The name of the workspace.",
				Required:            true,
			},
			"description": {
				Type:                types.StringType,
				MarkdownDescription: "A description of the workspace.",
				Description:         "A description of the workspace.",
				Required:            true,
			},
			"full_path": {
				Type:                types.StringType,
				MarkdownDescription: "The path of the parent namespace plus the name of the workspace.",
				Description:         "The path of the parent namespace plus the name of the workspace.",
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					resource.UseStateForUnknown(),
				},
			},
			"group_path": {
				Type:                types.StringType,
				MarkdownDescription: "Path of the parent group.",
				Description:         "Path of the parent group.",
				Required:            true,
			},
			"max_job_duration": {
				Type:                types.Int64Type,
				MarkdownDescription: "Maximum job duration in minutes.",
				Description:         "Maximum job duration in minutes.",
				Optional:            true,
				Computed:            true, // API sets a default value if not specified.
			},
			"terraform_version": {
				Type:                types.StringType,
				MarkdownDescription: "Terraform version for this workspace.",
				Description:         "Terraform version for this workspace.",
				Optional:            true,
				Computed:            true, // API sets a default value if not specified.
			},
			"prevent_destroy_plan": {
				Type:                types.BoolType,
				MarkdownDescription: "Whether a destroy plan would be prevented.",
				Description:         "Whether a destroy plan would be prevented.",
				Optional:            true,
				Computed:            true, // API sets a (arguably trivial) default value if not specified.
			},
			"last_updated": {
				Type:                types.StringType,
				MarkdownDescription: "Timestamp when this workspace was most recently updated.",
				Description:         "Timestamp when this workspace was most recently updated.",
				Computed:            true,
			},
		},
	}, nil
}

// Configure lets the provider implement the ResourceWithConfigure interface.
func (t *workspaceResource) Configure(_ context.Context,
	req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	t.client = req.ProviderData.(*tharsis.Client)
}

func (t *workspaceResource) Create(ctx context.Context,
	req resource.CreateRequest, resp *resource.CreateResponse) {

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
	req resource.ReadRequest, resp *resource.ReadResponse) {

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
		resp.Diagnostics.AddError(
			"Error reading workspace",
			err.Error(),
		)
		return
	}

	if found == nil {
		// Handle the case that the workspace no longer exists if that fact is reported by returning nil.
		resp.State.RemoveResource(ctx)
		return
	}

	// Copy the from-Tharsis struct to the state.
	t.copyWorkspace(*found, &state)

	// Set the refreshed state, whether or not there is an error.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (t *workspaceResource) Update(ctx context.Context,
	req resource.UpdateRequest, resp *resource.UpdateResponse) {

	// Retrieve values from plan.
	var plan WorkspaceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update the workspace via Tharsis.
	// The ID is used to find the record to update.
	// The description is modified.
	var workspacePath *string
	if plan.FullPath.ValueString() != "" {
		workspacePath = ptr.String(plan.FullPath.ValueString())
	}
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
			WorkspacePath:      workspacePath,
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
	req resource.DeleteRequest, resp *resource.DeleteResponse) {

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
		if t.isErrorWorkspaceNotFound(err) {
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
	req resource.ImportStateRequest, resp *resource.ImportStateResponse) {

	// Get the workspace by full path from Tharsis.
	found, err := t.client.Workspaces.GetWorkspace(ctx, &ttypes.GetWorkspaceInput{
		Path: &req.ID,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Import workspace not found: "+req.ID,
			err.Error(),
		)
		return
	}
	if found == nil {
		resp.Diagnostics.AddError(
			"Import workspace not found: "+req.ID,
			"",
		)
		return
	}

	// Import by full path.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), found.Metadata.ID)...)
}

// copyWorkspace copies the contents of a workspace.
// It is intended to copy from a struct returned by Tharsis to a Terraform plan or state.
func (t *workspaceResource) copyWorkspace(src ttypes.Workspace, dest *WorkspaceModel) error {
	dest.ID = types.StringValue(src.Metadata.ID)
	dest.Name = types.StringValue(src.Name)
	dest.Description = types.StringValue(src.Description)
	dest.FullPath = types.StringValue(src.FullPath)
	dest.GroupPath = types.StringValue(t.getParentPath(src.FullPath))
	dest.MaxJobDuration = types.Int64Value(int64(src.MaxJobDuration))
	dest.TerraformVersion = types.StringValue(src.TerraformVersion)
	dest.PreventDestroyPlan = types.BoolValue(src.PreventDestroyPlan)

	// Must use time value from SDK/API.  Using time.Now() is not reliable.
	dest.LastUpdated = types.StringValue(src.Metadata.LastUpdatedTimestamp.Format(time.RFC850))

	return nil
}

// getParentPath returns the parent path
func (t *workspaceResource) getParentPath(fullPath string) string {
	return fullPath[:strings.LastIndex(fullPath, "/")]
}

// isErrorWorkspaceNotFound returns true iff the error message is that a workspace was not found.
// In theory, we should never see a message that some other ID was not found.
func (t *workspaceResource) isErrorWorkspaceNotFound(e error) bool {
	lowerError := strings.ToLower(e.Error())
	return strings.Contains(lowerError, "workspace with id ") &&
		strings.Contains(lowerError, " not found")
}

// The End.
