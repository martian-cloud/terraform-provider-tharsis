package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/martian-cloud/terraform-provider-tharsis/internal/modifiers"
	tharsis "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg"
	sdktypes "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

type doRunInput struct {
	model     ApplyModuleModel
	doDestroy bool
}

const (
	jobCompletionPollInterval = 5 * time.Second
)

var (
	applyRunComment = "terraform-provider-tharsis" // must be var, not const, to take address
)

// RunVariableModel is used in apply modules to set Terraform and environment variables.
type RunVariableModel struct {
	Value         *string `tfsdk:"value"`
	NamespacePath *string `tfsdk:"namespace_path"`
	Key           string  `tfsdk:"key"`
	Category      string  `tfsdk:"category"`
	HCL           bool    `tfsdk:"hcl"`
}

// FromTerraform5Value converts a RunVariable from Terraform values to Go equivalent.
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

	err = v["namespace_path"].As(&e.NamespacePath)
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
	WorkspacePath types.String        `tfsdk:"workspace_path"`
	ModuleSource  types.String        `tfsdk:"module_source"`
	ModuleVersion types.String        `tfsdk:"module_version"`
	Variables     basetypes.ListValue `tfsdk:"variables"`
}

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ resource.Resource                = (*applyModuleResource)(nil)
	_ resource.ResourceWithConfigure   = (*applyModuleResource)(nil)
	_ resource.ResourceWithImportState = (*applyModuleResource)(nil)
)

// NewApplyModuleResource is a helper function to simplify the provider implementation.
func NewApplyModuleResource() resource.Resource {
	return &applyModuleResource{}
}

type applyModuleResource struct {
	client *tharsis.Client
}

// Metadata returns the full name of the resource, including prefix, underscore, instance name.
func (t *applyModuleResource) Metadata(ctx context.Context,
	req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "tharsis_apply_module"
}

func (t *applyModuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	description := "Defines and manages an apply module."

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
				Computed:            true, // Terraform requires it to be computed if it's optional.
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
					modifiers.ListDefault([]attr.Value{}),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"value": schema.StringAttribute{
							MarkdownDescription: "Value of the variable.",
							Description:         "Value of the variable.",
							Required:            true,
						},
						"namespace_path": schema.StringAttribute{
							MarkdownDescription: "Path of the host namespace for this variable.",
							Description:         "Path of the host namespace for this variable.",
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
							MarkdownDescription: "Whether this variable is HCL (vs. environment).",
							Description:         "Whether this variable is HCL (vs. environment).",
							Required:            true,
						},
					},
				},
			},
		},
	}
}

// Configure lets the provider implement the ResourceWithConfigure interface.
func (t *applyModuleResource) Configure(_ context.Context,
	req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	t.client = req.ProviderData.(*tharsis.Client)
}

func (t *applyModuleResource) Create(ctx context.Context,
	req resource.CreateRequest, resp *resource.CreateResponse) {

	// FIXME: Remove this:
	tflog.Info(ctx, "******** Create method starting.")

	// Retrieve values from apply module.
	var applyModule ApplyModuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &applyModule)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Do plan and apply, no destroy.
	var created ApplyModuleModel
	resp.Diagnostics.Append(t.doRun(ctx, &doRunInput{
		model: applyModule,
	}, &created)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update the plan with the computed attribute values.
	resp.Diagnostics.Append(t.copyApplyModule(ctx, &created, &applyModule)...)

	// Set the response state to the fully-populated plan, whether or not there is an error.
	resp.Diagnostics.Append(resp.State.Set(ctx, applyModule)...)
}

