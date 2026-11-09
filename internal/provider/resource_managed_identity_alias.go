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
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/client"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ManagedIdentityAliasModel is the model for a managed identity alias.
type ManagedIdentityAliasModel struct {
	ID              types.String `tfsdk:"id"`
	ResourcePath    types.String `tfsdk:"resource_path"`
	Name            types.String `tfsdk:"name"`
	GroupPath       types.String `tfsdk:"group_path"`
	GroupID         types.String `tfsdk:"group_id"`
	LastUpdated     types.String `tfsdk:"last_updated"`
	AliasSourceID   types.String `tfsdk:"alias_source_id"`
	AliasSourcePath types.String `tfsdk:"alias_source_path"`
}

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ resource.Resource                = (*managedIdentityAliasResource)(nil)
	_ resource.ResourceWithConfigure   = (*managedIdentityAliasResource)(nil)
	_ resource.ResourceWithModifyPlan  = (*managedIdentityAliasResource)(nil)
	_ resource.ResourceWithImportState = (*managedIdentityAliasResource)(nil)
)

// NewManagedIdentityAliasResource is a helper function to simplify the provider implementation.
func NewManagedIdentityAliasResource() resource.Resource {
	return &managedIdentityAliasResource{}
}

type managedIdentityAliasResource struct {
	client *client.GRPCClient
}

