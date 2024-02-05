package provider

import (
	"context"

	"github.com/aws/smithy-go/ptr"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	tharsis "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg"
	ttypes "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

// AssignedManagedIdentityModel is the model for an assigned managed identity.
type AssignedManagedIdentityModel struct {
	ID                types.String `tfsdk:"id"`
	ManagedIdentityID types.String `tfsdk:"managed_identity_id"`
	WorkspaceID       types.String `tfsdk:"workspace_id"`
}

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ resource.Resource                = (*assignedManagedIdentityResource)(nil)
	_ resource.ResourceWithConfigure   = (*assignedManagedIdentityResource)(nil)
	_ resource.ResourceWithImportState = (*assignedManagedIdentityResource)(nil)
)

// NewAssignedManagedIdentityResource is a helper function to simplify the provider implementation.
func NewAssignedManagedIdentityResource() resource.Resource {
	return &assignedManagedIdentityResource{}
}

type assignedManagedIdentityResource struct {
	client *tharsis.Client
}

// Metadata returns the full name of the resource, including prefix, underscore, instance name.
func (t *assignedManagedIdentityResource) Metadata(_ context.Context,
	_ resource.MetadataRequest, resp *resource.MetadataResponse,
) {
	resp.TypeName = "tharsis_assigned_managed_identity"
}

func (t *assignedManagedIdentityResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	description := "Defines and manages an assigned managed identity."

	resp.Schema = schema.Schema{
		Version:             1,
		MarkdownDescription: description,
		Description:         description,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "An ID for this tharsis_assigned_managed_identity resource.",
				Description:         "An ID for this tharsis_assigned_managed_identity resource.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(), // set once during create, kept in state thereafter
				},
			},
			"managed_identity_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the assigned managed identity.",
				Description:         "The ID of the assigned managed identity.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"workspace_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the assigned-to workspace.",
				Description:         "The ID of the assigned-to workspace.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

// Configure lets the provider implement the ResourceWithConfigure interface.
func (t *assignedManagedIdentityResource) Configure(_ context.Context,
	req resource.ConfigureRequest, _ *resource.ConfigureResponse,
) {
	if req.ProviderData == nil {
		return
	}
	t.client = req.ProviderData.(*tharsis.Client)
}

func (t *assignedManagedIdentityResource) Create(ctx context.Context,
	req resource.CreateRequest, resp *resource.CreateResponse,
) {
	// Retrieve values from assigned managed identity.
	var assignment AssignedManagedIdentityModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &assignment)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the workspace in order to have the path.
	workspace, err := t.client.Workspaces.GetWorkspace(ctx,
		&ttypes.GetWorkspaceInput{
			ID: ptr.String(assignment.WorkspaceID.ValueString()),
		},
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting workspace",
			err.Error(),
		)
		return
	}

	// Create the assigned managed identity. (In other words, assign the managed identity to the workspace.)
	managedIdentityID := assignment.ManagedIdentityID.ValueString()
	_, err = t.client.ManagedIdentity.AssignManagedIdentityToWorkspace(ctx,
		&ttypes.AssignManagedIdentityInput{
			ManagedIdentityID: &managedIdentityID,
			WorkspacePath:     workspace.FullPath,
		})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating assigned managed identity",
			err.Error(),
		)
		return
	}

	created := AssignedManagedIdentityModel{
		ID:                types.StringValue(uuid.New().String()), // computed with no input from any other resource
		ManagedIdentityID: types.StringValue(managedIdentityID),
		WorkspaceID:       types.StringValue(workspace.Metadata.ID),
	}

	// Map the response body to the schema.  (There are no computed values to update to the plan.)
	t.copyAssignedManagedIdentity(&created, &assignment)

	// Set the response state to the fully-populated plan, whether or not there is an error.
	resp.Diagnostics.Append(resp.State.Set(ctx, assignment)...)
}

