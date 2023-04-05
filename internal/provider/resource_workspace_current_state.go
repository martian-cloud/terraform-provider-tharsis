package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	tharsis "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg"
	sdktypes "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

const (
	jobCompletionPollInterval = 5 * time.Second
)

var (
	applyRunComment = "terraform-provider-tharsis" // must be var, not const, to take address
)

// WorkspaceCurrentStateModel is the model for a workspace_current_state.
// Please note: Unlike many/most other resources, this model does not exist in the Tharsis API.
// The workspace path, module source, and module version uniquely identify this workspace_current_state.
type WorkspaceCurrentStateModel struct {
	WorkspacePath types.String `tfsdk:"workspace_path"`
	ModuleSource  types.String `tfsdk:"module_source"`
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
			"module_source": schema.StringAttribute{
				MarkdownDescription: "The source of the module, including the API hostname.",
				Description:         "The source of the module, including the API hostname.",
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

	// If teardown is not specified, make it false.
	workspaceCurrentState.setDefaultTeardown()

	if !workspaceCurrentState.Teardown.ValueBool() {
		t.doApplyOrDestroyRun(ctx, workspaceCurrentState, resp.Diagnostics)
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

	// If teardown is not specified, make it false.
	plan.setDefaultTeardown()

	// Apply or destroy, depending on the Teardown flag.
	t.doApplyOrDestroyRun(ctx, plan, resp.Diagnostics)

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

	// We know we're supposed to delete, but the current state's teardown is probably set to false.
	// Force teardown to true to make it do the delete operation.
	state.Teardown = types.BoolValue(true)

	t.doApplyOrDestroyRun(ctx, state, resp.Diagnostics)
}

// ImportState helps the provider implement the ResourceWithImportState interface.
func (t *workspaceCurrentStateResource) ImportState(ctx context.Context,
	req resource.ImportStateRequest, resp *resource.ImportStateResponse) {

	resp.Diagnostics.AddError(
		"Import of workspace_current_state is not supported.",
		"",
	)
}

func (t *workspaceCurrentStateResource) doApplyOrDestroyRun(ctx context.Context,
	model WorkspaceCurrentStateModel, diags diag.Diagnostics) {

	// Call CreateRun
	createdRun, err := t.client.Run.CreateRun(ctx, &sdktypes.CreateRunInput{
		WorkspacePath:          model.WorkspacePath.ValueString(),
		ConfigurationVersionID: nil, // using module registry path and version
		IsDestroy:              model.Teardown.ValueBool(),
		ModuleSource:           ptr.String(model.ModuleSource.ValueString()),
		ModuleVersion:          ptr.String(model.ModuleVersion.ValueString()),
		Variables:              []sdktypes.RunVariable{},
	})
	if err != nil {
		diags.AddError("Failed to create run", err.Error())
		return
	}

	if err = t.waitForJobCompletion(ctx, createdRun.Plan.CurrentJobID); err != nil {
		diags.AddError("Failed to wait for plan job completion", err.Error())
		return
	}

	plannedRun, err := t.client.Run.GetRun(ctx, &sdktypes.GetRunInput{ID: createdRun.Metadata.ID})
	if err != nil {
		diags.AddError("Failed to get planned run", err.Error())
		return
	}

	// If the plan fails, both plannedRun.Status and plannedRun.Plan.Status are "errored".
	// If the plan succeeds, plannedRun.Status is "planned",
	// while plannedRun.Plan.Status is "finished".
	//
	if !strings.HasPrefix(string(plannedRun.Status), "planned") {
		diags.AddError("Plan failed", string(plannedRun.Status))
		return
	}
	if plannedRun.Plan.Status != "finished" {
		diags.AddError("Plan failed", string(plannedRun.Plan.Status))
		return
	}

	// Do the apply run.
	appliedRun, err := t.client.Run.ApplyRun(ctx, &sdktypes.ApplyRunInput{
		RunID:   createdRun.Metadata.ID,
		Comment: &applyRunComment,
	})
	if err != nil {
		diags.AddError("Failed to apply a run", err.Error())
		return
	}

	// Make sure the run has an apply.
	if appliedRun.Apply == nil {
		msg := fmt.Sprintf("Created run does not have an apply: %s", appliedRun.Metadata.ID)
		diags.AddError(msg, "")
		return
	}

	if err = t.waitForJobCompletion(ctx, appliedRun.Apply.CurrentJobID); err != nil {
		diags.AddError("Failed to wait for apply job completion", err.Error())
		return
	}

	finishedRun, err := t.client.Run.GetRun(ctx, &sdktypes.GetRunInput{ID: appliedRun.Metadata.ID})
	if err != nil {
		diags.AddError("Failed to get finished run", err.Error())
		return
	}

	// If an apply job succeeds, finishedRun.Status is "applied" and
	// finishedRun.Apply.Status is "finished".
	if finishedRun.Status != "applied" {
		diags.AddError("Apply failed", string(finishedRun.Status))
		return
	}
	if finishedRun.Apply.Status != "finished" {
		diags.AddError("Apply status", string(finishedRun.Apply.Status))
		return
	}
}

func (t *workspaceCurrentStateResource) waitForJobCompletion(ctx context.Context, jobID *string) error {
	if jobID == nil {
		return fmt.Errorf("nil job ID")
	}

	// Poll until job has finished.
	for {

		job, err := t.client.Job.GetJob(ctx, &sdktypes.GetJobInput{
			ID: *jobID,
		})
		if err != nil {
			return fmt.Errorf("failed to get job ID %s", *jobID)
		}

		if job.Status == "finished" {
			return nil
		}

		time.Sleep(jobCompletionPollInterval)
	}

}

// The End.