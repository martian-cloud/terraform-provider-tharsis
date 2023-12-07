package provider

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	tharsis "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg"
	ttypes "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

// TerraformProviderModel is the model for a Terraform provider.
type TerraformProviderModel struct {
	ID                types.String `tfsdk:"id"`
	LastUpdated       types.String `tfsdk:"last_updated"`
	Name              types.String `tfsdk:"name"`
	GroupPath         types.String `tfsdk:"group_path"`
	ResourcePath      types.String `tfsdk:"resource_path"`
	RegistryNamespace types.String `tfsdk:"registry_namespace"`
	RepositoryURL     types.String `tfsdk:"repository_url"`
	Private           types.Bool   `tfsdk:"private"`
}

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ resource.Resource                = (*terraformProviderResource)(nil)
	_ resource.ResourceWithConfigure   = (*terraformProviderResource)(nil)
	_ resource.ResourceWithImportState = (*terraformProviderResource)(nil)
)

// NewTerraformProviderResource is a helper function to simplify the provider implementation.
func NewTerraformProviderResource() resource.Resource {
	return &terraformProviderResource{}
}

type terraformProviderResource struct {
	client *tharsis.Client
}

// Metadata returns the full name of the resource, including prefix, underscore, instance name.
func (t *terraformProviderResource) Metadata(_ context.Context,
	_ resource.MetadataRequest, resp *resource.MetadataResponse,
) {
	resp.TypeName = "tharsis_terraform_provider"
}

func (t *terraformProviderResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	description := "Defines and manages a Terraform provider."

	resp.Schema = schema.Schema{
		Version:             1,
		MarkdownDescription: description,
		Description:         description,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "String identifier of the Terraform provider.",
				Description:         "String identifier of the Terraform provider.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the Terraform provider.",
				Description:         "The name of the Terraform provider.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"group_path": schema.StringAttribute{
				MarkdownDescription: "The path of the group where this Terraform provider resides.",
				Description:         "The path of the group where this Terraform provider resides.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"resource_path": schema.StringAttribute{
				MarkdownDescription: "String identifier of this Terraform provider.",
				Description:         "String identifier of this Terraform provider.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"registry_namespace": schema.StringAttribute{
				MarkdownDescription: "The top-level group where this Terraform provider resides.",
				Description:         "The top-level group where this Terraform provider resides.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"repository_url": schema.StringAttribute{
				MarkdownDescription: "The repository URL where this Terraform provider can be found.",
				Description:         "The repository URL where this Terraform provider can be found.",
				Optional:            true,
				// Can be updated in place, so no RequiresReplace plan modifier.
			},
			"private": schema.BoolAttribute{
				MarkdownDescription: "Whether this Terraform provider is hidden from other top-level groups.",
				Description:         "Whether this Terraform provider is hidden from other top-level groups.",
				Optional:            true,
				// Can be updated in place, so no RequiresReplace plan modifier.
			},
			"last_updated": schema.StringAttribute{
				MarkdownDescription: "Timestamp when this Terraform provider was most recently updated.",
				Description:         "Timestamp when this Terraform provider was most recently updated.",
				Computed:            true,
			},
		},
	}
}

// Configure lets the provider implement the ResourceWithConfigure interface.
func (t *terraformProviderResource) Configure(_ context.Context,
	req resource.ConfigureRequest, _ *resource.ConfigureResponse,
) {
	if req.ProviderData == nil {
		return
	}
	t.client = req.ProviderData.(*tharsis.Client)
}

