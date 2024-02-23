package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	tharsis "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg"
	sdktypes "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

const (
	// getLogsChunkSize is the maximum number of bytes to request in a single log request.
	getLogsChunkSize = 5000
)

type createRunInput struct {
	model       *ApplyModuleModel
	doDestroy   bool
	speculative bool
}

type createRunOutput struct {
	moduleVersion     string
	resolvedVariables []sdktypes.RunVariable
}

// appliedModuleInfo contains what information was available about the latest applied run.
// One or both fields may be nil, in which case information was not available.
type appliedModuleInfo struct {
	moduleSource         *string
	moduleVersion        *string
	wasSuccessfulDestroy bool
	wasManualUpdate      bool
	resolvedVariables    []sdktypes.RunVariable
}

const (
	jobCompletionPollInterval = 5 * time.Second
)

var applyRunComment = "terraform-provider-tharsis" // must be var, not const, to take address

// RunVariableModel is used in apply modules to set Terraform and environment variables.
type RunVariableModel struct {
	Value         string `tfsdk:"value"`
	NamespacePath string `tfsdk:"namespace_path"`
	Key           string `tfsdk:"key"`
	Category      string `tfsdk:"category"`
	HCL           bool   `tfsdk:"hcl"`
}

// FromTerraform5Value converts a RunVariable from Terraform values to Go equivalent.
// This method name is required by the interface we are implementing.  Please see
// https://pkg.go.dev/github.com/hashicorp/terraform-plugin-go/tfprotov5/tftypes
func (e *RunVariableModel) FromTerraform5Value(val tftypes.Value) error {
	v := map[string]tftypes.Value{}
	err := val.As(&v)
	if err != nil {
		return err
	}

	err = v["value"].As(&e.Value)
	if err != nil {
		return err
	}

	err = v["key"].As(&e.Key)
	if err != nil {
		return err
	}

	err = v["category"].As(&e.Category)
	if err != nil {
		return err
	}

	err = v["hcl"].As(&e.HCL)
	if err != nil {
		return err
	}

	return nil
}

// ApplyModuleModel is the model for an apply_module.
// Please note: Unlike many/most other resources, this model does not exist in the Tharsis API.
// The workspace path, module source, and module version uniquely identify this apply_module.
type ApplyModuleModel struct {
	ID                types.String        `tfsdk:"id"`
	WorkspacePath     types.String        `tfsdk:"workspace_path"`
	ModuleSource      types.String        `tfsdk:"module_source"`
	ModuleVersion     types.String        `tfsdk:"module_version"`
	Variables         basetypes.ListValue `tfsdk:"variables"`
	ResolvedVariables basetypes.ListValue `tfsdk:"resolved_variables"`
	Speculative       types.Bool          `tfsdk:"speculative"`
}

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ resource.Resource              = (*applyModuleResource)(nil)
	_ resource.ResourceWithConfigure = (*applyModuleResource)(nil)
)

// NewApplyModuleResource is a helper function to simplify the provider implementation.
func NewApplyModuleResource() resource.Resource {
	return &applyModuleResource{}
}

type applyModuleResource struct {
	client *tharsis.Client
}

// Metadata returns the full name of the resource, including prefix, underscore, instance name.
func (t *applyModuleResource) Metadata(_ context.Context,
	_ resource.MetadataRequest, resp *resource.MetadataResponse,
) {
	resp.TypeName = "tharsis_apply_module"
}

