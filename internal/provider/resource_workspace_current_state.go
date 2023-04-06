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
					stringplanmodifier.RequiresReplace(),
				},
			},
			"module_source": schema.StringAttribute{
				MarkdownDescription: "The source of the module, including the API hostname.",
				Description:         "The source of the module, including the API hostname.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"module_version": schema.StringAttribute{
				MarkdownDescription: "The version identifier of the module.",
				Description:         "The version identifier of the module.",
				Optional:            true,
				Computed:            true, // computed if not supplied
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
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

	created := t.doApplyOrDestroyRun(ctx, workspaceCurrentState, false, resp.Diagnostics)

	// Map the response body to the schema and update the plan with the computed attribute values.
	t.copyWorkspaceCurrentState(created, &workspaceCurrentState)

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

	// FIXME: See other review items to do the necessary things here.

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

	// FIXME: See other review items to set this correctly.
	isDestroyRun := false

	// Apply or destroy, depending on the isDestroyRun argument.
	updated := t.doApplyOrDestroyRun(ctx, plan, isDestroyRun, resp.Diagnostics)

	// Copy all fields returned by Tharsis back into the plan.
	t.copyWorkspaceCurrentState(updated, &plan)

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

	// The workspace current state is being deleted, so don't use the returned value.
	_ = t.doApplyOrDestroyRun(ctx, state, true, resp.Diagnostics)
}

// ImportState helps the provider implement the ResourceWithImportState interface.
func (t *workspaceCurrentStateResource) ImportState(ctx context.Context,
	req resource.ImportStateRequest, resp *resource.ImportStateResponse) {

	resp.Diagnostics.AddError(
		"Import of workspace_current_state is not supported.",
		"",
	)
}

// Because there is no Tharsis-defined struct for a workspace current state resource, return this module's struct.
func (t *workspaceCurrentStateResource) doApplyOrDestroyRun(ctx context.Context,
	model WorkspaceCurrentStateModel, isDestroy bool, diags diag.Diagnostics,
) *WorkspaceCurrentStateModel {

	// Call CreateRun
	createdRun, err := t.client.Run.CreateRun(ctx, &sdktypes.CreateRunInput{
		WorkspacePath:          model.WorkspacePath.ValueString(),
		ConfigurationVersionID: nil, // using module registry path and version
		IsDestroy:              isDestroy,
		ModuleSource:           ptr.String(model.ModuleSource.ValueString()),
		ModuleVersion:          ptr.String(model.ModuleVersion.ValueString()),
		Variables:              []sdktypes.RunVariable{},
	})
	if err != nil {
		diags.AddError("Failed to create run", err.Error())
		return nil
	}

	if err = t.waitForJobCompletion(ctx, createdRun.Plan.CurrentJobID); err != nil {
		diags.AddError("Failed to wait for plan job completion", err.Error())
		return nil
	}

	plannedRun, err := t.client.Run.GetRun(ctx, &sdktypes.GetRunInput{ID: createdRun.Metadata.ID})
	if err != nil {
		diags.AddError("Failed to get planned run", err.Error())
		return nil
	}

	// If the plan fails, both plannedRun.Status and plannedRun.Plan.Status are "errored".
	// If the plan succeeds, plannedRun.Status is "planned",
	// while plannedRun.Plan.Status is "finished".
	//
	if !strings.HasPrefix(string(plannedRun.Status), "planned") {
		diags.AddError("Plan failed", string(plannedRun.Status))
		return nil
	}
	if plannedRun.Plan.Status != "finished" {
		diags.AddError("Plan failed", string(plannedRun.Plan.Status))
		return nil
	}

	// Do the apply run.
	appliedRun, err := t.client.Run.ApplyRun(ctx, &sdktypes.ApplyRunInput{
		RunID:   createdRun.Metadata.ID,
		Comment: &applyRunComment,
	})
	if err != nil {
		diags.AddError("Failed to apply a run", err.Error())
		return nil
	}

	// Make sure the run has an apply.
	if appliedRun.Apply == nil {
		msg := fmt.Sprintf("Created run does not have an apply: %s", appliedRun.Metadata.ID)
		diags.AddError(msg, "")
		return nil
	}

	if err = t.waitForJobCompletion(ctx, appliedRun.Apply.CurrentJobID); err != nil {
		diags.AddError("Failed to wait for apply job completion", err.Error())
		return nil
	}

	finishedRun, err := t.client.Run.GetRun(ctx, &sdktypes.GetRunInput{ID: appliedRun.Metadata.ID})
	if err != nil {
		diags.AddError("Failed to get finished run", err.Error())
		return nil
	}

	// If an apply job succeeds, finishedRun.Status is "applied" and
	// finishedRun.Apply.Status is "finished".
	if finishedRun.Status != "applied" {
		diags.AddError("Apply failed", string(finishedRun.Status))
		return nil
	}
	if finishedRun.Apply.Status != "finished" {
		diags.AddError("Apply status", string(finishedRun.Apply.Status))
		return nil
	}

	// In case of a rainy day, make sure the ModuleSource and ModuleVersion *string aren't nil.
	if finishedRun.ModuleSource == nil {
		diags.AddError("Finished run's module source is nil.", "")
		return nil
	}
	if finishedRun.ModuleVersion == nil {
		diags.AddError("Finished run's module version is nil.", "")
		return nil
	}

	// Return a workspace current state model based on the finished run.
	return &WorkspaceCurrentStateModel{
		WorkspacePath: types.StringValue(finishedRun.WorkspacePath),
		ModuleSource:  types.StringValue(*finishedRun.ModuleSource),
		ModuleVersion: types.StringValue(*finishedRun.ModuleVersion),
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

// copyWorkspaceCurrentState copies the contents of a workspace current state.
// It copies the fields from the same type, because there is not a workspace current state defined by Tharsis.
func (t *workspaceCurrentStateResource) copyWorkspaceCurrentState(src, dest *WorkspaceCurrentStateModel) {
	dest.WorkspacePath = src.WorkspacePath
	dest.ModuleSource = src.ModuleSource
	dest.ModuleVersion = src.ModuleVersion
}

// The End.
