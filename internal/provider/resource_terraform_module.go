package provider

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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

	// FIXME: Remove this:
	tattle(fmt.Sprintf("*** in NewTerraformModuleResource\n"))

	return &terraformModuleResource{}
}

type terraformModuleResource struct {
	client *tharsis.Client
}

// Metadata returns the full name of the resource, including prefix, underscore, instance name.
func (t *terraformModuleResource) Metadata(ctx context.Context,
	req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "tharsis_terraform_module"

	// FIXME: Remove this:
	tattle(fmt.Sprintf("*** in Metadata: resp: %#v\n", resp))

}

// The diagnostics return value is required by the interface even though this function returns only nil.
func (t *terraformModuleResource) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	description := "Defines and manages a terraform module."

	// FIXME: Remove this:
	tattle(fmt.Sprintf("*** in GetSchema.\n"))

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

	// FIXME: Remove this:
	tattle(fmt.Sprintf("*** in Configure.\n"))

	if req.ProviderData == nil {
		return
	}
	t.client = req.ProviderData.(*tharsis.Client)
}

// FIXME: Remove this:
func tattle(s string) {
	p := filepath.Join("/home/rrichesjr/projects/martian-cloud/terraform-provider-tharsis", "z.log")
	fh, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		panic(fmt.Errorf("failed to open tattle file"))
	}
	defer fh.Close()
	if _, err = fh.WriteString(s); err != nil {
		panic(fmt.Errorf("failed to write to tattle file"))
	}
}

func (t *terraformModuleResource) Create(ctx context.Context,
	req resource.CreateRequest, resp *resource.CreateResponse) {

	// FIXME: Remove this:
	tattle(fmt.Sprintf("*** in Create.\n"))

	// Retrieve values from terraform module.
	var terraformModule TerraformModuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &terraformModule)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// FIXME: Remove this:
	tattle(fmt.Sprintf("*** to create: Name: %#v\n", terraformModule))
	tattle(fmt.Sprintf("*** to create: Name: %s\n", terraformModule.Name))
	tattle(fmt.Sprintf("*** to create: System: %s\n", terraformModule.System))
	tattle(fmt.Sprintf("*** to create: ResourcePath: %s\n", terraformModule.ResourcePath.ValueString()))
	tattle(fmt.Sprintf("*** to create: GroupPath: %s\n", terraformModule.GroupPath.ValueString()))
	tattle(fmt.Sprintf("*** to create: RepositoryURL: %s\n", terraformModule.RepositoryURL))
	tattle(fmt.Sprintf("*** to create: Private: %v\n", terraformModule.Private))

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

	// FIXME: Remove this:
	tattle(fmt.Sprintf("*** in Create: final state: %#v\n", terraformModule))
	tattle(fmt.Sprintf("*** in Create: final state system: %s\n", terraformModule.System.ValueString()))
	tattle(fmt.Sprintf("*** in Create: final state resource: %s\n", terraformModule.ResourcePath.ValueString()))

	// Set the response state to the fully-populated plan, whether or not there is an error.
	resp.Diagnostics.Append(resp.State.Set(ctx, terraformModule)...)
}

func (t *terraformModuleResource) Read(ctx context.Context,
	req resource.ReadRequest, resp *resource.ReadResponse) {

	// FIXME: Remove this:
	tattle(fmt.Sprintf("*** starting Read.\n"))

	// Get the current state.
	var state TerraformModuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// FIXME: Remove this:
	tattle(fmt.Sprintf("*** in Read: initial state: %#v\n", state))
	tattle(fmt.Sprintf("*** in Read: initial state system: %s\n", state.System.ValueString()))
	tattle(fmt.Sprintf("*** in Read: initial state resource: %s\n", state.ResourcePath.ValueString()))

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

	// FIXME: Remove this:
	tattle(fmt.Sprintf("*** in Read: found: %#v\n", found))
	tattle(fmt.Sprintf("*** in Read: found system: %s\n", found.System))
	tattle(fmt.Sprintf("*** in Read: found resource: %s\n", found.ResourcePath))

	// Copy the from-Tharsis struct to the state.
	t.copyTerraformModule(*found, &state)

	// FIXME: Remove this:
	tattle(fmt.Sprintf("*** in Read: final state: %#v\n", state))
	tattle(fmt.Sprintf("*** in Read: final state system: %s\n", state.System.ValueString()))
	tattle(fmt.Sprintf("*** in Read: final state resource: %s\n", state.ResourcePath.ValueString()))

	// Set the refreshed state, whether or not there is an error.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (t *terraformModuleResource) Update(ctx context.Context,
	req resource.UpdateRequest, resp *resource.UpdateResponse) {

	// FIXME: Remove this:
	tattle(fmt.Sprintf("*** in Update.\n"))

	// Retrieve values from plan.
	var plan TerraformModuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// FIXME: Remove this:
	tattle(fmt.Sprintf("*** in Update: initial plan: %#v\n", plan))
	tattle(fmt.Sprintf("*** in Update: initial plan system: %s\n", plan.System.ValueString()))
	tattle(fmt.Sprintf("*** in Update: initial plan resource: %s\n", plan.ResourcePath.ValueString()))

	// Update the terraform module via Tharsis.
	// The ID is used to find the record to update.
	updated, err := t.client.TerraformModule.UpdateModule(ctx,
		&ttypes.UpdateTerraformModuleInput{
			ID:            plan.ID.ValueString(),
			Name:          ptr.String(plan.Name.ValueString()),
			System:        ptr.String(plan.System.ValueString()),
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

	// FIXME: Remove this:
	tattle(fmt.Sprintf("*** in Update: updated: %#v\n", updated))
	tattle(fmt.Sprintf("*** in Update: updated system: %s\n", updated.System))
	tattle(fmt.Sprintf("*** in Update: updated resource: %s\n", updated.ResourcePath))

	// Copy all fields returned by Tharsis back into the plan.
	t.copyTerraformModule(*updated, &plan)

	// FIXME: Remove this:
	tattle(fmt.Sprintf("*** in Update: final plan: %#v\n", plan))
	tattle(fmt.Sprintf("*** in Update: final plan group: %s\n", plan.System.ValueString()))
	tattle(fmt.Sprintf("*** in Update: final plan resource: %s\n", plan.ResourcePath.ValueString()))

	// Set the response state to the fully-populated plan, with or without error.
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (t *terraformModuleResource) Delete(ctx context.Context,
	req resource.DeleteRequest, resp *resource.DeleteResponse) {

	// FIXME: Remove this:
	tattle(fmt.Sprintf("*** in Delete.\n"))

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

	// FIXME: Remove this:
	tattle(fmt.Sprintf("*** in ImportState: req.ID: %s.\n", req.ID))

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

	// FIXME: Remove this:
	tattle(fmt.Sprintf("*** in copyTerraformModule.\n"))

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
