package tharsis

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	tharsis "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg"
	ttypes "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

// universalInputData has all fields required for input to the encoded data string.
type universalInputData struct {
	Role     string `json:"role,omitempty"`
	ClientID string `json:"clientId,omitempty"`
	TenantID string `json:"tenantId,omitempty"`
}

// universalData has all fields required for output from the encoded data string.
type universalData struct {
	Role     *string `json:"role,omitempty"`
	ClientID *string `json:"clientId,omitempty"`
	TenantID *string `json:"tenantId,omitempty"`
	Subject  string  `json:"subject,omitempty"`
}

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ resource.Resource                = &managedIdentityResource{}
	_ resource.ResourceWithConfigure   = &managedIdentityResource{}
	_ resource.ResourceWithImportState = &managedIdentityResource{}
)

// NewManagedIdentityResource is a helper function to simplify the provider implementation.
func NewManagedIdentityResource() resource.Resource {
	return &managedIdentityResource{}
}

type managedIdentityResource struct {
	client *tharsis.Client
}

// Metadata returns the full name of the resource, including prefix, underscore, instance name.
func (t *managedIdentityResource) Metadata(ctx context.Context,
	req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "tharsis_managed_identity"
}

// The diagnostics return value is required by the interface even though this function returns only nil.
func (t *managedIdentityResource) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	description := "Defines and manages a managed identity."

	// Prepare to nest the access rule schema inside this schema.
	accessRulesSchema, ruleDiags := NewManagedIdentityAccessRuleResource().GetSchema(ctx)
	if ruleDiags.HasError() {
		// The above call currently does not return diags, but this future-proofs us in case that changes.
		return tfsdk.Schema{}, ruleDiags
	}
	rulesAttribute := tfsdk.Attribute{
		MarkdownDescription: "List of access rules for the managed identity.",
		Description:         "List of access rules for the managed identity.",
		Optional:            true,
		Attributes:          tfsdk.SetNestedAttributes(accessRulesSchema.Attributes),
	}

	return tfsdk.Schema{
		Version: 1,

		MarkdownDescription: description,
		Description:         description,

		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:                types.StringType,
				MarkdownDescription: "String identifier of the managed identity.",
				Description:         "String identifier of the managed identity.",
				Computed:            true,
			},
			"type": {
				Type:                types.StringType,
				MarkdownDescription: "Type of managed identity, AWS or Azure.",
				Description:         "Type of managed identity, AWS or Azure.",
				Required:            true,
			},
			"resource_path": {
				Type:                types.StringType,
				MarkdownDescription: "The path of the parent group plus the name of the managed identity.",
				Description:         "The path of the parent group plus the name of the managed identity.",
				Computed:            true,
			},
			"name": {
				Type:                types.StringType,
				MarkdownDescription: "The name of the managed identity.",
				Description:         "The name of the managed identity.",
				Required:            true,
			},
			"description": {
				Type:                types.StringType,
				MarkdownDescription: "A description of the managed identity.",
				Description:         "A description of the managed identity.",
				Optional:            true,
			},
			"group_path": {
				Type:                types.StringType,
				MarkdownDescription: "Full path of the parent group.",
				Description:         "Full path of the parent group.",
				Required:            true,
			},
			"role": {Type: types.StringType,
				MarkdownDescription: "AWS role",
				Description:         "AWS role",
				Optional:            true,
			},
			"client_id": {Type: types.StringType,
				MarkdownDescription: "Azure client ID",
				Description:         "Azure client ID",
				Optional:            true,
			},
			"tenant_id": {Type: types.StringType,
				MarkdownDescription: "Azure tenant ID",
				Description:         "Azure tenant ID",
				Optional:            true,
			},
			"subject": {Type: types.StringType,
				MarkdownDescription: "subject string for AWS and Azure",
				Description:         "subject string for AWS and Azure",
				Computed:            true,
			},
			"access_rules": rulesAttribute,
			"last_updated": {
				Type:                types.StringType,
				MarkdownDescription: "Timestamp when this managed identity was most recently updated.",
				Description:         "Timestamp when this managed identity was most recently updated.",
				Computed:            true,
			},
		},
	}, nil
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

	// Retrieve values from plan.
	var plan ManagedIdentityModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build the access rule inputs.
	accessRuleInputs := []ttypes.ManagedIdentityAccessRuleInput{}
	for _, thInput := range plan.AccessRules {
		accessRuleInputs = append(accessRuleInputs, ttypes.ManagedIdentityAccessRuleInput{
			RunStage:               ttypes.JobType(thInput.RunStage.ValueString()),
			AllowedUsers:           valueStrings(thInput.AllowedUsers),
			AllowedServiceAccounts: valueStrings(thInput.AllowedServiceAccounts),
			AllowedTeams:           valueStrings(thInput.AllowedTeams),
		})
	}

	encodedData, err := encodeDataString(plan.Type,
		universalInputData{
			Role:     plan.Role.ValueString(),
			ClientID: plan.ClientID.ValueString(),
			TenantID: plan.TenantID.ValueString(),
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
			Type:        ttypes.ManagedIdentityType(plan.Type.ValueString()),
			Name:        plan.Name.ValueString(),
			Description: plan.Description.ValueString(),
			GroupPath:   plan.GroupPath.ValueString(),
			Data:        encodedData,
			AccessRules: accessRuleInputs,
		})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating managed identity",
			err.Error(),
		)
		return
	}

	// The CreateManagedIdentity call above does not return the access rules,
	// even though the rules are written to the database.
	createdAccessRules, err := t.client.ManagedIdentity.GetManagedIdentityAccessRules(ctx,
		&ttypes.GetManagedIdentityInput{ID: created.Metadata.ID})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting managed identity access rules",
			err.Error(),
		)
		return
	}
	created.AccessRules = createdAccessRules

	// Map the response body to the schema and update the plan with the computed attribute values.
	// Because the schema uses the Set type rather than the List type, make sure to set all fields.
	copyManagedIdentity(*created, &plan)

	// Set the response state to the fully-populated plan.
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (t *managedIdentityResource) Read(ctx context.Context,
	req resource.ReadRequest, resp *resource.ReadResponse) {

	// Get the current state.
	var state ManagedIdentityModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the managed identity from Tharsis.
	found, err := t.client.ManagedIdentity.GetManagedIdentity(ctx, &ttypes.GetManagedIdentityInput{
		ID: state.ID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading managed identity",
			err.Error(),
		)
		return
	}

	// Copy the from-Tharsis struct to the state.
	copyManagedIdentity(*found, &state)

	// Set the refreshed state.
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (t *managedIdentityResource) Update(ctx context.Context,
	req resource.UpdateRequest, resp *resource.UpdateResponse) {

	// Get the current state for its ID.
	var state ManagedIdentityModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Retrieve values from plan for the description and data.
	var plan ManagedIdentityModel
	diags = req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	encodedData, err := encodeDataString(plan.Type,
		universalInputData{
			Role:     plan.Role.ValueString(),
			ClientID: plan.ClientID.ValueString(),
			TenantID: plan.TenantID.ValueString(),
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
			ID:          state.ID.ValueString(),
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
	copyManagedIdentity(*updated, &plan)

	// Set the response state to the fully-populated plan.
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (t *managedIdentityResource) Delete(ctx context.Context,
	req resource.DeleteRequest, resp *resource.DeleteResponse) {

	// Get the current state.
	var state ManagedIdentityModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
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
		resp.Diagnostics.AddError(
			"Error deleting managed identity",
			err.Error(),
		)
		return
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
func copyManagedIdentity(src ttypes.ManagedIdentity, dest *ManagedIdentityModel) error {

	decodedData, err := decodeDataString(src.Data)
	if err != nil {
		return err
	}

	dest.ID = types.StringValue(src.Metadata.ID)
	dest.Type = types.StringValue(string(src.Type))
	dest.ResourcePath = types.StringValue(src.ResourcePath)
	dest.Name = types.StringValue(src.Name)
	dest.Description = types.StringValue(src.Description)
	dest.GroupPath = types.StringValue(getGroupPath(src.ResourcePath))
	if decodedData.Role != nil {
		dest.Role = types.StringValue(*decodedData.Role)
	}
	if decodedData.ClientID != nil {
		dest.ClientID = types.StringValue(*decodedData.ClientID)
	}
	if decodedData.TenantID != nil {
		dest.TenantID = types.StringValue(*decodedData.TenantID)
	}
	dest.Subject = types.StringValue(decodedData.Subject)

	for ruleIx, rule := range src.AccessRules {

		allowedUsers := []types.String{}
		for _, user := range rule.AllowedUsers {
			allowedUsers = append(allowedUsers, types.StringValue(user.Username))
		}

		allowedServiceAccounts := []types.String{}
		for _, serviceAccount := range rule.AllowedServiceAccounts {
			allowedServiceAccounts = append(allowedServiceAccounts,
				types.StringValue(serviceAccount.ResourcePath))
		}

		allowedTeams := []types.String{}
		for _, team := range rule.AllowedTeams {
			allowedTeams = append(allowedTeams, types.StringValue(team.Name))
		}

		dest.AccessRules[ruleIx] = ManagedIdentityAccessRuleModel{
			ID:                     types.StringValue(rule.Metadata.ID),
			RunStage:               types.StringValue(string(rule.RunStage)),
			ManagedIdentityID:      dest.ID,
			AllowedUsers:           allowedUsers,
			AllowedServiceAccounts: allowedServiceAccounts,
			AllowedTeams:           allowedTeams,
		}
	}
	// Must use time value from SDK/API.  Using time.Now() is not reliable.
	dest.LastUpdated = types.StringValue(src.Metadata.LastUpdatedTimestamp.Format(time.RFC850))

	return nil
}

// encodeDataString checks the role, client ID, tenant ID, and subject fields
// and then marshals them into the appropriate type and base64 encodes that.
func encodeDataString(managedIdentityType types.String, input universalInputData) (string, error) {
	type2 := ttypes.ManagedIdentityType(managedIdentityType.ValueString())

	// What to check depends on the type of managed identity this is.
	switch type2 {
	case ttypes.ManagedIdentityAWSFederated:
		if input.Role == "" {
			return "", fmt.Errorf("non-empty role is required for AWS managed identity")
		}
		if input.ClientID != "" {
			return "", fmt.Errorf("non-empty role is not allowed for AWS managed identity")
		}
		if input.TenantID != "" {
			return "", fmt.Errorf("non-empty role is not allowed for AWS managed identity")
		}
	case ttypes.ManagedIdentityAzureFederated:
		if input.Role != "" {
			return "", fmt.Errorf("non-empty role is not allowed for Azure managed identity")
		}
		if input.ClientID == "" {
			return "", fmt.Errorf("non-empty role is required for Azure managed identity")
		}
		if input.TenantID == "" {
			return "", fmt.Errorf("non-empty role is required for Azure managed identity")
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

// decodeDataString checks the role, client ID, tenant ID, and subject fields
// and then marshals them into the appropriate type and base64 encodes that.
func decodeDataString(encoded string) (*universalData, error) {

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}

	var result universalData
	if jErr := json.Unmarshal(decoded, &result); jErr != nil {
		return nil, err
	}

	return &result, nil
}

// getGroupPath returns the group path
func getGroupPath(resourcePath string) string {
	return resourcePath[:strings.LastIndex(resourcePath, "/")]
}

// The End.