func (t *applyModuleResource) Read(ctx context.Context,
	req resource.ReadRequest, resp *resource.ReadResponse) {

	// FIXME: Remove this:
	tflog.Info(ctx, "******** Read method starting.")

	// Get the current state.
	var state ApplyModuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var applied ApplyModuleModel
	resp.Diagnostics.Append(t.getCurrentApplied(ctx, state, &applied)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update the state with the computed attribute values.
	resp.Diagnostics.Append(t.copyApplyModule(ctx, &applied, &state)...)

	// TODO: Eventually, when the API and SDK support speculative runs with a module source,
	// this should do a speculative run here to determine whether changes are needed.

	// Set the refreshed state, whether or not there is an error.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (t *applyModuleResource) Update(ctx context.Context,
	req resource.UpdateRequest, resp *resource.UpdateResponse) {

	// FIXME: Remove this:
	tflog.Info(ctx, "******** Update method starting.")

	// Retrieve values from plan.
	var plan ApplyModuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// TODO: Please note that when the API and SDK support speculative runs with a module source,
	// this will need to look at the results from the Read method's speculative run to determine
	// whether to do an update.  A way will have to be found to force Terraform to allow the update.

	// Do the run.
	var updated ApplyModuleModel
	resp.Diagnostics.Append(t.doRun(ctx, &doRunInput{
		model: plan,
	}, &updated)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Copy all fields returned by Tharsis back into the plan.
	resp.Diagnostics.Append(t.copyApplyModule(ctx, &updated, &plan)...)

	// Set the response state to the fully-populated plan, with or without error.
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (t *applyModuleResource) Delete(ctx context.Context,
	req resource.DeleteRequest, resp *resource.DeleteResponse) {

	// FIXME: Remove this:
	tflog.Info(ctx, "******** Delete method starting.")

	// Get the current state.
	var state ApplyModuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var applied ApplyModuleModel
	resp.Diagnostics.Append(t.getCurrentApplied(ctx, state, &applied)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If the module source or module version differs, error out.
	if state.ModuleSource != applied.ModuleSource {
		resp.Diagnostics.AddError("Module source differs, cannot delete", "")
		return
	}
	if state.ModuleVersion != applied.ModuleVersion {
		resp.Diagnostics.AddError("Module version differs, cannot delete", "")
		return
	}

	// The apply module is being deleted, so don't use the returned value.
	var deleted ApplyModuleModel
	resp.Diagnostics.Append(t.doRun(ctx, &doRunInput{
		model:     state,
		doDestroy: true,
	}, &deleted)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// ImportState helps the provider implement the ResourceWithImportState interface.
func (t *applyModuleResource) ImportState(ctx context.Context,
	req resource.ImportStateRequest, resp *resource.ImportStateResponse) {

	// FIXME: Remove this:
	tflog.Info(ctx, "******** ImportState method starting.")

	resp.Diagnostics.AddError(
		"Import of workspace is not supported.",
		"",
	)
}

// doRun launches a remote run and waits for it to complete.
func (t *applyModuleResource) doRun(ctx context.Context,
	input *doRunInput, output *ApplyModuleModel) diag.Diagnostics {
	var diags diag.Diagnostics

	// FIXME: Remove this:
	tflog.Info(ctx, "**************** doRun: starting", map[string]interface{}{"input": input})

	// Convert the run variables.
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
	// Using module registry path and version, so no ConfigurationVersionID.
	createdRun, err := t.client.Run.CreateRun(ctx, &sdktypes.CreateRunInput{
		WorkspacePath: input.model.WorkspacePath.ValueString(),
		IsDestroy:     input.doDestroy,
		ModuleSource:  ptr.String(input.model.ModuleSource.ValueString()),
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

	// Capture the run ID.
	runID := plannedRun.Metadata.ID

	// TODO: Also take this early return when the API and SDK support speculative runs and PlanOnly is implemented.

	if plannedRun.Status == "planned_and_finished" {
		// Return the output.
		output.WorkspacePath = types.StringValue(plannedRun.WorkspacePath)
		output.ModuleSource = types.StringValue(*plannedRun.ModuleSource)
		output.ModuleVersion = types.StringValue(*plannedRun.ModuleVersion)
		output.Variables = input.model.Variables // Cannot get variables back from a workspace or run, so pass them through.
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

	// Return the output.
	output.WorkspacePath = types.StringValue(finishedRun.WorkspacePath)
	output.ModuleSource = types.StringValue(*finishedRun.ModuleSource)
	output.ModuleVersion = types.StringValue(*finishedRun.ModuleVersion)
	output.Variables = input.model.Variables // Cannot get variables back from a workspace or run, so pass them through.
	return nil
}

func (t *applyModuleResource) waitForJobCompletion(ctx context.Context, jobID *string) error {
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

// getCurrentApplied returns an ApplyModuleModel reflecting what is currently applied.
func (t *applyModuleResource) getCurrentApplied(ctx context.Context,
	tfState ApplyModuleModel, target *ApplyModuleModel) diag.Diagnostics {
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

// copyApplyModule copies the contents of an apply module.
// It copies the fields from the same type, because there is not an apply module defined by Tharsis.
func (t *applyModuleResource) copyApplyModule(ctx context.Context, src, dest *ApplyModuleModel) diag.Diagnostics {
	dest.WorkspacePath = src.WorkspacePath
	dest.ModuleSource = src.ModuleSource
	dest.ModuleVersion = src.ModuleVersion
	dest.Variables = src.Variables

	// Make sure variables aren't unknown, because Terraform doesn't like that.
	var listDiags diag.Diagnostics
	if dest.Variables.IsUnknown() {
		dest.Variables, listDiags = basetypes.NewListValueFrom(ctx, basetypes.ObjectType{
			AttrTypes: map[string]attr.Type{
				"value":          types.StringType,
				"namespace_path": types.StringType,
				"key":            types.StringType,
				"category":       types.StringType,
				"hcl":            types.BoolType,
			},
		}, []types.Object{})
		if listDiags.HasError() {
			return listDiags
		}
	}

	return nil
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
			Value:         model.Value,
			NamespacePath: model.NamespacePath,
			Key:           model.Key,
			Category:      sdktypes.VariableCategory(model.Category),
			HCL:           model.HCL,
		})
	}

	// Terraform generally wants to see nil rather than an empty list.
	if len(result) == 0 {
		result = nil
	}

	return result, nil
}

// The End.