// Metadata returns the full name of the resource, including prefix, underscore, instance name.
func (t *managedIdentityAliasResource) Metadata(_ context.Context,
	_ resource.MetadataRequest, resp *resource.MetadataResponse) {
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
				DeprecationMessage:  "Use the id field instead. This field will be removed in a future version.",
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
				Optional:            true,
				Computed:            true,
				DeprecationMessage:  "Use group_id instead. This field will be removed in a future version.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"group_id": schema.StringAttribute{
				MarkdownDescription: "The ID or TRN of the group where alias will be created.",
				Description:         "The ID or TRN of the group where alias will be created.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_updated": schema.StringAttribute{
				MarkdownDescription: "Timestamp when this managed identity alias was most recently updated by Terraform.",
				Description:         "Timestamp when this managed identity alias was most recently updated by Terraform.",
				Computed:            true,
			},
			"alias_source_id": schema.StringAttribute{
				MarkdownDescription: "The ID or TRN of the managed identity being aliased.",
				Description:         "The ID or TRN of the managed identity being aliased.",
				Optional:            true,
				Computed:            true,
			},
			"alias_source_path": schema.StringAttribute{
				MarkdownDescription: "Full path of the managed identity being aliased.",
				Description:         "Full path of the managed identity being aliased.",
				Optional:            true,
				DeprecationMessage:  "Use alias_source_id instead. This field will be removed in a future version.",
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
	t.client = req.ProviderData.(*client.GRPCClient)
}

func (t *managedIdentityAliasResource) Create(ctx context.Context,
	req resource.CreateRequest, resp *resource.CreateResponse) {

	// Retrieve values from managedIdentityAlias.
	var managedIdentityAlias ManagedIdentityAliasModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &managedIdentityAlias)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Resolve alias source: prefer ID, fall back to path converted to TRN.
	var aliasSourceID string
	if v := managedIdentityAlias.AliasSourceID.ValueString(); v != "" {
		aliasSourceID = v
	} else if v := managedIdentityAlias.AliasSourcePath.ValueString(); v != "" {
		aliasSourceID = trn.TypeManagedIdentity.Build(v)
	} else {
		resp.Diagnostics.AddError(
			"Error creating managed identity alias",
			"Exactly one of alias_source_id or alias_source_path must be specified",
		)
		return
	}

	var groupID string
	if v := managedIdentityAlias.GroupID.ValueString(); v != "" {
		groupID = v
	} else if v := managedIdentityAlias.GroupPath.ValueString(); v != "" {
		groupID = trn.TypeGroup.Build(v)
	} else {
		resp.Diagnostics.AddError("Either group_id or group_path must be specified", "")
		return
	}

	// Create the managed identity alias.
	created, err := t.client.ManagedIdentitiesClient.CreateManagedIdentityAlias(ctx,
		&pb.CreateManagedIdentityAliasRequest{
			Name:          managedIdentityAlias.Name.ValueString(),
			AliasSourceId: aliasSourceID,
			GroupId:       groupID,
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
	plannedGroupID := managedIdentityAlias.GroupID
	plannedAliasSourceID := managedIdentityAlias.AliasSourceID
	t.copyManagedIdentityAlias(ctx, created, &managedIdentityAlias)
	// The API acted on the planned reference, so the planned form is kept.
	managedIdentityAlias.GroupID = keepPlanned(plannedGroupID, created.GroupId)
	managedIdentityAlias.AliasSourceID = keepPlanned(plannedAliasSourceID, created.GetAliasSourceId())

	// Must use time value from SDK/API.  Using time.Now() is not reliable.
	managedIdentityAlias.LastUpdated = types.StringValue(created.Metadata.UpdatedAt.AsTime().Format(time.RFC850))

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
	found, err := t.client.ManagedIdentitiesClient.GetManagedIdentityByID(ctx,
		&pb.GetManagedIdentityByIDRequest{
			Id: state.ID.ValueString(),
		})
	if err != nil {
		if status.Code(err) == codes.NotFound {
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
	t.copyManagedIdentityAlias(ctx, found, &state)
	// Keep the stored reference forms while they still name the same objects; otherwise take
	// the API's Global IDs so a change made outside Terraform shows as drift. The response
	// carries only the source's Global ID, so a TRN source is resolved to compare.
	state.GroupID = keepOrRefresh(state.GroupID, found.GroupId,
		trn.TypeGroup.Build(trn.MustParseAny(found.Metadata.Trn).ParentPath()))
	state.AliasSourceID = keepOrRefreshResolve(ctx, t.client, state.AliasSourceID, found.GetAliasSourceId(),
		trn.TypeManagedIdentity)

	// Read normally leaves the last updated timestamp alone because it reflects the last change made by
	// Terraform.  After an import there is no prior value, so use the time reported by Tharsis.
	if state.LastUpdated.IsNull() {
		state.LastUpdated = types.StringValue(found.Metadata.UpdatedAt.AsTime().Format(time.RFC850))
	}

	// Set the refreshed state, whether or not there is an error.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is reached only when group_id or alias_source_id switches between an object's
// Global ID and its TRN: every other input forces a replacement, and so does either
// reference moving to a different object. An alias cannot be updated in Tharsis and
// nothing changes there, so only the stored forms are updated.
func (t *managedIdentityAliasResource) Update(ctx context.Context,
	req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Every other input forces a replacement, so Update is reached only for a switch between
	// an object's Global ID and its TRN, which changes nothing in Tharsis. Store the planned
	// form and keep everything else as it was; an unknown planned value keeps the stored one.
	var state, plan ManagedIdentityAliasModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if isKnownValue(plan.GroupID) {
		state.GroupID = plan.GroupID
	}
	if isKnownValue(plan.AliasSourceID) {
		state.AliasSourceID = plan.AliasSourceID
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
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
	_, err := t.client.ManagedIdentitiesClient.DeleteManagedIdentityAlias(ctx,
		&pb.DeleteManagedIdentityAliasRequest{
			Id: state.ID.ValueString(),
		})
	if err != nil {

		// Handle the case that the managed identity alias no longer exists.
		if status.Code(err) == codes.NotFound {
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
func (t *managedIdentityAliasResource) copyManagedIdentityAlias(_ context.Context, src *pb.ManagedIdentity,
	dest *ManagedIdentityAliasModel,
) {
	parsed := trn.MustParseAny(src.Metadata.Trn)
	dest.ID = types.StringValue(src.Metadata.Id)
	dest.ResourcePath = types.StringValue(parsed.Path())
	dest.Name = types.StringValue(src.Name)
	// group_id and alias_source_id are set by the callers: Read keeps or refreshes the stored
	// form, Create keeps the planned value.
	dest.GroupPath = types.StringValue(parsed.ParentPath())
}

// ModifyPlan forces a replacement only when either group_id or alias_source_id moves to a
// different object. A switch between a Global ID and a TRN of the same object is an
// in-place update.
func (t *managedIdentityAliasResource) ModifyPlan(ctx context.Context,
	req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse,
) {
	requireReplaceIfDifferentObject(ctx, t.client, req, resp, path.Root("group_id"), trn.TypeGroup)
	requireReplaceIfDifferentObject(ctx, t.client, req, resp, path.Root("alias_source_id"), trn.TypeManagedIdentity)
}
