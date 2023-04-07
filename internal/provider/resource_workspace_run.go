package provider

import (
	"context"
	"encoding/json"
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

// WorkspaceRunModel is the model for a workspace_run.
// Please note: Unlike many/most other resources, this model does not exist in the Tharsis API.
// The workspace path, module source, and module version uniquely identify this workspace_run.
type WorkspaceRunModel struct {
	WorkspacePath types.String `tfsdk:"workspace_path"`
	ModuleSource  types.String `tfsdk:"module_source"`
	ModuleVersion types.String `tfsdk:"module_version"`
	Variables     types.String `tfsdk:"variables"`
}

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ resource.Resource                = (*workspaceRunResource)(nil)
	_ resource.ResourceWithConfigure   = (*workspaceRunResource)(nil)
	_ resource.ResourceWithImportState = (*workspaceRunResource)(nil)
)

// NewWorkspaceRunResource is a helper function to simplify the provider implementation.
func NewWorkspaceRunResource() resource.Resource {
	return &workspaceRunResource{}
}

type workspaceRunResource struct {
	client *tharsis.Client
}

// Metadata returns the full name of the resource, including prefix, underscore, instance name.
func (t *workspaceRunResource) Metadata(ctx context.Context,
	req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "tharsis_workspace_run"
}

func (t *workspaceRunResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	description := "Defines and manages a workspace run."

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
			},
			"module_version": schema.StringAttribute{
				MarkdownDescription: "The version identifier of the module.",
				Description:         "The version identifier of the module.",
				Optional:            true,
				Computed:            true, // computed if not supplied
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"variables": schema.StringAttribute{
				MarkdownDescription: "Optional variables for the run in the target workspace.",
				Description:         "Optional variables for the run in the target workspace.",
				Optional:            true,
				// Will remain unset if not supplied.
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Configure lets the provider implement the ResourceWithConfigure interface.
func (t *workspaceRunResource) Configure(_ context.Context,
	req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	t.client = req.ProviderData.(*tharsis.Client)
}

func (t *workspaceRunResource) Create(ctx context.Context,
	req resource.CreateRequest, resp *resource.CreateResponse) {

	// Retrieve values from workspace run.
	var workspaceRun WorkspaceRunModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &workspaceRun)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var created WorkspaceRunModel
	resp.Diagnostics.Append(t.doApplyOrDestroyRun(ctx, workspaceRun, false, &created)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update the plan with the computed attribute values.
	t.copyWorkspaceRun(&created, &workspaceRun)

	// Set the response state to the fully-populated plan, whether or not there is an error.
	resp.Diagnostics.Append(resp.State.Set(ctx, workspaceRun)...)
}

func (t *workspaceRunResource) Read(ctx context.Context,
	req resource.ReadRequest, resp *resource.ReadResponse) {

	// Get the current state.
	var state WorkspaceRunModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var deployed WorkspaceRunModel
	resp.Diagnostics.Append(t.getCurrentDeployment(ctx, state, &deployed)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update the state with the computed attribute values.
	t.copyWorkspaceRun(&deployed, &state)

	// Set the refreshed state, whether or not there is an error.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (t *workspaceRunResource) Update(ctx context.Context,
	req resource.UpdateRequest, resp *resource.UpdateResponse) {

	// Retrieve values from plan.
	var plan WorkspaceRunModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// FIXME: See other review items to set this correctly.
	isDestroyRun := false

	// Apply or destroy, depending on the isDestroyRun argument.
	var updated WorkspaceRunModel
	resp.Diagnostics.Append(t.doApplyOrDestroyRun(ctx, plan, isDestroyRun, &updated)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Copy all fields returned by Tharsis back into the plan.
	t.copyWorkspaceRun(&updated, &plan)

	// Set the response state to the fully-populated plan, with or without error.
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (t *workspaceRunResource) Delete(ctx context.Context,
	req resource.DeleteRequest, resp *resource.DeleteResponse) {

	// Get the current state.
	var state WorkspaceRunModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var deployed WorkspaceRunModel
	resp.Diagnostics.Append(t.getCurrentDeployment(ctx, state, &deployed)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If the module source or module version differs, error out.
	if state.ModuleSource != deployed.ModuleSource {
		resp.Diagnostics.AddError("Module source differs, cannot delete", "")
		return
	}
	if state.ModuleVersion != deployed.ModuleVersion {
		resp.Diagnostics.AddError("Module version differs, cannot delete", "")
		return
	}

	// The workspace run is being deleted, so don't use the returned value.
	var deleted WorkspaceRunModel
	resp.Diagnostics.Append(t.doApplyOrDestroyRun(ctx, state, true, &deleted)...)
	if resp.Diagnostics.HasError() {
		return
	}
	_ = deleted
}

// ImportState helps the provider implement the ResourceWithImportState interface.
func (t *workspaceRunResource) ImportState(ctx context.Context,
	req resource.ImportStateRequest, resp *resource.ImportStateResponse) {

	resp.Diagnostics.AddError(
		"Import of workspace is not supported.",
		"",
	)
}

// Because there is no Tharsis-defined struct for a workspace run resource, return this module's struct.
func (t *workspaceRunResource) doApplyOrDestroyRun(ctx context.Context,
	model WorkspaceRunModel, isDestroy bool, target *WorkspaceRunModel) diag.Diagnostics {
	var diags diag.Diagnostics

	// If variables are supplied, unmarshal them.
	var vars []sdktypes.RunVariable
	if !model.Variables.IsUnknown() {
		s := model.Variables.ValueString()
		if s != "" { // If empty string is passed in, don't try to unmarshal it.
			if err := json.Unmarshal([]byte(s), &vars); err != nil {
				diags.AddError("Failed to unmarshal the run variables", err.Error())
				return diags
			}
		}
	}

	// Call CreateRun
	var moduleVersion *string
	if !model.ModuleVersion.IsUnknown() {
		moduleVersion = ptr.String(model.ModuleVersion.ValueString())
	}
	// Using module registry path and version, so no ConfigurationVersionID.
	createdRun, err := t.client.Run.CreateRun(ctx, &sdktypes.CreateRunInput{
		WorkspacePath: model.WorkspacePath.ValueString(),
		IsDestroy:     isDestroy,
		ModuleSource:  ptr.String(model.ModuleSource.ValueString()),
		ModuleVersion: moduleVersion,
		Variables:     vars,
	})
	if err != nil {
		diags.AddError("Failed to create run", err.Error())
		return diags
	}

	if err = t.waitForJobCompletion(ctx, createdRun.Plan.CurrentJobID); err != nil {
		diags.AddError("Failed to wait for plan job completion", err.Error())
		return diags
	}

	plannedRun, err := t.client.Run.GetRun(ctx, &sdktypes.GetRunInput{ID: createdRun.Metadata.ID})
	if err != nil {
		diags.AddError("Failed to get planned run", err.Error())
		return diags
	}

	// If the plan fails, both plannedRun.Status and plannedRun.Plan.Status are "errored".
	// If the plan succeeds, plannedRun.Status is "planned",
	// while plannedRun.Plan.Status is "finished".
	//
	if !strings.HasPrefix(string(plannedRun.Status), "planned") {
		diags.AddError("Plan failed", string(plannedRun.Status))
		return diags
	}
	if plannedRun.Plan.Status != "finished" {
		diags.AddError("Plan failed", string(plannedRun.Plan.Status))
		return diags
	}

	// Do the apply run.
	appliedRun, err := t.client.Run.ApplyRun(ctx, &sdktypes.ApplyRunInput{
		RunID:   createdRun.Metadata.ID,
		Comment: &applyRunComment,
	})
	if err != nil {
		diags.AddError("Failed to apply a run", err.Error())
		return diags
	}

	// Make sure the run has an apply.
	if appliedRun.Apply == nil {
		msg := fmt.Sprintf("Created run does not have an apply: %s", appliedRun.Metadata.ID)
		diags.AddError(msg, "")
		return diags
	}

	if err = t.waitForJobCompletion(ctx, appliedRun.Apply.CurrentJobID); err != nil {
		diags.AddError("Failed to wait for apply job completion", err.Error())
		return diags
	}

	finishedRun, err := t.client.Run.GetRun(ctx, &sdktypes.GetRunInput{ID: appliedRun.Metadata.ID})
	if err != nil {
		diags.AddError("Failed to get finished run", err.Error())
		return diags
	}

	// If an apply job succeeds, finishedRun.Status is "applied" and
	// finishedRun.Apply.Status is "finished".
	if finishedRun.Status != "applied" {
		diags.AddError("Apply failed", string(finishedRun.Status))
		return diags
	}
	if finishedRun.Apply.Status != "finished" {
		diags.AddError("Apply status", string(finishedRun.Apply.Status))
		return diags
	}

	// In case of a rainy day, make sure the ModuleSource and ModuleVersion *string aren't nil.
	if finishedRun.ModuleSource == nil {
		diags.AddError("Finished run's module source is nil.", "")
		return diags
	}
	if finishedRun.ModuleVersion == nil {
		diags.AddError("Finished run's module version is nil.", "")
		return diags
	}

	// Return a workspace run model based on the finished run.
	target.WorkspacePath = types.StringValue(finishedRun.WorkspacePath)
	target.ModuleSource = types.StringValue(*finishedRun.ModuleSource)
	target.ModuleVersion = types.StringValue(*finishedRun.ModuleVersion)
	target.Variables = model.Variables // Cannot get variables back from a workspace or run, so pass them through.
	return nil
}

func (t *workspaceRunResource) waitForJobCompletion(ctx context.Context, jobID *string) error {
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

// getCurrentDeployment returns a WorkspaceRunModel reflecting what is currently deployed.
func (t *workspaceRunResource) getCurrentDeployment(ctx context.Context,
	tfState WorkspaceRunModel, target *WorkspaceRunModel) diag.Diagnostics {
	var diags diag.Diagnostics

	// Get latest run on the target workspace.
	wsPath := tfState.WorkspacePath.ValueString()
	ws, err := t.client.Workspaces.GetWorkspace(ctx, &sdktypes.GetWorkspaceInput{
		Path: &wsPath,
	})
	if err != nil {
		diags.AddError(fmt.Sprintf("Failed to get specified workspace by path: %s", wsPath), err.Error())
		return diags
	}
	latestRun, err := t.client.Run.GetRun(ctx, &sdktypes.GetRunInput{
		ID: ws.CurrentStateVersion.RunID,
	})
	if err != nil {
		diags.AddError("Failed to get latest run", err.Error())
		return diags
	}

	// Make sure the module source and module version are not nil.
	if latestRun.ModuleSource == nil {
		diags.AddError("No module source available", fmt.Sprintf("for workspace %s", latestRun.WorkspacePath))
		return diags
	}
	if latestRun.ModuleVersion == nil {
		diags.AddError("No module version available", fmt.Sprintf("for workspace %s", latestRun.WorkspacePath))
		return diags
	}

	target.WorkspacePath = tfState.WorkspacePath
	target.ModuleSource = types.StringValue(*latestRun.ModuleSource)
	target.ModuleVersion = types.StringValue(*latestRun.ModuleVersion)
	target.Variables = tfState.Variables

	return nil
}

// copyWorkspaceRun copies the contents of a workspace run.
// It copies the fields from the same type, because there is not a workspace run defined by Tharsis.
func (t *workspaceRunResource) copyWorkspaceRun(src, dest *WorkspaceRunModel) {
	dest.WorkspacePath = src.WorkspacePath
	dest.ModuleSource = src.ModuleSource
	dest.ModuleVersion = src.ModuleVersion
	dest.Variables = src.Variables
}

// The End.
