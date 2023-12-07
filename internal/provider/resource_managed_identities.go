package provider

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
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

// ManagedIdentityModel is the model for a managed identity.
type ManagedIdentityModel struct {
	ID                        types.String `tfsdk:"id"`
	Type                      types.String `tfsdk:"type"`
	ResourcePath              types.String `tfsdk:"resource_path"`
	Name                      types.String `tfsdk:"name"`
	Description               types.String `tfsdk:"description"`
	GroupPath                 types.String `tfsdk:"group_path"`
	AWSRole                   types.String `tfsdk:"aws_role"`
	AzureClientID             types.String `tfsdk:"azure_client_id"`
	AzureTenantID             types.String `tfsdk:"azure_tenant_id"`
	TharsisServiceAccountPath types.String `tfsdk:"tharsis_service_account_path"`
	Subject                   types.String `tfsdk:"subject"`
	LastUpdated               types.String `tfsdk:"last_updated"`
}

// managedIdentityDataInput has all fields required for input to the encoded data string.
// The vendor-specific prefixes are not used in the SDK, so they are omitted from the JSON tags.
type managedIdentityDataInput struct {
	AWSRole                   string `json:"role,omitempty"`
	AzureClientID             string `json:"clientId,omitempty"`
	AzureTenantID             string `json:"tenantId,omitempty"`
	TharsisServiceAccountPath string `json:"serviceAccountPath,omitempty"`
}

// managedIdentityData has all fields required for output from the encoded data string.
// The vendor-specific prefixes are not used in the SDK, so they are omitted from the JSON tags.
type managedIdentityData struct {
	AWSRole                   *string `json:"role,omitempty"`
	AzureClientID             *string `json:"clientId,omitempty"`
	AzureTenantID             *string `json:"tenantId,omitempty"`
	TharsisServiceAccountPath *string `json:"serviceAccountPath,omitempty"`
	Subject                   string  `json:"subject,omitempty"`
}

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ resource.Resource                = (*managedIdentityResource)(nil)
	_ resource.ResourceWithConfigure   = (*managedIdentityResource)(nil)
	_ resource.ResourceWithImportState = (*managedIdentityResource)(nil)
)

// NewManagedIdentityResource is a helper function to simplify the provider implementation.
func NewManagedIdentityResource() resource.Resource {
	return &managedIdentityResource{}
}

type managedIdentityResource struct {
	client *tharsis.Client
}

// Metadata returns the full name of the resource, including prefix, underscore, instance name.
func (t *managedIdentityResource) Metadata(_ context.Context,
	_ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "tharsis_managed_identity"
}

func (t *managedIdentityResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	description := "Defines and manages a managed identity."

	resp.Schema = schema.Schema{
		Version:             1,
		MarkdownDescription: description,
		Description:         description,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "String identifier of the managed identity.",
				Description:         "String identifier of the managed identity.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Type of managed identity: AWS, Azure, or Tharsis.",
				Description:         "Type of managed identity: AWS, Azure, or Tharsis.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"resource_path": schema.StringAttribute{
				MarkdownDescription: "The path of the parent group plus the name of the managed identity.",
				Description:         "The path of the parent group plus the name of the managed identity.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the managed identity.",
				Description:         "The name of the managed identity.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A description of the managed identity.",
				Description:         "A description of the managed identity.",
				Optional:            true,
				// Description can be updated in place, so no RequiresReplace plan modifier.
			},
			"group_path": schema.StringAttribute{
				MarkdownDescription: "Full path of the parent group.",
				Description:         "Full path of the parent group.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"aws_role": schema.StringAttribute{
				MarkdownDescription: "AWS role",
				Description:         "AWS role",
				Optional:            true,
				// Can be updated in place, so no RequiresReplace plan modifier.
			},
			"azure_client_id": schema.StringAttribute{
				MarkdownDescription: "Azure client ID",
				Description:         "Azure client ID",
				Optional:            true,
				// Can be updated in place, so no RequiresReplace plan modifier.
			},
			"azure_tenant_id": schema.StringAttribute{
				MarkdownDescription: "Azure tenant ID",
				Description:         "Azure tenant ID",
				Optional:            true,
				// Can be updated in place, so no RequiresReplace plan modifier.
			},
			"tharsis_service_account_path": schema.StringAttribute{
				MarkdownDescription: "Tharsis service account path",
				Description:         "Tharsis service account path",
				Optional:            true,
				// Can be updated in place, so no RequiresReplace plan modifier.
			},
			"subject": schema.StringAttribute{
				MarkdownDescription: "subject string for AWS, Azure, and Tharsis",
				Description:         "subject string for AWS. Azure, and Tharsis",
				Computed:            true,
			},
			"last_updated": schema.StringAttribute{
				MarkdownDescription: "Timestamp when this managed identity was most recently updated.",
				Description:         "Timestamp when this managed identity was most recently updated.",
				Computed:            true,
			},
		},
	}
}

