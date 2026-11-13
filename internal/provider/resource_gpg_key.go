package provider

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
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

// GPGKeyModel is the model for a GPG key.
// Fields intentionally omitted: AssignedManagedIdentities, ManagedIdentities, ServiceAccounts,
// StateVersions, Memberships, Variables, ActivityEvents.
// Also for now, omitting DirtyState, Locked, CurrentStateVersionID, and CurrentJobID.
type GPGKeyModel struct {
	ID           types.String `tfsdk:"id"`
	LastUpdated  types.String `tfsdk:"last_updated"`
	CreatedBy    types.String `tfsdk:"created_by"`
	ASCIIArmor   types.String `tfsdk:"ascii_armor"`
	Fingerprint  types.String `tfsdk:"fingerprint"`
	GPGKeyID     types.String `tfsdk:"gpg_key_id"`
	GroupPath    types.String `tfsdk:"group_path"`
	GroupID      types.String `tfsdk:"group_id"`
	ResourcePath types.String `tfsdk:"resource_path"`
}

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ resource.Resource                = (*gpgKeyResource)(nil)
	_ resource.ResourceWithConfigure   = (*gpgKeyResource)(nil)
	_ resource.ResourceWithModifyPlan  = (*gpgKeyResource)(nil)
	_ resource.ResourceWithImportState = (*gpgKeyResource)(nil)
)

// NewGPGKeyResource is a helper function to simplify the provider implementation.
func NewGPGKeyResource() resource.Resource {
	return &gpgKeyResource{}
}

type gpgKeyResource struct {
	client *client.GRPCClient
}

// Metadata returns the full name of the resource, including prefix, underscore, instance name.
func (t *gpgKeyResource) Metadata(_ context.Context,
	_ resource.MetadataRequest, resp *resource.MetadataResponse,
) {
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
				MarkdownDescription: "Timestamp when this GPG key was most recently updated by Terraform.",
				Description:         "Timestamp when this GPG key was most recently updated by Terraform.",
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIf(requireReplaceIfDifferentKey,
						"Changing to a different key replaces the GPG key.",
						"Changing to a different key replaces the GPG key."),
				},
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
				Optional:            true,
				Computed:            true,
				DeprecationMessage:  "Use group_id instead. This field will be removed in a future version.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"group_id": schema.StringAttribute{
				MarkdownDescription: "The ID or TRN of the parent group.",
				Description:         "The ID or TRN of the parent group.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"resource_path": schema.StringAttribute{
				MarkdownDescription: "Path of this GPG key.",
				Description:         "Path of this GPG key.",
				Computed:            true,
				DeprecationMessage:  "Use the id field instead. This field will be removed in a future version.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Configure lets the provider implement the ResourceWithConfigure interface.
func (t *gpgKeyResource) Configure(_ context.Context,
	req resource.ConfigureRequest, _ *resource.ConfigureResponse,
) {
	if req.ProviderData == nil {
		return
	}
	t.client = req.ProviderData.(*client.GRPCClient)
}

func (t *gpgKeyResource) Create(ctx context.Context,
	req resource.CreateRequest, resp *resource.CreateResponse,
) {
	// Retrieve values from GPG key.
	var gpgKey GPGKeyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &gpgKey)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create the GPG key.
	var groupID string
	if v := gpgKey.GroupID.ValueString(); v != "" {
		groupID = v
	} else if v := gpgKey.GroupPath.ValueString(); v != "" {
		groupID = trn.TypeGroup.Build(v)
	} else {
		resp.Diagnostics.AddError("Either group_id or group_path must be specified", "")
		return
	}

	created, err := t.client.GPGKeysClient.CreateGPGKey(ctx,
		&pb.CreateGPGKeyRequest{
			AsciiArmor: gpgKey.ASCIIArmor.ValueString(),
			GroupId:    groupID,
		})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating GPG key",
			err.Error(),
		)
		return
	}

	// Map the response body to the schema and update the plan with the computed attribute values.
	plannedGroupID := gpgKey.GroupID
	plannedArmor := gpgKey.ASCIIArmor
	t.copyGPGKey(created, &gpgKey)
	// The API acted on the planned reference, so the planned form is kept.
	gpgKey.GroupID = keepPlanned(plannedGroupID, created.GroupId)
	// Tharsis returns the key re-armored, so keep the configured text of the same key.
	gpgKey.ASCIIArmor = plannedArmor

	// Must use time value from SDK/API.  Using time.Now() is not reliable.
	gpgKey.LastUpdated = types.StringValue(created.Metadata.UpdatedAt.AsTime().Format(time.RFC850))

	// Set the response state to the fully-populated plan, whether or not there is an error.
	resp.Diagnostics.Append(resp.State.Set(ctx, gpgKey)...)
}

