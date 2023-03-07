package provider

import (
	"context"
	"reflect"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	tharsis "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg"
	ttypes "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

// ManagedIdentityAliasModel is the model for a managed identity alias.
type ManagedIdentityAliasModel struct {
	ID            types.String `tfsdk:"id"`
	ResourcePath  types.String `tfsdk:"resource_path"`
	Name          types.String `tfsdk:"name"`
	GroupPath     types.String `tfsdk:"group_path"`
	LastUpdated   types.String `tfsdk:"last_updated"`
	AliasSourceID types.String `tfsdk:"alias_source_id"`
}

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ resource.Resource                = (*managedIdentityAliasResource)(nil)
	_ resource.ResourceWithConfigure   = (*managedIdentityAliasResource)(nil)
	_ resource.ResourceWithImportState = (*managedIdentityAliasResource)(nil)
)

// NewManagedIdentityAliasResource is a helper function to simplify the provider implementation.
func NewManagedIdentityAliasResource() resource.Resource {
	return &managedIdentityAliasResource{}
}

type managedIdentityAliasResource struct {
	client *tharsis.Client
}

// Metadata returns the full name of the resource, including prefix, underscore, instance name.
func (t *managedIdentityAliasResource) Metadata(ctx context.Context,
	req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "tharsis_managed_identity_alias"
}

func (t *managedIdentityAliasResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	description := "Defines and manages a managed identity alias."

	resp.Schema = schema.Schema{
		Version:             1,
		MarkdownDescription: description,
		Description:         description,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "String identifier of the managed identity alias.",
				Description:         "String identifier of the managed identity alias.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"resource_path": schema.StringAttribute{
				MarkdownDescription: "The path of the parent group plus the name of the managed identity alias.",
				Description:         "The path of the parent group plus the name of the managed identity alias.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the managed identity alias.",
				Description:         "The name of the managed identity alias.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"group_path": schema.StringAttribute{
				MarkdownDescription: "Full path of the group where alias will be created.",
				Description:         "Full path of the group where alias will be created.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"last_updated": schema.StringAttribute{
				MarkdownDescription: "Timestamp when this managed identity alias was most recently updated.",
				Description:         "Timestamp when this managed identity alias was most recently updated.",
				Computed:            true,
			},
			"alias_source_id": schema.StringAttribute{
				MarkdownDescription: "ID of the managed identity being aliased",
				Description:         "ID of the managed identity being aliased",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

// Configure lets the provider implement the ResourceWithConfigure interface.
func (t *managedIdentityAliasResource) Configure(_ context.Context,
	req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	t.client = req.ProviderData.(*tharsis.Client)
}

func (t *managedIdentityAliasResource) Create(ctx context.Context,
	req resource.CreateRequest, resp *resource.CreateResponse) {

	// Retrieve values from managedIdentityAlias.
	var managedIdentityAlias ManagedIdentityAliasModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &managedIdentityAlias)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create the managed identity alias.
	created, err := t.client.ManagedIdentity.CreateManagedIdentityAlias(ctx,
		&ttypes.CreateManagedIdentityAliasInput{
			Name:          managedIdentityAlias.Name.ValueString(),
			AliasSourceID: ptr.String(managedIdentityAlias.AliasSourceID.ValueString()),
			GroupPath:     managedIdentityAlias.GroupPath.ValueString(),
		})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating managed identity alias",
			err.Error(),
		)
		return
	}

	// Map the response body to the schema and update the plan with the computed attribute values.
	// Because the schema uses the Set type rather than the List type, make sure to set all fields.
	if err = t.copyManagedIdentityAlias(*created, &managedIdentityAlias); err != nil {
		resp.Diagnostics.AddError(
			"Error setting state",
			err.Error(),
		)
		return
	}

	// Set the response state to the fully-populated plan, whether or not there is an error.
	resp.Diagnostics.Append(resp.State.Set(ctx, managedIdentityAlias)...)
}

func (t *managedIdentityAliasResource) Read(ctx context.Context,
	req resource.ReadRequest, resp *resource.ReadResponse) {

	// Get the current state.
	var state ManagedIdentityAliasModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the managed identity from Tharsis.
	found, err := t.client.ManagedIdentity.GetManagedIdentity(ctx, &ttypes.GetManagedIdentityInput{
		ID: state.ID.ValueString(),
	})
	if err != nil {
		if tharsis.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error reading managed identity alias",
			err.Error(),
		)
		return
	}

	// Copy the from-Tharsis struct to the state.
	if err = t.copyManagedIdentityAlias(*found, &state); err != nil {
		resp.Diagnostics.AddError(
			"Error setting state",
			err.Error(),
		)
		return
	}

	// Set the refreshed state, whether or not there is an error.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (t *managedIdentityAliasResource) Update(ctx context.Context,
	req resource.UpdateRequest, resp *resource.UpdateResponse) {

	// Retrieve values from plan.
	var plan ManagedIdentityAliasModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ManagedIdentityAliasModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !reflect.DeepEqual(plan, state) {
		resp.Diagnostics.AddError(
			"Error updating managed identity alias",
			"An alias cannot be updated",
		)
	}
}

func (t *managedIdentityAliasResource) Delete(ctx context.Context,
	req resource.DeleteRequest, resp *resource.DeleteResponse) {

	// Get the current state.
	var state ManagedIdentityAliasModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the managed identity alias via Tharsis.
	// The ID is used to find the record to delete.
	err := t.client.ManagedIdentity.DeleteManagedIdentityAlias(ctx,
		&ttypes.DeleteManagedIdentityAliasInput{
			ID: state.ID.ValueString(),
		})
	if err != nil {

		// Handle the case that the managed identity alias no longer exists.
		if tharsis.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error deleting managed identity alias",
			err.Error(),
		)
	}
}

// ImportState helps the provider implement the ResourceWithImportState interface.
func (t *managedIdentityAliasResource) ImportState(ctx context.Context,
	req resource.ImportStateRequest, resp *resource.ImportStateResponse) {

	// Retrieve import ID and save to id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// copyManagedIdentityAlias copies the contents of a managed identity alias.
// It is intended to copy from a struct returned by Tharsis to a Terraform plan or state.
func (t *managedIdentityAliasResource) copyManagedIdentityAlias(src ttypes.ManagedIdentity, dest *ManagedIdentityAliasModel) error {

	dest.ID = types.StringValue(src.Metadata.ID)
	dest.ResourcePath = types.StringValue(src.ResourcePath)
	dest.Name = types.StringValue(src.Name)
	dest.GroupPath = types.StringValue(src.GroupPath)
	dest.AliasSourceID = types.StringValue(*src.AliasSourceID)

	// Must use time value from SDK/API.  Using time.Now() is not reliable.
	dest.LastUpdated = types.StringValue(src.Metadata.LastUpdatedTimestamp.Format(time.RFC850))

	return nil
}