func (t *assignedManagedIdentityResource) Read(ctx context.Context,
	req resource.ReadRequest, resp *resource.ReadResponse,
) {
	// Get the current state.
	var state AssignedManagedIdentityModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the workspace from Tharsis so we can have its ID.
	workspace, err := t.client.Workspaces.GetWorkspace(ctx,
		&ttypes.GetWorkspaceInput{
			ID: ptr.String(state.WorkspaceID.ValueString()),
		},
	)
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

	// Get the assigned managed identities from Tharsis.
	managedIdentities, err := t.client.Workspaces.GetAssignedManagedIdentities(ctx,
		&ttypes.GetAssignedManagedIdentitiesInput{
			ID: ptr.String(state.WorkspaceID.ValueString()),
		})
	if err != nil {
		if tharsis.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error reading assigned managed identities",
			err.Error(),
		)
		return
	}

	// Find the assigned managed identity by ID.
	wantID := state.ManagedIdentityID.ValueString()
	var found *AssignedManagedIdentityModel
	for _, candidate := range managedIdentities {
		if candidate.Metadata.ID == wantID {
			found = &AssignedManagedIdentityModel{
				ManagedIdentityID: types.StringValue(candidate.Metadata.ID),
				WorkspaceID:       types.StringValue(workspace.Metadata.ID),
			}
			break
		}
	}
	if found == nil {
		resp.State.RemoveResource(ctx)
		resp.Diagnostics.AddError(
			"Error finding assigned specified managed identity",
			"error finding assigned specified managed identity",
		)
		return
	}

	// Copy the from-Tharsis struct to the state.
	t.copyAssignedManagedIdentity(found, &state)

	// Set the refreshed state, whether or not there is an error.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (t *assignedManagedIdentityResource) Update(_ context.Context,
	_ resource.UpdateRequest, resp *resource.UpdateResponse,
) {
	// This method must exist to comply with the required interfaces,
	// but all input attributes have the RequiresReplace plan modifier,
	// so there's nothing for it to do.  It should never be called.
	// If it is, it should error out.

	resp.Diagnostics.AddError(
		"Error updating assigned managed identity.",
		"assigned managed identity should never be updated in place.",
	)
}

func (t *assignedManagedIdentityResource) Delete(ctx context.Context,
	req resource.DeleteRequest, resp *resource.DeleteResponse,
) {
	// Get the current state.
	var state AssignedManagedIdentityModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the workspace from Tharsis so we can have its path, required to unassign the managed identity.
	workspace, err := t.client.Workspaces.GetWorkspace(ctx,
		&ttypes.GetWorkspaceInput{
			ID: ptr.String(state.WorkspaceID.ValueString()),
		},
	)
	if err != nil {
		if tharsis.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error getting workspace",
			err.Error(),
		)
		return
	}

	// Delete the assigned managed identity via Tharsis.
	// In other words, unassign the managed identity from the workspace.
	_, err = t.client.ManagedIdentity.UnassignManagedIdentityFromWorkspace(ctx,
		&ttypes.AssignManagedIdentityInput{
			WorkspacePath:     workspace.FullPath,
			ManagedIdentityID: ptr.String(state.ManagedIdentityID.ValueString()),
		})
	if err != nil {

		// Handle the case that the assigned managed identity no longer exists.
		if tharsis.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error deleting assigned managed identity",
			err.Error(),
		)
	}
}

// ImportState helps the provider implement the ResourceWithImportState interface.
func (t *assignedManagedIdentityResource) ImportState(ctx context.Context,
	req resource.ImportStateRequest, resp *resource.ImportStateResponse,
) {
	// Retrieve import ID and save to id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// copyAssignedManagedIdentity copies the contents of an assigned managed identity.
// It is intended to copy from a struct returned by Tharsis to a Terraform plan or state.
func (t *assignedManagedIdentityResource) copyAssignedManagedIdentity(
	src, dest *AssignedManagedIdentityModel,
) {
	dest.ManagedIdentityID = src.ManagedIdentityID
	dest.WorkspaceID = src.WorkspaceID
}