func (t *gpgKeyResource) Read(ctx context.Context,
	req resource.ReadRequest, resp *resource.ReadResponse,
) {
	// Get the current state.
	var state GPGKeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the GPG key from Tharsis.
	found, err := t.client.GPGKeysClient.GetGPGKeyByID(ctx, &pb.GetGPGKeyByIDRequest{
		Id: state.ID.ValueString(),
	})
	if err != nil {
		if status.Code(err) == codes.NotFound {
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
	storedArmor := state.ASCIIArmor
	t.copyGPGKey(found, &state)
	// A key can't change in Tharsis, so stored armor still holds it; an import has none and takes Tharsis's.
	if !storedArmor.IsNull() {
		state.ASCIIArmor = storedArmor
	}
	// Keep the stored group_id form while it still names the group; otherwise take the API's
	// Global ID so a change made outside Terraform shows as drift.
	state.GroupID = keepOrRefresh(state.GroupID, found.GroupId,
		trn.TypeGroup.Build(trn.MustParseAny(found.Metadata.Trn).ParentPath()))

	// Read normally leaves the last updated timestamp alone because it reflects the last change made by
	// Terraform.  After an import there is no prior value, so use the time reported by Tharsis.
	if state.LastUpdated.IsNull() {
		state.LastUpdated = types.StringValue(found.Metadata.UpdatedAt.AsTime().Format(time.RFC850))
	}

	// Set the refreshed state, whether or not there is an error.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is reached only when group_id switches between the group's Global ID and its
// TRN, or ascii_armor switches to another armoring of the same key: every other input forces
// a replacement, and so does moving to a different group or key. Nothing changes in Tharsis,
// so only the stored forms are updated.
func (t *gpgKeyResource) Update(ctx context.Context,
	req resource.UpdateRequest, resp *resource.UpdateResponse,
) {
	// Store the planned forms and keep everything else as it was; an unknown planned value keeps the stored one.
	var state, plan GPGKeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if isKnownValue(plan.GroupID) {
		state.GroupID = plan.GroupID
	}
	if isKnownValue(plan.ASCIIArmor) {
		state.ASCIIArmor = plan.ASCIIArmor
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (t *gpgKeyResource) Delete(ctx context.Context,
	req resource.DeleteRequest, resp *resource.DeleteResponse,
) {
	// Get the current state.
	var state GPGKeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the GPG key via Tharsis.
	_, err := t.client.GPGKeysClient.DeleteGPGKey(ctx,
		&pb.DeleteGPGKeyRequest{
			Id: state.ID.ValueString(),
		})
	if err != nil {

		// Handle the case that the GPG key no longer exists.
		if status.Code(err) == codes.NotFound {
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
	req resource.ImportStateRequest, resp *resource.ImportStateResponse,
) {
	// Retrieve import ID and save to id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// copyGPGKey copies the contents of a GPG key.
// It is intended to copy from a struct returned by Tharsis to a Terraform plan or state.
func (t *gpgKeyResource) copyGPGKey(src *pb.GPGKey, dest *GPGKeyModel) {
	parsed := trn.MustParseAny(src.Metadata.Trn)
	dest.ID = types.StringValue(src.Metadata.Id)
	dest.CreatedBy = types.StringValue(src.CreatedBy)
	dest.ASCIIArmor = types.StringValue(src.AsciiArmor)
	dest.Fingerprint = types.StringValue(src.Fingerprint)
	dest.GPGKeyID = types.StringValue(src.GpgKeyId)
	// group_id is set by the callers: Read keeps or refreshes the stored form, Create keeps
	// the planned value.
	dest.GroupPath = types.StringValue(parsed.ParentPath())
	dest.ResourcePath = types.StringValue(parsed.Path())
}

// ModifyPlan forces a replacement only when group_id moves to a different object. A
// switch between a Global ID and a TRN of the same object is an in-place update.
func (t *gpgKeyResource) ModifyPlan(ctx context.Context,
	req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse,
) {
	requireReplaceIfDifferentObject(ctx, t.client, req, resp, path.Root("group_id"), trn.TypeGroup)
}

// requireReplaceIfDifferentKey replaces the GPG key unless the planned and stored armor hold the same key.
func requireReplaceIfDifferentKey(_ context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
	resp.RequiresReplace = !sameGPGKey(req.PlanValue.ValueString(), req.StateValue.ValueString())
}

// sameGPGKey reports whether two armored public key blocks hold the same key, compared as Tharsis stores it: re-serialized.
func sameGPGKey(armoredA, armoredB string) bool {
	keyA, err := serializeGPGKey(armoredA)
	if err != nil {
		return false
	}

	keyB, err := serializeGPGKey(armoredB)
	if err != nil {
		return false
	}

	return bytes.Equal(keyA, keyB)
}

// serializeGPGKey returns the packets of an armored public key, rejecting what Tharsis rejects so that input never counts as the same key.
func serializeGPGKey(armored string) ([]byte, error) {
	if strings.Count(armored, "-----BEGIN ") != 1 {
		return nil, fmt.Errorf("expected exactly one armor block")
	}

	block, err := armor.Decode(strings.NewReader(armored))
	if err != nil {
		return nil, err
	}

	if block.Type != openpgp.PublicKeyType {
		return nil, fmt.Errorf("expected armor type %s, got %s", openpgp.PublicKeyType, block.Type)
	}

	entities, err := openpgp.ReadKeyRing(block.Body)
	if err != nil {
		return nil, err
	}

	if len(entities) != 1 {
		return nil, fmt.Errorf("expected one key, got %d", len(entities))
	}

	if hasPrivateKey(entities[0]) {
		return nil, fmt.Errorf("armor contains private key material")
	}

	var buf bytes.Buffer
	if err := entities[0].Serialize(&buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// hasPrivateKey reports whether the key or any of its subkeys includes private key material.
func hasPrivateKey(entity *openpgp.Entity) bool {
	if entity.PrivateKey != nil {
		return true
	}

	for _, subkey := range entity.Subkeys {
		if subkey.PrivateKey != nil {
			return true
		}
	}

	return false
}
