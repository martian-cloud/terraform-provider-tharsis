package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-hclog"
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
	runpkg "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/run"
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

// RunVariableModel is used in apply modules to set Terraform and environment variables.
type RunVariableModel struct {
	Value         string `tfsdk:"value"`
	NamespacePath string `tfsdk:"namespace_path"`
	Key           string `tfsdk:"key"`
	Category      string `tfsdk:"category"`
}

// enumToDomain renders a generated proto enum as the lowercase domain value Tharsis uses
// elsewhere, e.g. RunStatus_PLANNED_AND_FINISHED -> "planned_and_finished".
//
// The API used to send these statuses as plain lowercase strings and now sends typed
// enums. Converting at the boundary keeps the status comparisons in this file reading as
// the domain vocabulary rather than as generated constants.
func enumToDomain[E interface{ String() string }](v E) string {
	return strings.ToLower(v.String())
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
	WaitTimeout       types.String        `tfsdk:"wait_timeout"`
	Variables         basetypes.ListValue `tfsdk:"variables"`
	ResolvedVariables basetypes.ListValue `tfsdk:"resolved_variables"`
}

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ resource.Resource              = (*applyModuleResource)(nil)
	_ resource.ResourceWithConfigure = (*applyModuleResource)(nil)

	// This resource is deliberately not importable. It implements
	// ResourceWithImportState only so that an import attempt returns an
	// explanation instead of Terraform's generic unsupported-import error.
	_ resource.ResourceWithImportState = (*applyModuleResource)(nil)
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
				MarkdownDescription: "The ID or TRN of the workspace.",
				Description:         "The ID or TRN of the workspace.",
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
			"wait_timeout": schema.StringAttribute{
				MarkdownDescription: "How long to wait for a run blocked on plan approval or a policy gate to be approved, " +
					"overridden or discarded in the Tharsis UI, as a Go duration string (e.g. \"30m\", \"2h\"). " +
					"The provider cannot approve a run itself, so it waits. Defaults to \"1h\".",
				Description: "How long to wait for a run blocked on plan approval or a policy gate to be approved, " +
					"overridden or discarded in the Tharsis UI, as a Go duration string (e.g. \"30m\", \"2h\"). " +
					"The provider cannot approve a run itself, so it waits. Defaults to \"1h\".",
				Optional: true,
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

// ImportState helps the provider implement the ResourceWithImportState interface.
//
// This resource cannot be imported, and this method exists only to explain why
// rather than leaving the user with Terraform's generic "resource does not
// support import" message.
//
// Unlike every other resource in this provider, a tharsis_apply_module has no
// counterpart in the Tharsis API. There is no apply module model, service, or
// identifier; the resource instead performs an action, launching a run against a
// workspace and waiting for it to finish. Its id attribute is a UUID this
// provider manufactures locally. With no stored remote object to look up, there
// is nothing an import could read back.
func (t *applyModuleResource) ImportState(_ context.Context,
	_ resource.ImportStateRequest, resp *resource.ImportStateResponse,
) {
	resp.Diagnostics.AddError(
		"tharsis_apply_module cannot be imported",
		"This resource represents the act of applying a module to a workspace rather than a "+
			"stored Tharsis resource, so there is no existing object for Terraform to import.\n\n"+
			"To manage a workspace that already exists, import tharsis_workspace instead. To apply "+
			"a module to it, add a tharsis_apply_module resource to your configuration and run "+
			"terraform apply, which will start a new run.",
	)
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

	// Parse the wait timeout, which bounds how long we wait for a run blocked on a decision. Done
	// before creating the run so a bad value doesn't leave an unfollowed auto-apply run behind.
	waitTimeout, err := parseWaitTimeout(input.model.WaitTimeout)
	if err != nil {
		diags.AddError("Invalid wait_timeout", err.Error())
		return nil, diags
	}

	createdRun, err := t.client.RunsClient.CreateRun(ctx, &pb.CreateRunRequest{
		WorkspaceId:   workspaceID,
		IsDestroy:     input.doDestroy,
		ModuleSource:  new(input.model.ModuleSource.ValueString()),
		ModuleVersion: moduleVersion,
		Refresh:       new(input.model.Refresh.ValueBool()),
		Variables:     vars,
		// Auto-apply preserves this resource's long-standing behavior of applying the plan without a
		// separate approval step. The provider can't approve a plan interactively, and terraform
		// apply is itself the user's approval. Policy gates still block and are waited on below.
		AutoApply: new(true),
	})
	if err != nil {
		diags.AddError("Failed to create run", err.Error())
		return nil, diags
	}

	// Follow the run through its stages: plan, policy checks and apply. The provider can't make
	// decisions, so it leaves a nil Decider, which makes the follower wait for a blocked run to be
	// approved, overridden or discarded in the Tharsis UI, bounded by waitTimeout. Interrupts leave
	// the run going rather than canceling it mid-apply; Terraform cancellation shouldn't tear down
	// infrastructure changes already underway.
	outcome, err := runpkg.Follow(ctx, t.client, createdRun, &runpkg.Options{
		Observer:    &runObserver{ctx: ctx},
		WaitTimeout: waitTimeout,
		// Run progress is surfaced through the observer to tflog; the follower's own debug logger
		// isn't needed, so discard it.
		Logger: hclog.NewNullLogger(),
	})
	if err != nil {
		diags.AddError("Failed to follow run", err.Error())
		return nil, diags
	}

	switch outcome.Kind {
	case runpkg.OutcomeCompleted:
		// The run applied, or finished planning with no changes.
	case runpkg.OutcomeDetached:
		if outcome.TimedOut {
			diags.AddError("Timed out waiting for the run",
				fmt.Sprintf("Run %q is still waiting on a decision after %s; approve, override or "+
					"discard it in the Tharsis UI, or raise wait_timeout.", createdRun.Metadata.Id, waitTimeout))
		} else {
			diags.AddError("Run is waiting on a decision",
				fmt.Sprintf("Run %q is waiting on a decision that must be made in the Tharsis UI.", createdRun.Metadata.Id))
		}
		return nil, diags
	default:
		// OutcomeFailed, OutcomeDiscarded or OutcomeInterrupted: the run didn't apply.
		diags.AddError("Run did not complete", runFailureMessage(createdRun.Metadata.Id, outcome))
		return nil, diags
	}

	finishedRun := outcome.Run

	// Get the resolved variables from the run.
	resolvedVarsResp, err := t.client.RunsClient.GetRunVariables(ctx, &pb.GetRunVariablesRequest{Id: createdRun.Metadata.Id})
	if err != nil {
		diags.AddError("Failed to get resolved variables", err.Error())
		return nil, diags
	}

	result := &createRunOutput{resolvedVariables: resolvedVarsResp.Variables}
	// Match the original behavior: record the module version when present, and tolerate its absence
	// rather than erroring. A successful module run carries a version, so the fallbacks are belt-and-
	// suspenders; the old no-changes path left it empty on nil, so do the same here.
	if finishedRun.ModuleVersion != nil {
		result.moduleVersion = *finishedRun.ModuleVersion
	} else if createdRun.ModuleVersion != nil {
		result.moduleVersion = *createdRun.ModuleVersion
	}
	return result, diags
}

// parseWaitTimeout parses the wait_timeout attribute, defaulting to the follower's default when
// unset or empty.
func parseWaitTimeout(v types.String) (time.Duration, error) {
	if v.IsNull() || v.IsUnknown() || v.ValueString() == "" {
		return runpkg.DefaultWaitTimeout, nil
	}
	d, err := time.ParseDuration(v.ValueString())
	if err != nil {
		return 0, fmt.Errorf("wait_timeout %q is not a valid duration: %w", v.ValueString(), err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("wait_timeout must be positive, got %q", v.ValueString())
	}
	return d, nil
}

// runFailureMessage explains why a run didn't complete, with the plan or apply error when the run
// has one, or the stage it failed at otherwise.
func runFailureMessage(runID string, outcome *runpkg.Outcome) string {
	msg := fmt.Sprintf("Run %q ended with status %s", runID, outcome.Run.Status.String())
	if errMsg := outcome.Run.GetApply().GetErrorMessage(); errMsg != "" {
		return fmt.Sprintf("%s: apply failed: %s", msg, errMsg)
	}
	if errMsg := outcome.Run.GetPlan().GetErrorMessage(); errMsg != "" {
		return fmt.Sprintf("%s: plan failed: %s", msg, errMsg)
	}
	if outcome.Stage.Name != "" {
		return fmt.Sprintf("%s during %s.", msg, outcome.Stage.Name)
	}

	return msg + "."
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
			if latestRun.IsDestroy && enumToDomain(latestRun.Status) == "applied" {
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