// Configure lets the provider implement the ResourceWithConfigure interface.
func (t *managedIdentityResource) Configure(_ context.Context,
	req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	t.client = req.ProviderData.(*tharsis.Client)
}

func (t *managedIdentityResource) Create(ctx context.Context,
	req resource.CreateRequest, resp *resource.CreateResponse) {

	// Retrieve values from managedIdentity.
	var managedIdentity ManagedIdentityModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &managedIdentity)...)
	if resp.Diagnostics.HasError() {
		return
	}

	encodedData, err := t.encodeDataString(managedIdentity.Type,
		managedIdentityDataInput{
			AWSRole:                   managedIdentity.AWSRole.ValueString(),
			AzureClientID:             managedIdentity.AzureClientID.ValueString(),
			AzureTenantID:             managedIdentity.AzureTenantID.ValueString(),
			TharsisServiceAccountPath: managedIdentity.TharsisServiceAccountPath.ValueString(),
		})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error encoding managed identity data field",
			err.Error(),
		)
		return
	}

	// Create the managed identity.
	created, err := t.client.ManagedIdentity.CreateManagedIdentity(ctx,
		&ttypes.CreateManagedIdentityInput{
			Type:        ttypes.ManagedIdentityType(managedIdentity.Type.ValueString()),
			Name:        managedIdentity.Name.ValueString(),
			Description: managedIdentity.Description.ValueString(),
			GroupPath:   managedIdentity.GroupPath.ValueString(),
			Data:        encodedData,
		})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating managed identity",
			err.Error(),
		)
		return
	}

	// Map the response body to the schema and update the plan with the computed attribute values.
	// Because the schema uses the Set type rather than the List type, make sure to set all fields.
	if err = t.copyManagedIdentity(*created, &managedIdentity); err != nil {
		resp.Diagnostics.AddError(
			"Error setting state",
			err.Error(),
		)
		return
	}

	// Set the response state to the fully-populated plan, whether or not there is an error.
	resp.Diagnostics.Append(resp.State.Set(ctx, managedIdentity)...)
}

func (t *managedIdentityResource) Read(ctx context.Context,
	req resource.ReadRequest, resp *resource.ReadResponse) {

	// Get the current state.
	var state ManagedIdentityModel
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
			"Error reading managed identity",
			err.Error(),
		)
		return
	}

	// Copy the from-Tharsis struct to the state.
	if err = t.copyManagedIdentity(*found, &state); err != nil {
		resp.Diagnostics.AddError(
			"Error setting state",
			err.Error(),
		)
		return
	}

	// Set the refreshed state, whether or not there is an error.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (t *managedIdentityResource) Update(ctx context.Context,
	req resource.UpdateRequest, resp *resource.UpdateResponse) {

	// Retrieve values from plan for the ID, the description, and the data.
	var plan ManagedIdentityModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	encodedData, err := t.encodeDataString(plan.Type,
		managedIdentityDataInput{
			AWSRole:                   plan.AWSRole.ValueString(),
			AzureClientID:             plan.AzureClientID.ValueString(),
			AzureTenantID:             plan.AzureTenantID.ValueString(),
			TharsisServiceAccountPath: plan.TharsisServiceAccountPath.ValueString(),
		})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error encoding managed identity data field",
			err.Error(),
		)
		return
	}

	// Update the managed identity via Tharsis.
	// The ID is used to find the record to update.
	// The description and data are modified.
	updated, err := t.client.ManagedIdentity.UpdateManagedIdentity(ctx,
		&ttypes.UpdateManagedIdentityInput{
			ID:          plan.ID.ValueString(),
			Description: plan.Description.ValueString(),
			Data:        encodedData,
		})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating managed identity",
			err.Error(),
		)
		return
	}

	// Copy all fields returned by Tharsis back into the plan.
	if err = t.copyManagedIdentity(*updated, &plan); err != nil {
		resp.Diagnostics.AddError(
			"Error setting state",
			err.Error(),
		)
		return
	}

	// Set the response state to the fully-populated plan, with or without error.
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (t *managedIdentityResource) Delete(ctx context.Context,
	req resource.DeleteRequest, resp *resource.DeleteResponse) {

	// Get the current state.
	var state ManagedIdentityModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the managed identity via Tharsis.
	// The ID is used to find the record to delete.
	err := t.client.ManagedIdentity.DeleteManagedIdentity(ctx,
		&ttypes.DeleteManagedIdentityInput{
			ID: state.ID.ValueString(),
		})
	if err != nil {

		// Handle the case that the managed identity no longer exists.
		if tharsis.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error deleting managed identity",
			err.Error(),
		)
	}
}

