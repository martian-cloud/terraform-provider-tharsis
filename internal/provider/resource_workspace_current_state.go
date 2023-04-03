package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	tharsis "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg"
)

// WorkspaceCurrentStateModel is the model for a workspace_current_state.
// Please note: Unlike many/most other resources, this model does not exist in the Tharsis API.
// The workspace path, module path, and module version uniquely identify this workspace_current_state.
type WorkspaceCurrentStateModel struct {
	WorkspacePath types.String `tfsdk:"full_path"`
	ModulePath    types.String `tfsdk:"module_path"`
	ModuleVersion types.String `tfsdk:"module_version"`
	Teardown      types.Bool   `tfsdk:"teardown"`
}

// Set the Teardown field's default value of false.
func (m *WorkspaceCurrentStateModel) setDefaultTeardown() {
	if m.Teardown.IsNull() || m.Teardown.IsUnknown() {
		m.Teardown = types.BoolValue(false)
	}
}

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ resource.Resource                = (*workspaceCurrentStateResource)(nil)
	_ resource.ResourceWithConfigure   = (*workspaceCurrentStateResource)(nil)
	_ resource.ResourceWithImportState = (*workspaceCurrentStateResource)(nil)
)

// NewWorkspaceCurrentStateResource is a helper function to simplify the provider implementation.
func NewWorkspaceCurrentStateResource() resource.Resource {
	return &workspaceCurrentStateResource{}
}

type workspaceCurrentStateResource struct {
	client *tharsis.Client
}

// Metadata returns the full name of the resource, including prefix, underscore, instance name.
func (t *workspaceCurrentStateResource) Metadata(ctx context.Context,
	req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "tharsis_workspace_current_state"
}

func (t *workspaceCurrentStateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	description := "Defines and manages a workspace current state."

	resp.Schema = schema.Schema{
		Version:             1,
		MarkdownDescription: description,
		Description:         description,
		Attributes: map[string]schema.Attribute{
			"workspace_path": schema.StringAttribute{
				MarkdownDescription: "The full path of the workspace.",
				Description:         "The full path of the workspace.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"module_path": schema.StringAttribute{
				MarkdownDescription: "The resource path of the module.",
				Description:         "The resource path of the module.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"module_version": schema.StringAttribute{
				MarkdownDescription: "The version identifier of the module.",
				Description:         "The version identifier of the module.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"teardown": schema.BoolAttribute{
				MarkdownDescription: "Whether to teardown the deployment of the module to the workspace.",
				Description:         "Whether to teardown the deployment of the module to the workspace.",
				Optional:            true,
				Computed:            true, // API sets a default value of false if not specified.
				// Can be updated in place, so no RequiresReplace plan modifier.
			},
		},
	}
}

// Configure lets the provider implement the ResourceWithConfigure interface.
func (t *workspaceCurrentStateResource) Configure(_ context.Context,
	req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	t.client = req.ProviderData.(*tharsis.Client)
}

func (t *workspaceCurrentStateResource) Create(ctx context.Context,
	req resource.CreateRequest, resp *resource.CreateResponse) {

	// Retrieve values from workspace current state.
	var workspaceCurrentState WorkspaceCurrentStateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &workspaceCurrentState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	workspaceCurrentState.setDefaultTeardown()

	if !workspaceCurrentState.Teardown.ValueBool() {

		// FIXME: Do an apply run.

	}

	// Set the response state to the fully-populated plan, whether or not there is an error.
	resp.Diagnostics.Append(resp.State.Set(ctx, workspaceCurrentState)...)
}

func (t *workspaceCurrentStateResource) Read(ctx context.Context,
	req resource.ReadRequest, resp *resource.ReadResponse) {

	// Get the current state.
	var state WorkspaceCurrentStateModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// No need to set Teardown's default value.

	// There is no model in the Tharsis API, so there's nothing really to read.

	// Set the refreshed state, whether or not there is an error.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (t *workspaceCurrentStateResource) Update(ctx context.Context,
	req resource.UpdateRequest, resp *resource.UpdateResponse) {

	// Retrieve values from plan.
	var plan WorkspaceCurrentStateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.setDefaultTeardown()

	// FIXME: Do an apply or destroy run.

	// Set the response state to the fully-populated plan, with or without error.
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (t *workspaceCurrentStateResource) Delete(ctx context.Context,
	req resource.DeleteRequest, resp *resource.DeleteResponse) {

	// Get the current state.
	var state WorkspaceCurrentStateModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// FIXME: Do a destroy run.

}

// ImportState helps the provider implement the ResourceWithImportState interface.
func (t *workspaceCurrentStateResource) ImportState(ctx context.Context,
	req resource.ImportStateRequest, resp *resource.ImportStateResponse) {

	resp.Diagnostics.AddError(
		"Import of workspace_current_state is not supported.",
		"",
	)
}

// The End.
