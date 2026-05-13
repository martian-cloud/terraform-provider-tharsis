package provider

import (
	"context"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/client"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TerraformModuleModel is the model for a Terraform module.
type TerraformModuleModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	System            types.String `tfsdk:"system"`
	GroupPath         types.String `tfsdk:"group_path"`
	GroupID           types.String `tfsdk:"group_id"`
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
	client *client.GRPCClient
}

// Metadata returns the full name of the resource, including prefix, underscore, instance name.
func (t *terraformModuleResource) Metadata(_ context.Context,
	_ resource.MetadataRequest, resp *resource.MetadataResponse,
) {
	resp.TypeName = "tharsis_terraform_module"
}

func (t *terraformModuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	description := "Defines and manages a Terraform module."

	resp.Schema = schema.Schema{
		Version:             1,
		MarkdownDescription: description,
		Description:         description,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "String identifier of the Terraform module.",
				Description:         "String identifier of the Terraform module.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the Terraform module.",
				Description:         "The name of the Terraform module.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"system": schema.StringAttribute{
				MarkdownDescription: "The target system for the module (e.g. aws, azure, etc.).",
				Description:         "The target system for the module (e.g. aws, azure, etc.).",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"group_path": schema.StringAttribute{
				MarkdownDescription: "The group path for this module.",
				Description:         "The group path for this module.",
				Optional:            true,
				DeprecationMessage:  "Use group_id instead. This field will be removed in a future version.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"group_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the parent group.",
				Description:         "The ID of the parent group.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"resource_path": schema.StringAttribute{
				MarkdownDescription: "The path of the parent namespace plus the name of the terraform module.",
				Description:         "The path of the parent namespace plus the name of the terraform module.",
				Computed:            true,
				DeprecationMessage:  "Use the id field instead. This field will be removed in a future version.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"registry_namespace": schema.StringAttribute{
				MarkdownDescription: "The top-level group in which this module resides.",
				Description:         "The top-level group in which this module resides.",
				Computed:            true,
				DeprecationMessage:  "This field will be removed in a future version.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"repository_url": schema.StringAttribute{
				MarkdownDescription: "The URL in a repository where this module is found.",
				Description:         "The URL in a repository where this module is found.",
				Optional:            true,
				// Can be updated in place, so no RequiresReplace plan modifier.
			},
			"private": schema.BoolAttribute{
				MarkdownDescription: "Whether other groups are blocked from seeing this module.",
				Description:         "Whether other groups are blocked from seeing this module.",
				Optional:            true,
				// Can be updated in place, so no RequiresReplace plan modifier.
			},
			// Keep this:
			"last_updated": schema.StringAttribute{
				MarkdownDescription: "Timestamp when this terraform module was most recently updated.",
				Description:         "Timestamp when this terraform module was most recently updated.",
				Computed:            true,
			},
		},
	}
}

// Configure lets the provider implement the ResourceWithConfigure interface.
func (t *terraformModuleResource) Configure(_ context.Context,
	req resource.ConfigureRequest, _ *resource.ConfigureResponse,
) {
	if req.ProviderData == nil {
		return
	}
	t.client = req.ProviderData.(*client.GRPCClient)
}

func (t *terraformModuleResource) Create(ctx context.Context,
	req resource.CreateRequest, resp *resource.CreateResponse,
) {
	// Retrieve values from Terraform module.
	var terraformModule TerraformModuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &terraformModule)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var groupID string
	if v := terraformModule.GroupID.ValueString(); v != "" {
		groupID = v
	} else if v := terraformModule.GroupPath.ValueString(); v != "" {
		groupID = trn.TypeGroup.Build(v)
	} else {
		resp.Diagnostics.AddError("Either group_id or group_path must be specified", "")
		return
	}

	created, err := t.client.TerraformModulesClient.CreateTerraformModule(ctx,
		&pb.CreateTerraformModuleRequest{
			Name:          terraformModule.Name.ValueString(),
			System:        terraformModule.System.ValueString(),
			GroupId:       groupID,
			RepositoryUrl: terraformModule.RepositoryURL.ValueString(),
			Private:       terraformModule.Private.ValueBool(),
		})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Terraform module",
			err.Error(),
		)
		return
	}

	// Map the response body to the schema and update the plan with the computed attribute values.
	t.copyTerraformModule(created, &terraformModule)

	// Set the response state to the fully-populated plan, whether or not there is an error.
	resp.Diagnostics.Append(resp.State.Set(ctx, terraformModule)...)
}

