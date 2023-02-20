package provider

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	tharsis "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg"
	ttypes "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

// VariableModel is the model for a namespace variable.
// Fields intentionally omitted: NamespaceMemberships and ActivityEvents.
type VariableModel struct {
	ID            types.String `tfsdk:"id"`
	NamespacePath types.String `tfsdk:"namespace_path"`
	Category      types.String `tfsdk:"category"`
	Key           types.String `tfsdk:"key"`
	Value         types.String `tfsdk:"value"`
	Hcl           types.Bool   `tfsdk:"hcl"`
}

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ resource.Resource                = (*variableResource)(nil)
	_ resource.ResourceWithConfigure   = (*variableResource)(nil)
	_ resource.ResourceWithImportState = (*variableResource)(nil)
)

// NewVariableResource is a helper function to simplify the provider implementation.
func NewVariableResource() resource.Resource {
	return &variableResource{}
}

type variableResource struct {
	client *tharsis.Client
}

// Metadata returns the full name of the resource, including prefix, underscore, instance name.
func (t *variableResource) Metadata(ctx context.Context,
	req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "tharsis_variable"
}

func (t *variableResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	description := "Defines and manages a namespace variable."

	resp.Schema = schema.Schema{
		Version:             1,
		MarkdownDescription: description,
		Description:         description,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "String identifier of the namespace variable.",
				Description:         "String identifier of the namespace variable.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"namespace_path": schema.StringAttribute{
				MarkdownDescription: "The path to this variable's namespace.",
				Description:         "The path to this variable's namespace.",
				Required:            true,
			},
			"category": schema.StringAttribute{
				MarkdownDescription: "Whether this variable is a Terraform or an environment variable.",
				Description:         "Whether this variable is a Terraform or an environment variable.",
				Required:            true,
			},
			"hcl": schema.BoolAttribute{
				MarkdownDescription: "Whether this variable has an HCL value.",
				Description:         "Whether this variable has an HCL value.",
				Required:            true,
			},
			"key": schema.StringAttribute{
				MarkdownDescription: "This variable's key (within its namespace).",
				Description:         "This variable's key (within its namespace).",
				Required:            true,
			},
			"value": schema.StringAttribute{
				MarkdownDescription: "This variable's value.",
				Description:         "This variable's value.",
				Required:            true,
			},
		},
	}
}

// Configure lets the provider implement the ResourceWithConfigure interface.
func (t *variableResource) Configure(_ context.Context,
	req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	t.client = req.ProviderData.(*tharsis.Client)
}

func (t *variableResource) Create(ctx context.Context,
	req resource.CreateRequest, resp *resource.CreateResponse) {

	// Retrieve values from namespace variable.
	var variable VariableModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &variable)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create the namespace variable.
	created, err := t.client.Variable.CreateVariable(ctx,
		&ttypes.CreateNamespaceVariableInput{
			NamespacePath: variable.NamespacePath.ValueString(),
			Category:      ttypes.VariableCategory(variable.Category.ValueString()),
			HCL:           variable.Hcl.ValueBool(),
			Key:           variable.Key.ValueString(),
			Value:         variable.Value.ValueString(),
		})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating namespace variable",
			err.Error(),
		)
		return
	}

	// Map the response body to the schema and update the plan with the computed attribute values.
	// Because the schema uses the Set type rather than the List type, make sure to set all fields.
	if err = t.copyVariable(*created, &variable); err != nil {
		resp.Diagnostics.AddError(
			"Error setting state for variable",
			err.Error(),
		)
		return
	}

	// Set the response state to the fully-populated plan, whether or not there is an error.
	resp.Diagnostics.Append(resp.State.Set(ctx, variable)...)
}

func (t *variableResource) Read(ctx context.Context,
	req resource.ReadRequest, resp *resource.ReadResponse) {

	// Get the current state.
	var state VariableModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the namespace variable from Tharsis.
	found, err := t.client.Variable.GetVariable(ctx, &ttypes.GetNamespaceVariableInput{
		ID: state.ID.ValueString(),
	})
	if err != nil {
		if tharsis.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error reading namespace variable",
			err.Error(),
		)
		return
	}

	// Copy the from-Tharsis struct to the state.
	if err = t.copyVariable(*found, &state); err != nil {
		resp.Diagnostics.AddError(
			"Error setting state for variable",
			err.Error(),
		)
		return
	}

	// Set the refreshed state, whether or not there is an error.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (t *variableResource) Update(ctx context.Context,
	req resource.UpdateRequest, resp *resource.UpdateResponse) {

	// Retrieve values from plan.
	var plan VariableModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update the namespace variable via Tharsis.
	// The ID is used to find the record to update.
	// The description is modified.
	updated, err := t.client.Variable.UpdateVariable(ctx,
		&ttypes.UpdateNamespaceVariableInput{
			ID:    plan.ID.ValueString(),
			HCL:   plan.Hcl.ValueBool(),
			Key:   plan.Key.ValueString(),
			Value: plan.Value.ValueString(),
		})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating namespace variable",
			err.Error(),
		)
		return
	}

	// Copy all fields returned by Tharsis back into the plan.
	if err = t.copyVariable(*updated, &plan); err != nil {
		resp.Diagnostics.AddError(
			"Error setting state for variable",
			err.Error(),
		)
		return
	}

	// Set the response state to the fully-populated plan, with or without error.
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (t *variableResource) Delete(ctx context.Context,
	req resource.DeleteRequest, resp *resource.DeleteResponse) {

	// Get the current state.
	var state VariableModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the namespace variable via Tharsis.
	err := t.client.Variable.DeleteVariable(ctx,
		&ttypes.DeleteNamespaceVariableInput{
			ID: state.ID.ValueString(),
		})
	if err != nil {

		// Handle the case that the namespace variable no longer exists.
		if tharsis.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error deleting namespace variable",
			err.Error(),
		)
	}
}

// ImportState helps the provider implement the ResourceWithImportState interface.
func (t *variableResource) ImportState(ctx context.Context,
	req resource.ImportStateRequest, resp *resource.ImportStateResponse) {

	// Retrieve import ID and save to id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// copyVariable copies the contents of a namespace variable.
// It is intended to copy from a struct returned by Tharsis to a Terraform plan or state.
func (t *variableResource) copyVariable(src ttypes.NamespaceVariable, dest *VariableModel) error {
	if src.Value == nil {
		return errors.New("could not read variable value, ensure that you have the correct permissions to view this variable's value")
	}

	dest.ID = types.StringValue(src.Metadata.ID)
	dest.NamespacePath = types.StringValue(src.NamespacePath)
	dest.Category = types.StringValue(string(src.Category))
	dest.Hcl = types.BoolValue(src.HCL)
	dest.Key = types.StringValue(src.Key)
	dest.Value = types.StringValue(*src.Value)

	return nil
}

// The End.