func (t *applyModuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	description := "Defines and manages tharsis_apply_module resources, which launch runs in other workspaces."

	resp.Schema = schema.Schema{
		Version:             1,
		MarkdownDescription: description,
		Description:         description,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "An ID for this tharsis_apply_module resource.",
				Description:         "An ID for this tharsis_apply_module resource.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(), // set once during create, kept in state thereafter
				},
			},
			"workspace_path": schema.StringAttribute{
				MarkdownDescription: "The full path of the workspace.",
				Description:         "The full path of the workspace.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"module_source": schema.StringAttribute{
				MarkdownDescription: "The source of the module.",
				Description:         "The source of the module.",
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
			"variables": schema.ListNestedAttribute{
				MarkdownDescription: "Optional list of variables for the run in the target workspace.",
				Description:         "Optional list of variables for the run in the target workspace.",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"value": schema.StringAttribute{
							MarkdownDescription: "Value of the variable.",
							Description:         "Value of the variable.",
							Required:            true,
						},
						"key": schema.StringAttribute{
							MarkdownDescription: "Key or name of this variable.",
							Description:         "Key or name of this variable.",
							Required:            true,
						},
						"category": schema.StringAttribute{
							MarkdownDescription: "Category of this variable, 'terraform' or 'environment'.",
							Description:         "Category of this variable, 'terraform' or 'environment'.",
							Required:            true,
						},
						"hcl": schema.BoolAttribute{
							MarkdownDescription: "Whether this variable is HCL (vs. string).",
							Description:         "Whether this variable is HCL (vs. string).",
							Required:            true,
						},
					},
				},
			},
			"speculative": schema.BoolAttribute{
				MarkdownDescription: "Whether the run will be speculative, default is false.",
				Description:         "Whether the run will be speculative, default is false.",
				Optional:            true,
				Default:             booldefault.StaticBool(false),
				Computed:            true, // Must be computed if setting a default in the schema.
			},
			"resolved_variables": schema.ListNestedAttribute{
				MarkdownDescription: "The variables that were used by the run.",
				Description:         "The variables that were used by the run.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"value": schema.StringAttribute{
							MarkdownDescription: "Value of the variable.",
							Description:         "Value of the variable.",
							Computed:            true,
						},
						"namespace_path": schema.StringAttribute{
							MarkdownDescription: "Namespace path of the variable.",
							Description:         "Namespace path of the variable.",
							Computed:            true,
						},
						"key": schema.StringAttribute{
							MarkdownDescription: "Key or name of this variable.",
							Description:         "Key or name of this variable.",
							Computed:            true,
						},
						"category": schema.StringAttribute{
							MarkdownDescription: "Category of this variable, 'terraform' or 'environment'.",
							Description:         "Category of this variable, 'terraform' or 'environment'.",
							Computed:            true,
						},
						"hcl": schema.BoolAttribute{
							MarkdownDescription: "Whether this variable is HCL (vs. string).",
							Description:         "Whether this variable is HCL (vs. string).",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

// Configure lets the provider implement the ResourceWithConfigure interface.
func (t *applyModuleResource) Configure(_ context.Context,
	req resource.ConfigureRequest, _ *resource.ConfigureResponse,
) {
	if req.ProviderData == nil {
		return
	}
	t.client = req.ProviderData.(*tharsis.Client)
}

func (t *applyModuleResource) Create(ctx context.Context,
	req resource.CreateRequest, resp *resource.CreateResponse,
) {
	// Retrieve values from apply module.
	var applyModule ApplyModuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &applyModule)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Do plan and apply, no destroy.
	var didRun createRunOutput
	resp.Diagnostics.Append(t.createRun(ctx, &createRunInput{
		model:       &applyModule,
		speculative: applyModule.Speculative.ValueBool(),
	}, &didRun)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the resolved variables from the run.
	resolvedVars, diags := t.toProviderOutputVariables(ctx, didRun.resolvedVariables)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	// Update the plan with the computed ID.
	applyModule.ID = types.StringValue(uuid.New().String())
	applyModule.ModuleVersion = types.StringValue(didRun.moduleVersion)
	applyModule.ResolvedVariables = resolvedVars

	// Set the response state to the fully-populated plan, whether or not there is an error.
	resp.Diagnostics.Append(resp.State.Set(ctx, applyModule)...)
}

func (t *applyModuleResource) Read(ctx context.Context,
	req resource.ReadRequest, resp *resource.ReadResponse,
) {
	// Get the current state.
	var state ApplyModuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var currentApplied appliedModuleInfo
	resp.Diagnostics.Append(t.getCurrentApplied(ctx, state, &currentApplied)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update the state with the computed module source and version.
	if currentApplied.moduleSource != nil {
		state.ModuleSource = types.StringValue(*currentApplied.moduleSource)
	} else {
		state.ModuleSource = types.StringNull()
	}
	if currentApplied.moduleVersion != nil {
		state.ModuleVersion = types.StringValue(*currentApplied.moduleVersion)
	} else {
		state.ModuleVersion = types.StringNull()
	}

	// Get the resolved variables from the run that produced the state.
	resolvedVars, diags := t.toProviderOutputVariables(ctx, currentApplied.resolvedVariables)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	state.ResolvedVariables = resolvedVars

	// Set the refreshed state, whether or not there is an error.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (t *applyModuleResource) Update(ctx context.Context,
	req resource.UpdateRequest, resp *resource.UpdateResponse,
) {
	// Retrieve values from plan.
	var plan ApplyModuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Do the run.
	var didRun createRunOutput
	resp.Diagnostics.Append(t.createRun(ctx, &createRunInput{
		model:       &plan,
		speculative: plan.Speculative.ValueBool(),
	}, &didRun)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Capture the module version in case it changed.
	plan.ModuleVersion = types.StringValue(didRun.moduleVersion)

	// Get the resolved variables from the run.
	resolvedVars, diags := t.toProviderOutputVariables(ctx, didRun.resolvedVariables)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	plan.ResolvedVariables = resolvedVars

	// Set the response state to the fully-populated plan, with or without error.
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (t *applyModuleResource) Delete(ctx context.Context,
	req resource.DeleteRequest, resp *resource.DeleteResponse,
) {
	// Get the current state.
	var state ApplyModuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var currentApplied appliedModuleInfo
	resp.Diagnostics.Append(t.getCurrentApplied(ctx, state, &currentApplied)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If the latest run was a successful destroy, all resources have already
	// been destroyed, so there's nothing that needs to be done here.
	if currentApplied.wasSuccessfulDestroy {
		return
	}

	// Lack of a module source is not a reliable indication that a configuration version had been deployed,
	// so we can't use it to determine whether to refuse to delete.  For now, don't check for that.

	// Refuse to destroy if the current state was manually modified
	// (as measured by the current state having no run ID).
	if currentApplied.wasManualUpdate {
		resp.Diagnostics.AddError("Current state had been manually updated, will not delete", "")
		return
	}

	// Note: There's no need to check the PreventDestroyPlan flag, because the Tharsis API enforces that.

	// If the module source or module version is available and differs, error out.
	if currentApplied.moduleSource != nil {
		if state.ModuleSource.ValueString() != *currentApplied.moduleSource {
			resp.Diagnostics.AddError("Module source differs, cannot delete", "")
			return
		}
	}
	if currentApplied.moduleVersion != nil {
		if state.ModuleVersion.ValueString() != *currentApplied.moduleVersion {
			resp.Diagnostics.AddError("Module version differs, cannot delete", "")
			return
		}
	}

	// The apply module is being deleted, so don't use the module version output.
	resp.Diagnostics.Append(t.createRun(ctx, &createRunInput{
		model:     &state,
		doDestroy: true,
	}, nil)...) // nil means no module version output is wanted
	if resp.Diagnostics.HasError() {
		return
	}
}

// createRun launches a remote run and waits for it to complete.
func (t *applyModuleResource) createRun(ctx context.Context,
	input *createRunInput, output *createRunOutput,
) diag.Diagnostics {
	var diags diag.Diagnostics

	// Convert the input variables.
	vars, err := t.copyRunVariablesToInput(ctx, &input.model.Variables)
	if err != nil {
		diags.AddError("Failed to convert variables to SDK types", err.Error())
		return diags
	}

	// Call CreateRun
	var moduleVersion *string
	if !input.model.ModuleVersion.IsUnknown() {
		moduleVersion = ptr.String(input.model.ModuleVersion.ValueString())
	}
	createdRun, err := t.client.Run.CreateRun(ctx, &sdktypes.CreateRunInput{
		WorkspacePath: input.model.WorkspacePath.ValueString(),
		IsDestroy:     input.doDestroy,
		ModuleSource:  ptr.String(input.model.ModuleSource.ValueString()),
		ModuleVersion: moduleVersion,
		Speculative:   &input.speculative,
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

		// Bring in any error message(s) from the finished inner plan run.
		innerPlanRunDiags := t.extractRunError(ctx, plannedRun)
		diags.Append(innerPlanRunDiags...)
		if innerPlanRunDiags.HasError() {
			return diags
		}

		diags.AddError("Plan failed", string(plannedRun.Status))
		return diags
	}
	if plannedRun.Plan.Status != "finished" {

		// Bring in any error message(s) from the finished inner plan run.
		innerPlanRunDiags := t.extractRunError(ctx, plannedRun)
		diags.Append(innerPlanRunDiags...)
		if innerPlanRunDiags.HasError() {
			return diags
		}

		diags.AddError("Plan failed", string(plannedRun.Plan.Status))
		return diags
	}

	// Capture the run ID.
	runID := plannedRun.Metadata.ID

	// Get the resolved variables from the run.
	resolvedPlanVars, err := t.client.Run.GetRunVariables(ctx, &sdktypes.GetRunInput{ID: runID})
	if err != nil {
		diags.AddError("Failed to get resolved variables", err.Error())
		return diags
	}

	if plannedRun.Status == "planned_and_finished" {
		if output != nil {
			*output = createRunOutput{
				resolvedVariables: resolvedPlanVars,
			}

			if plannedRun.ModuleVersion != nil {
				output.moduleVersion = *plannedRun.ModuleVersion
			}
		}
		return nil
	}

	// Do the apply run.
	appliedRun, err := t.client.Run.ApplyRun(ctx, &sdktypes.ApplyRunInput{
		RunID:   runID,
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

		// Bring in any error message(s) from the finished inner apply run.
		innerApplyRunDiags := t.extractRunError(ctx, finishedRun)
		diags.Append(innerApplyRunDiags...)
		if innerApplyRunDiags.HasError() {
			return diags
		}

		diags.AddError("Apply failed", string(finishedRun.Status))
		return diags
	}
	if finishedRun.Apply.Status != "finished" {

		// Bring in any error message(s) from the finished inner apply run.
		innerApplyRunDiags := t.extractRunError(ctx, finishedRun)
		diags.Append(innerApplyRunDiags...)
		if innerApplyRunDiags.HasError() {
			return diags
		}

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

	// Get the resolved variables from the run.
	resolvedApplyVars, err := t.client.Run.GetRunVariables(ctx, &sdktypes.GetRunInput{ID: finishedRun.Metadata.ID})
	if err != nil {
		diags.AddError("Failed to get resolved variables", err.Error())
		return diags
	}

	if output != nil {
		*output = createRunOutput{
			resolvedVariables: resolvedApplyVars,
		}

		if finishedRun.ModuleVersion != nil {
			output.moduleVersion = *finishedRun.ModuleVersion
		}
	}

	// These diags may include those from the inner run if it errored out.
	return diags
}

func (t *applyModuleResource) waitForJobCompletion(ctx context.Context, jobID *string) error {
	if jobID == nil {
		return fmt.Errorf("nil job ID")
	}

	// Poll until job has finished or the context expires.
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context expired while waiting for job ID %s", *jobID)
		case <-time.After(jobCompletionPollInterval):
			job, err := t.client.Job.GetJob(ctx, &sdktypes.GetJobInput{
				ID: *jobID,
			})
			if err != nil {
				return fmt.Errorf("failed to get job ID %s", *jobID)
			}

			if job.Status == "finished" {
				return nil
			}
		}
	}
}

// getCurrentApplied returns an ApplyModuleModel reflecting what is currently applied.
func (t *applyModuleResource) getCurrentApplied(ctx context.Context,
	tfState ApplyModuleModel, moduleInfoOutput *appliedModuleInfo,
) diag.Diagnostics {
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

	// Get whatever information may be available about the latest applied module.
	if ws.CurrentStateVersion != nil {
		if ws.CurrentStateVersion.RunID != "" {
			latestRun, err := t.client.Run.GetRun(ctx, &sdktypes.GetRunInput{
				ID: ws.CurrentStateVersion.RunID,
			})
			if err != nil {
				diags.AddError("Failed to get latest run", err.Error())
				return diags
			}

			// Copy out the information that might have been available.
			if latestRun.ModuleSource != nil {
				moduleInfoOutput.moduleSource = latestRun.ModuleSource
			}
			if latestRun.ModuleVersion != nil {
				moduleInfoOutput.moduleVersion = latestRun.ModuleVersion
			}
			if latestRun.IsDestroy && (latestRun.Status == sdktypes.RunApplied) && (latestRun.Apply != nil) {
				moduleInfoOutput.wasSuccessfulDestroy = true
			}

			// Get the resolved variables from the run that produced the state.
			resolvedVars, err := t.client.Run.GetRunVariables(ctx, &sdktypes.GetRunInput{ID: latestRun.Metadata.ID})
			if err != nil {
				diags.AddError("Failed to get resolved variables", err.Error())
				return diags
			}

			moduleInfoOutput.resolvedVariables = resolvedVars
		} else {
			// Current state has no run ID, so it must have been manually updated.
			moduleInfoOutput.wasManualUpdate = true
		}
	}

	return nil
}

// extractRunError extracts the error from a run's logs (if the run errored out).
func (t *applyModuleResource) extractRunError(ctx context.Context, run *sdktypes.Run) diag.Diagnostics {
	var diags diag.Diagnostics

	if run.Status == sdktypes.RunErrored {
		var jobID string

		switch {
		case run.Apply != nil:
			if run.Apply.CurrentJobID != nil {
				jobID = *run.Apply.CurrentJobID
			}
		case run.Plan != nil:
			if run.Plan.CurrentJobID != nil {
				jobID = *run.Plan.CurrentJobID
			}
		}

		if jobID == "" {
			diags.AddWarning("Run status is errored, but no job ID found", "")
			return diags
		}

		// Must get the job to know the size of the logs to paginate in reverse.
		job, err := t.client.Job.GetJob(ctx, &sdktypes.GetJobInput{
			ID: jobID,
		})
		if err != nil {
			diags.AddError("Failed to get job", err.Error())
			return diags
		}

		// Get the logs from the end.  There will most likely be a smaller chunk at the beginning.
		allLogs := ""
		currentStart := int32(job.LogSize) - getLogsChunkSize
		nextChunkSize := int32(getLogsChunkSize)
		if currentStart < 0 {
			// Only one chunk to read.
			currentStart = 0
			nextChunkSize = int32(job.LogSize)
		}
		for {
			logs, err := t.client.Job.GetJobLogs(ctx, &sdktypes.GetJobLogsInput{
				JobID: jobID,
				Start: currentStart,
				Limit: &nextChunkSize,
			})
			if err != nil {
				diags.AddError("Failed to get job logs", err.Error())
				return diags
			}

			// Workaround: The API returns one more character than asked for.
			newLogs := logs.Logs
			if len(newLogs) > int(nextChunkSize) {
				newLogs = newLogs[:nextChunkSize]
			}

			allLogs = newLogs + allLogs
			if strings.HasPrefix(allLogs, "Error: ") {
				// Found the error, so break out of the loop.
				break
			}

			if currentStart == 0 {
				// No error found, and we've read the whole log.
				break
			}

			if currentStart >= getLogsChunkSize {
				currentStart -= getLogsChunkSize
			} else {
				// A smaller chunk at the beginning.
				nextChunkSize = currentStart
				currentStart = 0
			}
		}

		// Find the first mention of "error" in the logs.
		splitLogs := strings.Split(allLogs, "\n")
		foundIx := -1
		for i, log := range splitLogs {
			if strings.HasPrefix(strings.ToLower(log), "error") {
				foundIx = i
				break
			}
		}

		if foundIx < 0 {
			// No error found, so return empty diags.
			return diags
		}

		// Must truncate before the state creation line.
		splitLogs = splitLogs[foundIx:] // Only keep the error and what follows.
		for i, log := range splitLogs {
			if strings.Contains(log, "Created new state version") {
				if i == 0 {
					diags.AddWarning("Failed to find logs between 'error' and 'Created new state version'", "")
					return diags
				}
				splitLogs = splitLogs[:i-1]
				break
			}
		}

		// Must format the message as a single string.
		// Prefix each line with vertical bar and tab so it's clear this comes from the inner run's logs.
		if foundIx >= 0 {
			diags.AddError(strings.TrimPrefix(strings.Join(splitLogs, "\n"), "Error: "), "")
		}
	}

	return diags
}

// copyRunVariablesToInput converts from RunVariableModel to SDK equivalent.
func (t *applyModuleResource) copyRunVariablesToInput(ctx context.Context, list *basetypes.ListValue,
) ([]sdktypes.RunVariable, error) {
	result := []sdktypes.RunVariable{}

	for _, element := range list.Elements() {
		terraformValue, err := element.ToTerraformValue(ctx)
		if err != nil {
			return nil, err
		}

		var model RunVariableModel
		if err = terraformValue.As(&model); err != nil {
			return nil, err
		}

		result = append(result, sdktypes.RunVariable{
			Value:    &model.Value,
			Key:      model.Key,
			Category: sdktypes.VariableCategory(model.Category),
			HCL:      model.HCL,
		})
	}

	// Terraform generally wants to see nil rather than an empty list.
	if len(result) == 0 {
		result = nil
	}

	return result, nil
}

// toProviderOutputVariables converts SDK variables from a finished run to the types the provider can return to Terraform.
func (t *applyModuleResource) toProviderOutputVariables(
	ctx context.Context,
	arg []sdktypes.RunVariable,
) (basetypes.ListValue, diag.Diagnostics) {
	variables := []types.Object{}

	for _, variable := range arg {
		val := ""
		if variable.Value != nil {
			val = *variable.Value
		}

		model := &RunVariableModel{
			Value:    val,
			Key:      variable.Key,
			Category: string(variable.Category),
			HCL:      variable.HCL,
		}

		if variable.NamespacePath != nil {
			model.NamespacePath = *variable.NamespacePath
		}

		value, objectDiags := basetypes.NewObjectValueFrom(ctx, t.outputVariableAttributes(), model)
		if objectDiags.HasError() {
			return basetypes.ListValue{}, objectDiags
		}

		variables = append(variables, value)
	}

	list, listDiags := basetypes.NewListValueFrom(ctx, basetypes.ObjectType{
		AttrTypes: t.outputVariableAttributes(),
	}, variables)
	if listDiags.HasError() {
		return basetypes.ListValue{}, listDiags
	}

	return list, nil
}

func (t *applyModuleResource) outputVariableAttributes() map[string]attr.Type {
	return map[string]attr.Type{
		"value":          types.StringType,
		"namespace_path": types.StringType,
		"key":            types.StringType,
		"category":       types.StringType,
		"hcl":            types.BoolType,
	}
}
