package provider

import (
	"context"
	"fmt"
	"time"

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
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/client"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
)

type createRunInput struct {
	model     *ApplyModuleModel
	doDestroy bool
}

type createRunOutput struct {
	moduleVersion     string
	resolvedVariables []*pb.RunVariable
}

// appliedModuleInfo contains what information was available about the latest applied run.
// One or both fields may be nil, in which case information was not available.
type appliedModuleInfo struct {
	moduleSource         *string
	moduleVersion        *string
	wasSuccessfulDestroy bool
	wasManualUpdate      bool
}

const (
	jobCompletionPollInterval = 5 * time.Second
)

// RunVariableModel is used in apply modules to set Terraform and environment variables.
type RunVariableModel struct {
	Value         string `tfsdk:"value"`
	NamespacePath string `tfsdk:"namespace_path"`
	Key           string `tfsdk:"key"`
	Category      string `tfsdk:"category"`
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

	return nil
}

// ApplyModuleModel is the model for an apply_module.
// Please note: Unlike many/most other resources, this model does not exist in the Tharsis API.
// The workspace path, module source, and module version uniquely identify this apply_module.
type ApplyModuleModel struct {
	ID                types.String        `tfsdk:"id"`
	WorkspacePath     types.String        `tfsdk:"workspace_path"`
	WorkspaceID       types.String        `tfsdk:"workspace_id"`
	ModuleSource      types.String        `tfsdk:"module_source"`
	ModuleVersion     types.String        `tfsdk:"module_version"`
	Refresh           types.Bool          `tfsdk:"refresh"`
	Variables         basetypes.ListValue `tfsdk:"variables"`
	ResolvedVariables basetypes.ListValue `tfsdk:"resolved_variables"`
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
	client *client.GRPCClient
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
				Optional:            true,
				DeprecationMessage:  "Use workspace_id instead. This field will be removed in a future version.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"workspace_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the workspace.",
				Description:         "The ID of the workspace.",
				Optional:            true,
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
			"refresh": schema.BoolAttribute{
				MarkdownDescription: "Whether to do a Terraform refresh to update the state based on all managed remote objects.",
				Description:         "Whether to do a Terraform refresh to update the state based on all managed remote objects.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
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
					},
				},
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
	t.client = req.ProviderData.(*client.GRPCClient)
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
	didRun, newDiags := t.createRun(ctx, &createRunInput{
		model: &applyModule,
	})
	resp.Diagnostics.Append(newDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Transform the resolved variables from the run.
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

	currentApplied, newDiags := t.getCurrentApplied(ctx, state)
	resp.Diagnostics.Append(newDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If there is no current state version, currentApplied can be nil.
	// If available/possible, update the state with the computed module source and version.
	if currentApplied != nil && currentApplied.moduleSource != nil {
		state.ModuleSource = types.StringValue(*currentApplied.moduleSource)
	} else {
		state.ModuleSource = types.StringNull()
	}
	if currentApplied != nil && currentApplied.moduleVersion != nil {
		state.ModuleVersion = types.StringValue(*currentApplied.moduleVersion)
	} else {
		state.ModuleVersion = types.StringNull()
	}

	// Don't try to set the resolved variables in the Read method, because the run has not yet been done.

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
	didRun, newDiags := t.createRun(ctx, &createRunInput{
		model: &plan,
	})
	resp.Diagnostics.Append(newDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Capture the module version in case it changed.
	plan.ModuleVersion = types.StringValue(didRun.moduleVersion)

	// Transform the resolved variables from the run.
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

	currentApplied, newDiags := t.getCurrentApplied(ctx, state)
	resp.Diagnostics.Append(newDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If there is no current state version, currentApplied can be nil.
	if currentApplied == nil {
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
	didRun, newDiags2 := t.createRun(ctx, &createRunInput{
		model:     &state,
		doDestroy: true,
	})
	resp.Diagnostics.Append(newDiags2...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Transform the resolved variables from the destroy run.
	resolvedVars, diags := t.toProviderOutputVariables(ctx, didRun.resolvedVariables)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	state.ResolvedVariables = resolvedVars

	// Set the response state to be fully-populated, with or without error.
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// createRun launches a remote run and waits for it to complete.
func (t *applyModuleResource) createRun(ctx context.Context, input *createRunInput) (*createRunOutput, diag.Diagnostics) {
	var diags diag.Diagnostics

	// Convert the input variables.
	vars, err := t.copyRunVariablesToInput(ctx, &input.model.Variables)
	if err != nil {
		diags.AddError("Failed to convert variables to SDK types", err.Error())
		return nil, diags
	}

	// Call CreateRun
	var moduleVersion *string
	if !input.model.ModuleVersion.IsUnknown() {
		moduleVersion = new(input.model.ModuleVersion.ValueString())
	}

	// Convert workspace_path to workspace TRN.
	var workspaceID string
	if v := input.model.WorkspaceID.ValueString(); v != "" {
		workspaceID = v
	} else if v := input.model.WorkspacePath.ValueString(); v != "" {
		workspaceID = trn.TypeWorkspace.Build(v)
	} else {
		diags.AddError("Either workspace_id or workspace_path must be specified", "")
		return nil, diags
	}

	createdRun, err := t.client.RunsClient.CreateRun(ctx, &pb.CreateRunRequest{
		WorkspaceId:   workspaceID,
		IsDestroy:     input.doDestroy,
		ModuleSource:  new(input.model.ModuleSource.ValueString()),
		ModuleVersion: moduleVersion,
		Refresh:       input.model.Refresh.ValueBool(),
		Variables:     vars,
	})
	if err != nil {
		diags.AddError("Failed to create run", err.Error())
		return nil, diags
	}

	// Wait until the plan job has been created before requesting it, to avoid a race
	// where the job does not yet exist. The plan status is the authoritative signal.
	if err = t.waitForRunJob(ctx, createdRun.WorkspaceId, createdRun.Metadata.Id, func(ctx context.Context) (string, error) {
		plan, pErr := t.client.RunsClient.GetPlanByID(ctx, &pb.GetPlanByIDRequest{Id: createdRun.PlanId})
		if pErr != nil {
			return "", pErr
		}
		return plan.Status, nil
	}, planJobReady); err != nil {
		diags.AddError("Failed waiting for plan job", err.Error())
		return nil, diags
	}

	// Wait for plan job.
	planJob, err := t.client.JobsClient.GetLatestJobForPlan(ctx, &pb.GetLatestJobForPlanRequest{
		PlanId: createdRun.PlanId,
	})
	if err != nil {
		diags.AddError("Failed to get plan job", err.Error())
		return nil, diags
	}

	if err = t.waitForJobCompletion(ctx, planJob.Metadata.Id); err != nil {
		diags.AddError("Failed to wait for plan job completion", err.Error())
		return nil, diags
	}

	// Check plan status.
	plan, err := t.client.RunsClient.GetPlanByID(ctx, &pb.GetPlanByIDRequest{Id: createdRun.PlanId})
	if err != nil {
		diags.AddError("Failed to get plan", err.Error())
		return nil, diags
	}

	// Plan status arrives as the lowercased proto status value on the wire.
	switch plan.Status {
	case "canceled":
		diags.AddError("Plan was canceled", plan.Status)
		return nil, diags
	case "errored":
		msg := "Plan failed with unknown error"
		if plan.ErrorMessage != nil {
			msg = *plan.ErrorMessage
		}
		diags.AddError("Plan failed", msg)
		return nil, diags
	}

	// Capture the run ID.
	runID := createdRun.Metadata.Id

	// Get the resolved variables from the run.
	resolvedPlanVarsResp, err := t.client.RunsClient.GetRunVariables(ctx, &pb.GetRunVariablesRequest{Id: runID})
	if err != nil {
		diags.AddError("Failed to get resolved variables", err.Error())
		return nil, diags
	}

	if createdRun.Status == "planned_and_finished" {
		result := &createRunOutput{
			resolvedVariables: resolvedPlanVarsResp.Variables,
		}

		if createdRun.ModuleVersion != nil {
			result.moduleVersion = *createdRun.ModuleVersion
		}
		return result, diags
	}

	// Do the apply run.
	appliedRun, err := t.client.RunsClient.ApplyRun(ctx, &pb.ApplyRunRequest{
		RunId: runID,
	})
	if err != nil {
		diags.AddError("Failed to apply a run", err.Error())
		return nil, diags
	}

	// Wait until the apply job has been created before requesting it, to avoid a race
	// where the job does not yet exist. The apply status is the authoritative signal.
	if err = t.waitForRunJob(ctx, appliedRun.WorkspaceId, runID, func(ctx context.Context) (string, error) {
		apply, aErr := t.client.RunsClient.GetApplyByID(ctx, &pb.GetApplyByIDRequest{Id: appliedRun.ApplyId})
		if aErr != nil {
			return "", aErr
		}
		return apply.Status, nil
	}, applyJobReady); err != nil {
		diags.AddError("Failed waiting for apply job", err.Error())
		return nil, diags
	}

	// Wait for apply job.
	applyJob, err := t.client.JobsClient.GetLatestJobForApply(ctx, &pb.GetLatestJobForApplyRequest{
		ApplyId: appliedRun.ApplyId,
	})
	if err != nil {
		diags.AddError("Failed to get apply job", err.Error())
		return nil, diags
	}

	if err = t.waitForJobCompletion(ctx, applyJob.Metadata.Id); err != nil {
		diags.AddError("Failed to wait for apply job completion", err.Error())
		return nil, diags
	}

	// Check apply status.
	apply, err := t.client.RunsClient.GetApplyByID(ctx, &pb.GetApplyByIDRequest{Id: appliedRun.ApplyId})
	if err != nil {
		diags.AddError("Failed to get apply", err.Error())
		return nil, diags
	}

	// Apply status arrives as the lowercased proto status value on the wire.
	switch apply.Status {
	case "canceled":
		diags.AddError("Apply was canceled", apply.Status)
		return nil, diags
	case "errored":
		msg := "Apply failed with unknown error"
		if apply.ErrorMessage != nil {
			msg = *apply.ErrorMessage
		}
		diags.AddError("Apply failed", msg)
		return nil, diags
	}

	// In case of a rainy day, make sure the ModuleSource and ModuleVersion *string aren't nil.
	if createdRun.ModuleSource == nil {
		diags.AddError("Finished run's module source is nil.", "")
		return nil, diags
	}

	if createdRun.ModuleVersion == nil {
		diags.AddError("Finished run's module version is nil.", "")
		return nil, diags
	}

	// Get the resolved variables from the run.
	resolvedApplyVarsResp, err := t.client.RunsClient.GetRunVariables(ctx, &pb.GetRunVariablesRequest{Id: runID})
	if err != nil {
		diags.AddError("Failed to get resolved variables", err.Error())
		return nil, diags
	}

	// The module version was checked above, so it's safe to dereference.
	// These diags may include those from the inner run if it errored out.
	return &createRunOutput{
		resolvedVariables: resolvedApplyVarsResp.Variables,
		moduleVersion:     *createdRun.ModuleVersion,
	}, diags
}

// waitForRunJob blocks until ready reports the plan/apply job has been created. It
// checks the current status first, then subscribes to run events as a wake-up signal,
// re-checking the authoritative plan/apply status (getStatus) on each event. It returns
// an error if the plan/apply reaches a final state before a job becomes available, or
// if the run event stream cannot be established or closes first.
func (t *applyModuleResource) waitForRunJob(
	ctx context.Context,
	workspaceID, runID string,
	getStatus func(context.Context) (string, error),
	ready func(status string) (bool, error),
) error {
	// Check current state first; the subscription does not replay current state, so a
	// transition that already happened could otherwise be missed.
	status, err := getStatus(ctx)
	if err != nil {
		return err
	}
	if done, rErr := ready(status); rErr != nil || done {
		return rErr
	}

	// Subscribe to run events for wake-ups. The server keeps the subscription open
	// until the RPC is canceled, so use a child context that is canceled on return to
	// tear the stream down once the job is available.
	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	stream, err := t.client.RunsClient.SubscribeToRunEvents(subCtx, &pb.SubscribeToRunEventsRequest{
		WorkspaceId: &workspaceID,
		RunId:       &runID,
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to run events: %w", err)
	}

	for {
		// Block until the next run event, then re-check the authoritative status.
		_, recvErr := stream.Recv()

		status, err := getStatus(ctx)
		if err != nil {
			return err
		}
		done, err := ready(status)
		if err != nil {
			return err
		}
		if done {
			return nil
		}

		// The stream closed before the job became available; the status above is the
		// most current we can get, so report the stream failure.
		if recvErr != nil {
			return fmt.Errorf("run event stream closed before a job was available: %w", recvErr)
		}
	}
}

// planJobReady reports whether the plan's job has been created, based on the plan
// status (the lowercased proto status values sent on the wire). A job exists once the
// plan reaches queued and through its terminal states. A plan that reaches a final
// state without a job (canceled) is an error.
func planJobReady(status string) (bool, error) {
	switch status {
	case "queued", "running", "finished", "errored":
		// A job exists.
		return true, nil
	case "canceled":
		return false, fmt.Errorf("plan reached final state before a job was available; status: %s", status)
	default:
		// pending: job not created yet.
		return false, nil
	}
}

// applyJobReady reports whether the apply's job has been created, based on the apply
// status. A job exists once the apply reaches queued and through its terminal states.
// An apply that reaches a final state without a job (canceled) is an error.
func applyJobReady(status string) (bool, error) {
	switch status {
	case "queued", "running", "finished", "errored":
		// A job exists.
		return true, nil
	case "canceled":
		return false, fmt.Errorf("apply reached final state before a job was available; status: %s", status)
	default:
		// created, pending: job not created yet.
		return false, nil
	}
}

func (t *applyModuleResource) waitForJobCompletion(ctx context.Context, jobID string) error {
	if jobID == "" {
		return fmt.Errorf("empty job ID")
	}

	// Poll until job has finished or the context expires.
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context expired while waiting for job ID %s", jobID)
		case <-time.After(jobCompletionPollInterval):
			job, err := t.client.JobsClient.GetJobByID(ctx, &pb.GetJobByIDRequest{
				Id: jobID,
			})
			if err != nil {
				return fmt.Errorf("failed to get job ID %s", jobID)
			}

			if job.Status == pb.JobStatus_finished {
				return nil
			}
		}
	}
}

// getCurrentApplied returns an ApplyModuleModel reflecting what is currently applied.
func (t *applyModuleResource) getCurrentApplied(ctx context.Context,
	tfState ApplyModuleModel,
) (*appliedModuleInfo, diag.Diagnostics) {
	var diags diag.Diagnostics

	// Get latest run on the target workspace.
	var wsTRN string
	if v := tfState.WorkspaceID.ValueString(); v != "" {
		wsTRN = v
	} else if v := tfState.WorkspacePath.ValueString(); v != "" {
		wsTRN = trn.TypeWorkspace.Build(v)
	} else {
		diags.AddError("Either workspace_id or workspace_path must be specified", "")
		return nil, diags
	}
	ws, err := t.client.WorkspacesClient.GetWorkspaceByID(ctx, &pb.GetWorkspaceByIDRequest{
		Id: wsTRN,
	})
	if err != nil {
		diags.AddError(fmt.Sprintf("Failed to get specified workspace by path: %s", tfState.WorkspacePath.ValueString()), err.Error())
		return nil, diags
	}

	// Get whatever information may be available about the latest applied module.
	if ws.CurrentStateVersionId != "" {
		moduleInfoOutput := &appliedModuleInfo{}

		sv, err := t.client.StateVersionsClient.GetStateVersionByID(ctx, &pb.GetStateVersionByIDRequest{
			Id: ws.CurrentStateVersionId,
		})
		if err != nil {
			diags.AddError("Failed to get state version", err.Error())
			return nil, diags
		}

		if sv.RunId != nil {
			latestRun, err := t.client.RunsClient.GetRunByID(ctx, &pb.GetRunByIDRequest{
				Id: *sv.RunId,
			})
			if err != nil {
				diags.AddError("Failed to get latest run", err.Error())
				return nil, diags
			}

			// Copy out the information that might have been available.
			if latestRun.ModuleSource != nil {
				moduleInfoOutput.moduleSource = latestRun.ModuleSource
			}
			if latestRun.ModuleVersion != nil {
				moduleInfoOutput.moduleVersion = latestRun.ModuleVersion
			}
			if latestRun.IsDestroy && latestRun.Status == "applied" {
				moduleInfoOutput.wasSuccessfulDestroy = true
			}
		} else {
			// Current state has no run ID, so it must have been manually updated.
			moduleInfoOutput.wasManualUpdate = true
		}
		return moduleInfoOutput, diags
	}

	// There was no current state version.
	return nil, diags
}

// copyRunVariablesToInput converts from RunVariableModel to SDK equivalent.
func (t *applyModuleResource) copyRunVariablesToInput(ctx context.Context, list *basetypes.ListValue,
) ([]*pb.RunVariableInput, error) {
	var result []*pb.RunVariableInput

	for _, element := range list.Elements() {
		terraformValue, err := element.ToTerraformValue(ctx)
		if err != nil {
			return nil, err
		}

		var model RunVariableModel
		if err = terraformValue.As(&model); err != nil {
			return nil, err
		}

		result = append(result, &pb.RunVariableInput{
			Value:    &model.Value,
			Key:      model.Key,
			Category: model.Category,
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
	arg []*pb.RunVariable,
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
	}
}
