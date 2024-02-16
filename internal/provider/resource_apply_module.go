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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/martian-cloud/terraform-provider-tharsis/internal/modifiers"
	tharsis "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg"
	sdktypes "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

type doRunInput struct {
	model       *ApplyModuleModel
	doDestroy   bool
	speculative bool
}

type doRunOutput struct {
	moduleVersion string
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
	ID            types.String        `tfsdk:"id"`
	WorkspacePath types.String        `tfsdk:"workspace_path"`
	ModuleSource  types.String        `tfsdk:"module_source"`
	ModuleVersion types.String        `tfsdk:"module_version"`
	Variables     basetypes.ListValue `tfsdk:"variables"`
	RunVariables  basetypes.ListValue `tfsdk:"run_variables"`
	Speculative   types.Bool          `tfsdk:"speculative"`
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
				Computed:            true, // When not passed it, it needs to be set by Create.
			},
			"run_variables": schema.ListNestedAttribute{
				MarkdownDescription: "The variables that were used by the run.",
				Description:         "The variables that were used by the run.",
				Computed:            true,
				PlanModifiers: []planmodifier.List{
					modifiers.ListDefault([]attr.Value{}),
				},
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

	// Pass in Speculative if supplied.
	speculative := false
	if !applyModule.Speculative.IsNull() {
		speculative = applyModule.Speculative.ValueBool()
	}

	// Do plan and apply, no destroy.
	var didRun doRunOutput
	resp.Diagnostics.Append(t.doRun(ctx, &doRunInput{
		model:       &applyModule,
		speculative: speculative,
	}, &didRun)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update the plan with the computed ID.
	applyModule.ID = types.StringValue(uuid.New().String())
	applyModule.ModuleVersion = types.StringValue(didRun.moduleVersion)
	applyModule.Speculative = types.BoolValue(speculative) // has to be consistent with inputs

	// Add namespace paths to the variables.
	outVars, diags := t.addNamespacePaths(ctx, &applyModule.Variables, applyModule.WorkspacePath.ValueString())
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	applyModule.RunVariables = *outVars

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

	// not available from currentApplied; default it to false.
	if state.Speculative.IsUnknown() {
		state.Speculative = types.BoolValue(false)
	}

	// Add namespace paths to the variables.
	outVars, diags := t.addNamespacePaths(ctx, &state.Variables, state.WorkspacePath.ValueString())
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	state.RunVariables = *outVars

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
	var didRun doRunOutput
	resp.Diagnostics.Append(t.doRun(ctx, &doRunInput{
		model:       &plan,
		speculative: plan.Speculative.ValueBool(),
	}, &didRun)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Capture the module version in case it changed.
	plan.ModuleVersion = types.StringValue(didRun.moduleVersion)

	// not available from didRun; convert null or unknown to false.
	plan.Speculative = types.BoolValue(plan.Speculative.ValueBool())

	// Add namespace paths to the variables.
	outVars, diags := t.addNamespacePaths(ctx, &plan.Variables, plan.WorkspacePath.ValueString())
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	plan.RunVariables = *outVars

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

	// Refuse to destroy if a configuration version was deployed by the latest run
	// (as measured by lack of a module source).
	if currentApplied.moduleSource == nil {
		resp.Diagnostics.AddError("Workspace's latest run had deployed a configuration version, will not delete", "")
		return
	}

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
	resp.Diagnostics.Append(t.doRun(ctx, &doRunInput{
		model:     &state,
		doDestroy: true,
	}, nil)...) // nil means no module version output is wanted
	if resp.Diagnostics.HasError() {
		return
	}
}

// doRun launches a remote run and waits for it to complete.
func (t *applyModuleResource) doRun(ctx context.Context,
	input *doRunInput, output *doRunOutput,
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
		diags.AddError("Plan failed", string(plannedRun.Status))
		return diags
	}
	if plannedRun.Plan.Status != "finished" {
		diags.AddError("Plan failed", string(plannedRun.Plan.Status))
		return diags
	}

	// Capture the run ID.
	runID := plannedRun.Metadata.ID

	if plannedRun.Status == "planned_and_finished" {
		if (output != nil) && (plannedRun.ModuleVersion != nil) {
			*output = doRunOutput{
				moduleVersion: *plannedRun.ModuleVersion,
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

	if (output != nil) && (finishedRun.ModuleVersion != nil) {
		*output = doRunOutput{
			moduleVersion: *finishedRun.ModuleVersion,
		}
	}
	return nil
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
		} else {
			// Current state has no run ID, so it must have been manually updated.
			moduleInfoOutput.wasManualUpdate = true
		}
	}

	return nil
}

// addNamespacePaths converts from TF-Provider typed variables, adds namespace paths, and converts back.
func (t *applyModuleResource) addNamespacePaths(ctx context.Context,
	inputs *basetypes.ListValue, namespacePath string) (*basetypes.ListValue, diag.Diagnostics) {
	diags := diag.Diagnostics{}

	// Convert variables to SDK types to prepare to add namespace paths.
	sdkVariables, err := t.copyRunVariablesToInput(ctx, inputs)
	if err != nil {
		diags.AddError("Failed to convert variables to SDK types", err.Error())
		return nil, diags
	}

	// Add namespace paths to the variables.
	copies := make([]sdktypes.RunVariable, len(sdkVariables))
	for _, sdkVariable := range sdkVariables {
		localCopy := sdkVariable
		newPath := namespacePath + "/" + localCopy.Key
		localCopy.NamespacePath = &newPath
		copies = append(copies, localCopy)
	}

	// Convert back to TF-Provider typed variables.
	result, diags := t.toProviderOutputVariables(ctx, copies)
	if diags.HasError() {
		return nil, diags
	}

	return &result, diags
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
