package provider

import (
	"context"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	tharsis "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg"
	ttypes "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

// TerraformModuleModel is the model for a terraform module.
type TerraformModuleModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	System            types.String `tfsdk:"system"`
	GroupPath         types.String `tfsdk:"group_path"`
	ResourcePath      types.String `tfsdk:"resource_path"`
	RegistryNamespace types.String `tfsdk:"registry_namespace"`
	RepositoryURL     types.String `tfsdk:"repository_url"`
	LastUpdated       types.String `tfsdk:"last_updated"`
	Private           types.Bool   `tfsdk:"private"`
}

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ resource.Resource                = (*terraformModuleResource)(nil)
	_ resource.ResourceWithConfigure   = (*terraformModuleResource)(nil)
	_ resource.ResourceWithImportState = (*terraformModuleResource)(nil)
)

// NewTerraformModuleResource is a helper function to simplify the provider implementation.
func NewTerraformModuleResource() resource.Resource {
	return &terraformModuleResource{}
}

type terraformModuleResource struct {
	client *tharsis.Client
}

// Metadata returns the full name of the resource, including prefix, underscore, instance name.
func (t *terraformModuleResource) Metadata(ctx context.Context,
	req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "tharsis_terraform_module"
}

// The diagnostics return value is required by the interface even though this function returns only nil.
func (t *terraformModuleResource) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	description := "Defines and manages a terraform module."

	return tfsdk.Schema{
		Version: 1,

		MarkdownDescription: description,
		Description:         description,

		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:                types.StringType,
				MarkdownDescription: "String identifier of the terraform module.",
				Description:         "String identifier of the terraform module.",
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					resource.UseStateForUnknown(),
				},
			},
			"name": {
				Type:                types.StringType,
				MarkdownDescription: "The name of the terraform module.",
				Description:         "The name of the terraform module.",
				Required:            true,
			},
			"system": {
				Type:                types.StringType,
				MarkdownDescription: "The target system for the module (e.g. aws, azure, etc.).",
				Description:         "The target system for the module (e.g. aws, azure, etc.).",
				Required:            true,
			},
			"group_path": {
				Type:                types.StringType,
				MarkdownDescription: "The group path for this module.",
				Description:         "The group path for this module.",
				Required:            true,
			},
			"resource_path": {
				Type:                types.StringType,
				MarkdownDescription: "The path of the parent namespace plus the name of the terraform module.",
				Description:         "The path of the parent namespace plus the name of the terraform module.",
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					resource.UseStateForUnknown(),
				},
			},
			"registry_namespace": {
				Type:                types.StringType,
				MarkdownDescription: "The top-level group in which this module resides.",
				Description:         "The top-level group in which this module resides.",
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					resource.UseStateForUnknown(),
				},
			},
			"repository_url": {
				Type:                types.StringType,
				MarkdownDescription: "The URL in a repository where this module is found.",
				Description:         "The URL in a repository where this module is found.",
				Optional:            true,
			},
			"private": {
				Type:                types.BoolType,
				MarkdownDescription: "Whether other groups are blocked from seeing this module.",
				Description:         "Whether other groups are blocked from seeing this module.",
				Optional:            true,
			},
			// Keep this:
			"last_updated": {
				Type:                types.StringType,
				MarkdownDescription: "Timestamp when this terraform module was most recently updated.",
				Description:         "Timestamp when this terraform module was most recently updated.",
				Computed:            true,
			},
		},
	}, nil
}

// Configure lets the provider implement the ResourceWithConfigure interface.
func (t *terraformModuleResource) Configure(_ context.Context,
	req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	t.client = req.ProviderData.(*tharsis.Client)
}

func (t *terraformModuleResource) Create(ctx context.Context,
	req resource.CreateRequest, resp *resource.CreateResponse) {

	// Retrieve values from terraform module.
	var terraformModule TerraformModuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &terraformModule)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := t.client.TerraformModule.CreateModule(ctx,
		&ttypes.CreateTerraformModuleInput{
			Name:          terraformModule.Name.ValueString(),
			System:        terraformModule.System.ValueString(),
			GroupPath:     terraformModule.GroupPath.ValueString(),
			RepositoryURL: terraformModule.RepositoryURL.ValueString(),
			Private:       terraformModule.Private.ValueBool(),
		})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating terraform module",
			err.Error(),
		)
		return
	}

	// Map the response body to the schema and update the plan with the computed attribute values.
	t.copyTerraformModule(*created, &terraformModule)

	// Set the response state to the fully-populated plan, whether or not there is an error.
	resp.Diagnostics.Append(resp.State.Set(ctx, terraformModule)...)
}

