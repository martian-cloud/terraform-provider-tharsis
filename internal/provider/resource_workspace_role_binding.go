package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/client"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// WorkspaceRoleBindingModel is the model for a workspace role binding.
type WorkspaceRoleBindingModel struct {
	ID          types.String `tfsdk:"id"`
	WorkspaceID types.String `tfsdk:"workspace_id"`
	RoleName    types.String `tfsdk:"role_name"`
}

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ resource.Resource                = (*workspaceRoleBindingResource)(nil)
	_ resource.ResourceWithConfigure   = (*workspaceRoleBindingResource)(nil)
	_ resource.ResourceWithImportState = (*workspaceRoleBindingResource)(nil)
)

// NewWorkspaceRoleBindingResource is a helper function to simplify the provider implementation.
func NewWorkspaceRoleBindingResource() resource.Resource {
	return &workspaceRoleBindingResource{}
}

type workspaceRoleBindingResource struct {
	client *client.GRPCClient
}

// Metadata returns the full name of the resource, including prefix, underscore, instance name.
func (t *workspaceRoleBindingResource) Metadata(_ context.Context,
	_ resource.MetadataRequest, resp *resource.MetadataResponse,
) {
	resp.TypeName = "tharsis_workspace_role_binding"
}

func (t *workspaceRoleBindingResource) Schema(_ context.Context, _ resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	description := "Defines and manages a workspace role binding. A workspace role binding confers a " +
		"role on a workspace: the role's permissions become available to the workspace's job caller at " +
		"the workspace's direct parent namespace."

	resp.Schema = schema.Schema{
		Version:             1,
		MarkdownDescription: description,
		Description:         description,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "String identifier of the workspace role binding.",
				Description:         "String identifier of the workspace role binding.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"workspace_id": schema.StringAttribute{
				MarkdownDescription: "The ID or TRN of the workspace this role binding applies to.",
				Description:         "The ID or TRN of the workspace this role binding applies to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"role_name": schema.StringAttribute{
				MarkdownDescription: "The name of the role bound to the workspace.",
				Description:         "The name of the role bound to the workspace.",
				Required:            true,
				// Can be updated in place, so no RequiresReplace plan modifier.
			},
		},
	}
}

// Configure lets the provider implement the ResourceWithConfigure interface.
func (t *workspaceRoleBindingResource) Configure(_ context.Context,
	req resource.ConfigureRequest, _ *resource.ConfigureResponse,
) {
	if req.ProviderData == nil {
		return
	}
	t.client = req.ProviderData.(*client.GRPCClient)
}

func (t *workspaceRoleBindingResource) Create(ctx context.Context,
	req resource.CreateRequest, resp *resource.CreateResponse,
) {
	// Retrieve values from workspace role binding.
	var binding WorkspaceRoleBindingModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &binding)...)
	if resp.Diagnostics.HasError() {
		return
	}

	roleID := trn.TypeRole.Build(binding.RoleName.ValueString())

	// Create the workspace role binding.
	created, err := t.client.WorkspacesClient.SetWorkspaceRoleBinding(ctx,
		&pb.SetWorkspaceRoleBindingRequest{
			WorkspaceId: binding.WorkspaceID.ValueString(),
			RoleId:      &roleID,
		})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating workspace role binding",
			err.Error(),
		)
		return
	}

	// Map the response body to the schema and update the plan with the computed attribute values.
	if err = t.copyWorkspaceRoleBinding(ctx, created, &binding); err != nil {
		resp.Diagnostics.AddError(
			"Error reading back the bound role",
			err.Error(),
		)
		return
	}

	// Set the response state to the fully-populated plan, whether or not there is an error.
	resp.Diagnostics.Append(resp.State.Set(ctx, binding)...)
}

func (t *workspaceRoleBindingResource) Read(ctx context.Context,
	req resource.ReadRequest, resp *resource.ReadResponse,
) {
	// Get the current state.
	var state WorkspaceRoleBindingModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the workspace role binding from Tharsis.
	found, err := t.client.WorkspacesClient.GetWorkspaceRoleBindingByWorkspaceID(ctx,
		&pb.GetWorkspaceRoleBindingByWorkspaceIDRequest{
			WorkspaceId: state.WorkspaceID.ValueString(),
		})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error reading workspace role binding",
			err.Error(),
		)
		return
	}

	// Copy the from-Tharsis struct to the state.
	if err = t.copyWorkspaceRoleBinding(ctx, found, &state); err != nil {
		resp.Diagnostics.AddError(
			"Error reading back the bound role",
			err.Error(),
		)
		return
	}

	// Set the refreshed state, whether or not there is an error.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (t *workspaceRoleBindingResource) Update(ctx context.Context,
	req resource.UpdateRequest, resp *resource.UpdateResponse,
) {
	// Retrieve values from plan.
	var plan WorkspaceRoleBindingModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	roleID := trn.TypeRole.Build(plan.RoleName.ValueString())

	// Update the workspace role binding via Tharsis.
	// The workspace ID is used to find the record to update; the role ID is changed.
	updated, err := t.client.WorkspacesClient.SetWorkspaceRoleBinding(ctx,
		&pb.SetWorkspaceRoleBindingRequest{
			WorkspaceId: plan.WorkspaceID.ValueString(),
			RoleId:      &roleID,
		})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating workspace role binding",
			err.Error(),
		)
		return
	}

	// Copy all fields returned by Tharsis back into the plan.
	if err = t.copyWorkspaceRoleBinding(ctx, updated, &plan); err != nil {
		resp.Diagnostics.AddError(
			"Error reading back the bound role",
			err.Error(),
		)
		return
	}

	// Set the response state to the fully-populated plan, with or without error.
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (t *workspaceRoleBindingResource) Delete(ctx context.Context,
	req resource.DeleteRequest, resp *resource.DeleteResponse,
) {
	// Get the current state.
	var state WorkspaceRoleBindingModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the workspace role binding via Tharsis.
	// Omitting role_id removes the workspace's existing binding.
	_, err := t.client.WorkspacesClient.SetWorkspaceRoleBinding(ctx,
		&pb.SetWorkspaceRoleBindingRequest{
			WorkspaceId: state.WorkspaceID.ValueString(),
		})
	if err != nil {

		// Handle the case that the workspace no longer exists.
		if status.Code(err) == codes.NotFound {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error deleting workspace role binding",
			err.Error(),
		)
	}
}

// ImportState helps the provider implement the ResourceWithImportState interface.
// Workspace role bindings are imported by workspace ID, since that's what identifies
// the binding from the Tharsis API's perspective.
func (t *workspaceRoleBindingResource) ImportState(ctx context.Context,
	req resource.ImportStateRequest, resp *resource.ImportStateResponse,
) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("workspace_id"), req.ID)...)
}

// copyWorkspaceRoleBinding copies the contents of a workspace role binding.
// It is intended to copy from a struct returned by Tharsis to a Terraform plan or state.
// The role is looked up by ID to resolve its name, since the API returns only the role's ID.
func (t *workspaceRoleBindingResource) copyWorkspaceRoleBinding(ctx context.Context, src *pb.WorkspaceRoleBinding,
	dest *WorkspaceRoleBindingModel,
) error {
	role, err := t.client.RolesClient.GetRoleByID(ctx, &pb.GetRoleByIDRequest{
		Id: src.RoleId,
	})
	if err != nil {
		return err
	}

	dest.ID = types.StringValue(src.Metadata.Id)
	dest.WorkspaceID = types.StringValue(src.WorkspaceId)
	dest.RoleName = types.StringValue(role.Name)

	return nil
}