// ImportState helps the provider implement the ResourceWithImportState interface.
func (t *managedIdentityResource) ImportState(ctx context.Context,
	req resource.ImportStateRequest, resp *resource.ImportStateResponse) {

	// Retrieve import ID and save to id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// copyManagedIdentity copies the contents of a managed identity.
// It is intended to copy from a struct returned by Tharsis to a Terraform plan or state.
func (t *managedIdentityResource) copyManagedIdentity(src ttypes.ManagedIdentity, dest *ManagedIdentityModel) error {

	decodedData, err := t.decodeDataString(src.Data)
	if err != nil {
		return err
	}

	dest.ID = types.StringValue(src.Metadata.ID)
	dest.Type = types.StringValue(string(src.Type))
	dest.ResourcePath = types.StringValue(src.ResourcePath)
	dest.Name = types.StringValue(src.Name)
	dest.Description = types.StringValue(src.Description)
	dest.GroupPath = types.StringValue(src.GroupPath)
	if decodedData.AWSRole != nil {
		dest.AWSRole = types.StringValue(*decodedData.AWSRole)
	}
	if decodedData.AzureClientID != nil {
		dest.AzureClientID = types.StringValue(*decodedData.AzureClientID)
	}
	if decodedData.AzureTenantID != nil {
		dest.AzureTenantID = types.StringValue(*decodedData.AzureTenantID)
	}
	if decodedData.TharsisServiceAccountPath != nil {
		dest.TharsisServiceAccountPath = types.StringValue(*decodedData.TharsisServiceAccountPath)
	}
	dest.Subject = types.StringValue(decodedData.Subject)

	// Must use time value from SDK/API.  Using time.Now() is not reliable.
	dest.LastUpdated = types.StringValue(src.Metadata.LastUpdatedTimestamp.Format(time.RFC850))

	return nil
}

// encodeDataString checks the AWS role, Azure client ID, Azure tenant ID, Tharsis service account path,
// and subject fields and then marshals them into the appropriate type and base64 encodes that.
func (t *managedIdentityResource) encodeDataString(managedIdentityType types.String, input managedIdentityDataInput) (string, error) {
	type2 := ttypes.ManagedIdentityType(managedIdentityType.ValueString())

	// What to check depends on the type of managed identity this is.
	switch type2 {
	case ttypes.ManagedIdentityAWSFederated:
		if input.AWSRole == "" {
			return "", fmt.Errorf("non-empty role is required for AWS managed identity")
		}
		if input.AzureClientID != "" {
			return "", fmt.Errorf("non-empty client ID is not allowed for AWS managed identity")
		}
		if input.AzureTenantID != "" {
			return "", fmt.Errorf("non-empty tenant ID is not allowed for AWS managed identity")
		}
	case ttypes.ManagedIdentityAzureFederated:
		if input.AWSRole != "" {
			return "", fmt.Errorf("non-empty role is not allowed for Azure managed identity")
		}
		if input.AzureClientID == "" {
			return "", fmt.Errorf("non-empty client ID is required for Azure managed identity")
		}
		if input.AzureTenantID == "" {
			return "", fmt.Errorf("non-empty tenant ID is required for Azure managed identity")
		}
	case ttypes.ManagedIdentityTharsisFederated:
		if input.TharsisServiceAccountPath == "" {
			return "", fmt.Errorf("non-empty service account path is required for Tharsis managed identity")
		}
	default:
		return "", fmt.Errorf("invalid managed identity type: %s", type2)
	}

	// With the checking completed, JSON-encode the fields, taking advantage of omitempty.
	preResult, err := json.Marshal(input)
	if err != nil {
		return "", fmt.Errorf("failed to marshal managed identity data fields")
	}

	// Return it in base64-encoded form.
	return base64.StdEncoding.EncodeToString(preResult), nil
}

// decodeDataString base64 decodes and then unmarshals the
// AWS role, Azure client ID, Azure tenant ID, Tharsis service account path, and subject fields
func (t *managedIdentityResource) decodeDataString(encoded string) (*managedIdentityData, error) {

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}

	var result managedIdentityData
	if jErr := json.Unmarshal(decoded, &result); jErr != nil {
		return nil, err
	}

	return &result, nil
}

// The End.