func (t *terraformModuleResource) Read(ctx context.Context,
	req resource.ReadRequest, resp *resource.ReadResponse,
) {
	// Get the current state.
	var state TerraformModuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the Terraform module from Tharsis.
	found, err := t.client.TerraformModulesClient.GetTerraformModuleByID(ctx,
		&pb.GetTerraformModuleByIDRequest{
			Id: state.ID.ValueString(),
		})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading Terraform module",
			err.Error(),
		)
		return
	}

	// Copy the from-Tharsis struct to the state.
	t.copyTerraformModule(found, &state)

	// Set the refreshed state, whether or not there is an error.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (t *terraformModuleResource) Update(ctx context.Context,
	req resource.UpdateRequest, resp *resource.UpdateResponse,
) {
	// Retrieve values from plan.
	var plan TerraformModuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update the Terraform module via Tharsis.
	// The ID is used to find the record to update.
	updated, err := t.client.TerraformModulesClient.UpdateTerraformModule(ctx,
		&pb.UpdateTerraformModuleRequest{
			Id:            plan.ID.ValueString(),
			RepositoryUrl: new(plan.RepositoryURL.ValueString()),
			Private:       new(plan.Private.ValueBool()),
		})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Terraform module",
			err.Error(),
		)
		return
	}

	// Copy all fields returned by Tharsis back into the plan.
	t.copyTerraformModule(updated, &plan)

	// Set the response state to the fully-populated plan, with or without error.
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (t *terraformModuleResource) Delete(ctx context.Context,
	req resource.DeleteRequest, resp *resource.DeleteResponse,
) {
	// Get the current state.
	var state TerraformModuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the Terraform module via Tharsis.
	_, err := t.client.TerraformModulesClient.DeleteTerraformModule(ctx,
		&pb.DeleteTerraformModuleRequest{
			Id: state.ID.ValueString(),
		})
	if err != nil {
		// Handle the case that the Terraform module no longer exists.
		if status.Code(err) == codes.NotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error deleting Terraform module",
			err.Error(),
		)
	}
}

// ImportState helps the provider implement the ResourceWithImportState interface.
func (t *terraformModuleResource) ImportState(ctx context.Context,
	req resource.ImportStateRequest, resp *resource.ImportStateResponse,
) {
	// Retrieve import ID and save to id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// copyTerraformModule copies the contents of a Terraform module.
// It is intended to copy from a struct returned by Tharsis to a Terraform plan or state.
func (t *terraformModuleResource) copyTerraformModule(src *pb.TerraformModule, dest *TerraformModuleModel) {
	parsed := trn.MustParseAny(src.Metadata.Trn)
	dest.ID = types.StringValue(src.Metadata.Id)
	dest.Name = types.StringValue(src.Name)
	dest.System = types.StringValue(src.System)
	// TRN path is <group_path>/<name>/<system>, strip last two segments for group path.
	resourcePath := parsed.Path()
	suffix := "/" + src.Name + "/" + src.System
	dest.GroupPath = types.StringValue(strings.TrimSuffix(resourcePath, suffix))
	dest.GroupID = types.StringValue(src.GroupId)
	dest.ResourcePath = types.StringValue(parsed.Path())
	if parts := parsed.PathParts(); len(parts) > 0 {
		dest.RegistryNamespace = types.StringValue(parts[0])
	}
	dest.RepositoryURL = types.StringValue(src.RepositoryUrl)
	dest.Private = types.BoolValue(src.Private)

	// Must use time value from SDK/API.  Using time.Now() is not reliable.
	dest.LastUpdated = types.StringValue(src.Metadata.UpdatedAt.AsTime().Format(time.RFC850))
}