func (t *terraformProviderResource) Create(ctx context.Context,
	req resource.CreateRequest, resp *resource.CreateResponse,
) {
	// Retrieve values from Terraform provider.
	var terraformProvider TerraformProviderModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &terraformProvider)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create the Terraform provider.
	created, err := t.client.TerraformProvider.CreateProvider(ctx,
		&ttypes.CreateTerraformProviderInput{
			Name:          terraformProvider.Name.ValueString(),
			GroupPath:     terraformProvider.GroupPath.ValueString(),
			RepositoryURL: terraformProvider.RepositoryURL.ValueString(),
			Private:       terraformProvider.Private.ValueBool(),
		})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Terraform provider",
			err.Error(),
		)
		return
	}

	// Map the response body to the schema and update the plan with the computed attribute values.
	t.copyTerraformProvider(*created, &terraformProvider)

	// Set the response state to the fully-populated plan, whether or not there is an error.
	resp.Diagnostics.Append(resp.State.Set(ctx, terraformProvider)...)
}

func (t *terraformProviderResource) Read(ctx context.Context,
	req resource.ReadRequest, resp *resource.ReadResponse,
) {
	// Get the current state.
	var state TerraformProviderModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the Terraform provider from Tharsis.
	found, err := t.client.TerraformProvider.GetProvider(ctx, &ttypes.GetTerraformProviderInput{
		ID: state.ID.ValueString(),
	})
	if err != nil {
		if tharsis.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error reading Terraform provider",
			err.Error(),
		)
		return
	}

	// Copy the from-Tharsis struct to the state.
	t.copyTerraformProvider(*found, &state)

	// Set the refreshed state, whether or not there is an error.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (t *terraformProviderResource) Update(ctx context.Context,
	req resource.UpdateRequest, resp *resource.UpdateResponse,
) {
	// Retrieve values from plan.
	var plan TerraformProviderModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update the Terraform provider via Tharsis.
	// The ID is used to find the record to update.
	updated, err := t.client.TerraformProvider.UpdateProvider(ctx,
		&ttypes.UpdateTerraformProviderInput{
			ID:            plan.ID.ValueString(),
			RepositoryURL: plan.RepositoryURL.ValueString(),
			Private:       plan.Private.ValueBool(),
		})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Terraform provider",
			err.Error(),
		)
		return
	}

	// Copy all fields returned by Tharsis back into the plan.
	t.copyTerraformProvider(*updated, &plan)

	// Set the response state to the fully-populated plan, with or without error.
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (t *terraformProviderResource) Delete(ctx context.Context,
	req resource.DeleteRequest, resp *resource.DeleteResponse,
) {
	// Get the current state.
	var state TerraformProviderModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the Terraform provider via Tharsis.
	_, err := t.client.TerraformProvider.DeleteProvider(ctx,
		&ttypes.DeleteTerraformProviderInput{
			ID: state.ID.ValueString(),
		})
	if err != nil {

		// Handle the case that the Terraform provider no longer exists.
		if tharsis.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error deleting Terraform provider",
			err.Error(),
		)
	}
}

// ImportState helps the provider implement the ResourceWithImportState interface.
func (t *terraformProviderResource) ImportState(ctx context.Context,
	req resource.ImportStateRequest, resp *resource.ImportStateResponse,
) {
	// Retrieve import ID and save to id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// copyTerraformProvider copies the contents of a Terraform provider.
// It is intended to copy from a struct returned by Tharsis to a Terraform plan or state.
func (t *terraformProviderResource) copyTerraformProvider(src ttypes.TerraformProvider, dest *TerraformProviderModel) {
	dest.ID = types.StringValue(src.Metadata.ID)
	dest.Name = types.StringValue(src.Name)
	dest.GroupPath = types.StringValue(src.GroupPath)
	dest.ResourcePath = types.StringValue(src.ResourcePath)
	dest.RegistryNamespace = types.StringValue(src.RegistryNamespace)
	dest.RepositoryURL = types.StringValue(src.RepositoryURL)
	dest.Private = types.BoolValue(src.Private)

	// Must use time value from SDK/API.  Using time.Now() is not reliable.
	dest.LastUpdated = types.StringValue(src.Metadata.LastUpdatedTimestamp.Format(time.RFC850))
}