func (t *terraformModuleResource) Read(ctx context.Context,
	req resource.ReadRequest, resp *resource.ReadResponse) {

	// Get the current state.
	var state TerraformModuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the terraform module from Tharsis.
	found, err := t.client.TerraformModule.GetModule(ctx, &ttypes.GetTerraformModuleInput{
		ID: ptr.String(state.ID.ValueString()),
	})
	if err != nil {
		if tharsis.NotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading terraform module",
			err.Error(),
		)
		return
	}

	// Copy the from-Tharsis struct to the state.
	t.copyTerraformModule(*found, &state)

	// Set the refreshed state, whether or not there is an error.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (t *terraformModuleResource) Update(ctx context.Context,
	req resource.UpdateRequest, resp *resource.UpdateResponse) {

	// Retrieve values from plan.
	var plan TerraformModuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update the terraform module via Tharsis.
	// The ID is used to find the record to update.
	updated, err := t.client.TerraformModule.UpdateModule(ctx,
		&ttypes.UpdateTerraformModuleInput{
			ID:            plan.ID.ValueString(),
			RepositoryURL: ptr.String(plan.RepositoryURL.ValueString()),
			Private:       ptr.Bool(plan.Private.ValueBool()),
		})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating terraform module",
			err.Error(),
		)
		return
	}

	// Copy all fields returned by Tharsis back into the plan.
	t.copyTerraformModule(*updated, &plan)

	// Set the response state to the fully-populated plan, with or without error.
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (t *terraformModuleResource) Delete(ctx context.Context,
	req resource.DeleteRequest, resp *resource.DeleteResponse) {

	// Get the current state.
	var state TerraformModuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the terraform module via Tharsis.
	err := t.client.TerraformModule.DeleteModule(ctx,
		&ttypes.DeleteTerraformModuleInput{
			ID: state.ID.ValueString(),
		})
	if err != nil {
		// Handle the case that the terraform module no longer exists.
		if tharsis.NotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error deleting terraform module",
			err.Error(),
		)
	}
}

// ImportState helps the provider implement the ResourceWithImportState interface.
func (t *terraformModuleResource) ImportState(ctx context.Context,
	req resource.ImportStateRequest, resp *resource.ImportStateResponse) {

	// Get the terraform module by full path from Tharsis.
	found, err := t.client.TerraformModule.GetModule(ctx, &ttypes.GetTerraformModuleInput{
		Path: ptr.String(req.ID),
	})
	if err != nil {
		if tharsis.NotFoundError(err) {
			resp.Diagnostics.AddError(
				"Import terraform module not found: "+req.ID,
				"",
			)
			return
		}
		resp.Diagnostics.AddError(
			"Import terraform module not found: "+req.ID,
			err.Error(),
		)
		return
	}

	// Import by resource path.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), found.Metadata.ID)...)
}

// copyTerraformModule copies the contents of a terraform module.
// It is intended to copy from a struct returned by Tharsis to a Terraform plan or state.
func (t *terraformModuleResource) copyTerraformModule(src ttypes.TerraformModule, dest *TerraformModuleModel) {
	dest.ID = types.StringValue(src.Metadata.ID)
	dest.Name = types.StringValue(src.Name)
	dest.System = types.StringValue(src.System)
	dest.GroupPath = types.StringValue(src.GroupPath)
	dest.ResourcePath = types.StringValue(src.ResourcePath)
	dest.RegistryNamespace = types.StringValue(src.RegistryNamespace)
	dest.RepositoryURL = types.StringValue(src.RepositoryURL)
	dest.Private = types.BoolValue(src.Private)
	dest.LastUpdated = types.StringValue(src.Metadata.LastUpdatedTimestamp.Format(time.RFC850))
}

// The End.
