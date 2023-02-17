package provider

import (
	"context"
	"reflect"
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

// GPGKeyModel is the model for a GPG key.
// Fields intentionally omitted: AssignedManagedIdentities, ManagedIdentities, ServiceAccounts,
// StateVersions, Memberships, Variables, ActivityEvents.
// Also for now, omitting DirtyState, Locked, CurrentStateVersionID, and CurrentJobID.
type GPGKeyModel struct {
	ID          types.String `tfsdk:"id"`
	LastUpdated types.String `tfsdk:"last_updated"`
	CreatedBy   types.String `tfsdk:"created_by"`
	ASCIIArmor  types.String `tfsdk:"ascii_armor"`
	Fingerprint types.String `tfsdk:"fingerprint"`
	GPGKeyID    types.String `tfsdk:"gpg_key_id"`
	GroupPath   types.String `tfsdk:"group_path"`
}

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ resource.Resource                = (*gpgKeyResource)(nil)
	_ resource.ResourceWithConfigure   = (*gpgKeyResource)(nil)
	_ resource.ResourceWithImportState = (*gpgKeyResource)(nil)
)

// NewGPGKeyResource is a helper function to simplify the provider implementation.
func NewGPGKeyResource() resource.Resource {
	return &gpgKeyResource{}
}

type gpgKeyResource struct {
	client *tharsis.Client
}

// Metadata returns the full name of the resource, including prefix, underscore, instance name.
func (t *gpgKeyResource) Metadata(ctx context.Context,
	req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "tharsis_gpg_key"
}

func (t *gpgKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	description := "Defines and manages a GPG key."

	resp.Schema = schema.Schema{
		Version:             1,
		MarkdownDescription: description,
		Description:         description,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "String identifier of the GPG key.",
				Description:         "String identifier of the GPG key.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_updated": schema.StringAttribute{
				MarkdownDescription: "Timestamp when this GPG key was most recently updated.",
				Description:         "Timestamp when this GPG key was most recently updated.",
				Computed:            true,
			},
			"created_by": schema.StringAttribute{
				MarkdownDescription: "The email address of the user or account that created this GPG key.",
				Description:         "The email address of the user or account that created this GPG key.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ascii_armor": schema.StringAttribute{
				MarkdownDescription: "The ASCII armored key.",
				Description:         "The ASCII armored key.",
				Required:            true,
			},
			"fingerprint": schema.StringAttribute{
				MarkdownDescription: "The fingerprint of the GPG key.",
				Description:         "The fingerprint of the GPG key.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"gpg_key_id": schema.StringAttribute{
				MarkdownDescription: "The GPG key string for this GPG key.",
				Description:         "The GPG key string for this GPG key.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"group_path": schema.StringAttribute{
				MarkdownDescription: "Path of the parent group.",
				Description:         "Path of the parent group.",
				Required:            true,
			},
		},
	}
}

// Configure lets the provider implement the ResourceWithConfigure interface.
func (t *gpgKeyResource) Configure(_ context.Context,
	req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	t.client = req.ProviderData.(*tharsis.Client)
}

func (t *gpgKeyResource) Create(ctx context.Context,
	req resource.CreateRequest, resp *resource.CreateResponse) {

	// Retrieve values from GPG key.
	var gpgKey GPGKeyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &gpgKey)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create the GPG key.
	created, err := t.client.GPGKey.CreateGPGKey(ctx,
		&ttypes.CreateGPGKeyInput{
			ASCIIArmor: gpgKey.ASCIIArmor.ValueString(),
			GroupPath:  gpgKey.GroupPath.ValueString(),
		})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating GPG key",
			err.Error(),
		)
		return
	}

	// Map the response body to the schema and update the plan with the computed attribute values.
	t.copyGPGKey(*created, &gpgKey)

	// Set the response state to the fully-populated plan, whether or not there is an error.
	resp.Diagnostics.Append(resp.State.Set(ctx, gpgKey)...)
}

func (t *gpgKeyResource) Read(ctx context.Context,
	req resource.ReadRequest, resp *resource.ReadResponse) {

	// Get the current state.
	var state GPGKeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the GPG key from Tharsis.
	found, err := t.client.GPGKey.GetGPGKey(ctx, &ttypes.GetGPGKeyInput{
		ID: state.ID.ValueString(),
	})
	if err != nil {
		if tharsis.NotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error reading GPG key",
			err.Error(),
		)
		return
	}

	// Copy the from-Tharsis struct to the state.
	t.copyGPGKey(*found, &state)

	// Set the refreshed state, whether or not there is an error.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (t *gpgKeyResource) Update(ctx context.Context,
	req resource.UpdateRequest, resp *resource.UpdateResponse) {

	// Retrieve values from plan.
	var plan GPGKeyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state GPGKeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !reflect.DeepEqual(plan, state) {
		resp.Diagnostics.AddError(
			"Error updating GPG key",
			"A GPG key cannot be updated",
		)
	}
}

func (t *gpgKeyResource) Delete(ctx context.Context,
	req resource.DeleteRequest, resp *resource.DeleteResponse) {

	// Get the current state.
	var state GPGKeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the GPG key via Tharsis.
	_, err := t.client.GPGKey.DeleteGPGKey(ctx,
		&ttypes.DeleteGPGKeyInput{
			ID: state.ID.ValueString(),
		})
	if err != nil {

		// Handle the case that the GPG key no longer exists.
		if tharsis.NotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error deleting GPG key",
			err.Error(),
		)
	}
}

// ImportState helps the provider implement the ResourceWithImportState interface.
func (t *gpgKeyResource) ImportState(ctx context.Context,
	req resource.ImportStateRequest, resp *resource.ImportStateResponse) {

	// Get the GPG key by ID from Tharsis.
	found, err := t.client.GPGKey.GetGPGKey(ctx, &ttypes.GetGPGKeyInput{
		ID: req.ID,
	})
	if err != nil {
		if tharsis.NotFoundError(err) {
			resp.Diagnostics.AddError(
				"Import GPG key not found: "+req.ID,
				"",
			)
			return
		}

		resp.Diagnostics.AddError(
			"Import GPG key not found: "+req.ID,
			err.Error(),
		)
		return
	}

	// Import by full path.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), found.Metadata.ID)...)
}

// copyGPGKey copies the contents of a GPG key.
// It is intended to copy from a struct returned by Tharsis to a Terraform plan or state.
func (t *gpgKeyResource) copyGPGKey(src ttypes.GPGKey, dest *GPGKeyModel) {
	dest.ID = types.StringValue(src.Metadata.ID)
	dest.CreatedBy = types.StringValue(src.CreatedBy)
	dest.ASCIIArmor = types.StringValue(src.ASCIIArmor)
	dest.Fingerprint = types.StringValue(src.Fingerprint)
	dest.GPGKeyID = types.StringValue(src.GPGKeyID)
	dest.GroupPath = types.StringValue(src.GroupPath)

	// Must use time value from SDK/API.  Using time.Now() is not reliable.
	dest.LastUpdated = types.StringValue(src.Metadata.LastUpdatedTimestamp.Format(time.RFC850))
}

// The End.
