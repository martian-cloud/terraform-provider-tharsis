package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	tharsis "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg"
	ttypes "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

// ManagedIdentityAccessRuleModel is the model for a managed identity access rule.
type ManagedIdentityAccessRuleModel struct {
	ID                     types.String   `tfsdk:"id"`
	RunStage               types.String   `tfsdk:"run_stage"`
	ManagedIdentityID      types.String   `tfsdk:"managed_identity_id"`
	AllowedUsers           []types.String `tfsdk:"allowed_users"`
	AllowedServiceAccounts []types.String `tfsdk:"allowed_service_accounts"`
	AllowedTeams           []types.String `tfsdk:"allowed_teams"`
}

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ resource.Resource                = &managedIdentityAccessRuleResource{}
	_ resource.ResourceWithConfigure   = &managedIdentityAccessRuleResource{}
	_ resource.ResourceWithImportState = &managedIdentityAccessRuleResource{}
)

// NewManagedIdentityAccessRuleResource is a helper function to simplify the provider implementation.
func NewManagedIdentityAccessRuleResource() resource.Resource {
	return &managedIdentityAccessRuleResource{}
}

type managedIdentityAccessRuleResource struct {
	client *tharsis.Client
}

// Metadata returns the full name of the resource, including prefix, underscore, instance name.
func (t *managedIdentityAccessRuleResource) Metadata(ctx context.Context,
	req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "tharsis_managed_identity_access_rule"
}

// The diagnostics return value is required by the interface even though this function returns only nil.
func (t *managedIdentityAccessRuleResource) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	description := "Defines and manages a managed identity access rule."

	return tfsdk.Schema{
		Version: 1,

		MarkdownDescription: description,
		Description:         description,

		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:                types.StringType,
				MarkdownDescription: "String identifier of the access rule.",
				Description:         "String identifier of the access rule.",
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					resource.UseStateForUnknown(),
				},
			},
			"run_stage": {
				Type:                types.StringType,
				MarkdownDescription: "Type of job, plan or apply.",
				Description:         "Type of job, plan or apply.",
				Required:            true,
			},
			"managed_identity_id": {
				Type:                types.StringType,
				MarkdownDescription: "String identifier of the connected managed identity.",
				Description:         "String identifier of the connected managed identity.",
				Required:            true,
			},
			"allowed_users": {
				Type: types.SetType{
					ElemType: types.StringType,
				},
				MarkdownDescription: "List of usernames allowed to use the managed identity associated with this rule.",
				Description:         "List of usernames allowed to use the managed identity associated with this rule.",
				Optional:            true,
			},
			"allowed_service_accounts": {
				Type: types.SetType{
					ElemType: types.StringType,
				},
				MarkdownDescription: "List of resource paths of service accounts allowed to use the managed identity associated with this rule.",
				Description:         "List of resource paths of service accounts allowed to use the managed identity associated with this rule.",
				Optional:            true,
			},
			"allowed_teams": {
				Type: types.SetType{
					ElemType: types.StringType,
				},
				MarkdownDescription: "List of names of teams allowed to use the managed identity associated with this rule.",
				Description:         "List of names of teams allowed to use the managed identity associated with this rule.",
				Optional:            true,
			},
		},
	}, nil
}

// Configure lets the provider implement the ResourceWithConfigure interface.
func (t *managedIdentityAccessRuleResource) Configure(_ context.Context,
	req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	t.client = req.ProviderData.(*tharsis.Client)
}

func (t *managedIdentityAccessRuleResource) Create(ctx context.Context,
	req resource.CreateRequest, resp *resource.CreateResponse) {

	// Retrieve values from accessRule.
	var accessRule ManagedIdentityAccessRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &accessRule)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build the access rule input.
	accessRuleInput := ttypes.CreateManagedIdentityAccessRuleInput{
		ManagedIdentityID:      accessRule.ManagedIdentityID.ValueString(),
		RunStage:               ttypes.JobType(accessRule.RunStage.ValueString()),
		AllowedUsers:           t.valueStrings(accessRule.AllowedUsers),
		AllowedServiceAccounts: t.valueStrings(accessRule.AllowedServiceAccounts),
		AllowedTeams:           t.valueStrings(accessRule.AllowedTeams),
	}

	// Create the managed identity access rule.
	created, err := t.client.ManagedIdentity.CreateManagedIdentityAccessRule(ctx,
		&accessRuleInput)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating managed identity access rule",
			err.Error(),
		)
		return
	}

	// Map the response body to the schema and update the plan with the computed attribute values.
	// Because the schema uses the Set type rather than the List type, make sure to set all fields.
	accessRule.ID = types.StringValue(created.Metadata.ID)
	accessRule.RunStage = types.StringValue(string(created.RunStage))
	accessRule.ManagedIdentityID = types.StringValue(created.ManagedIdentityID)

	allowedUsers := []string{}
	for _, user := range created.AllowedUsers {
		allowedUsers = append(allowedUsers, user.Username)
	}
	accessRule.AllowedUsers = t.stringValues(allowedUsers)

	allowedServiceAccounts := []string{}
	for _, serviceAccount := range created.AllowedServiceAccounts {
		allowedServiceAccounts = append(allowedServiceAccounts, serviceAccount.ResourcePath)
	}
	accessRule.AllowedServiceAccounts = t.stringValues(allowedServiceAccounts)

	allowedTeams := []string{}
	for _, team := range created.AllowedTeams {
		allowedTeams = append(allowedTeams, team.Name)
	}
	accessRule.AllowedTeams = t.stringValues(allowedTeams)

	// Set the response state to the fully-populated plan, whether or not there is an error.
	resp.Diagnostics.Append(resp.State.Set(ctx, accessRule)...)
}

func (t *managedIdentityAccessRuleResource) Read(ctx context.Context,
	req resource.ReadRequest, resp *resource.ReadResponse) {

	// Get the current state.
	var state ManagedIdentityAccessRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the managed identity access rule from Tharsis.
	found, err := t.client.ManagedIdentity.GetManagedIdentityAccessRule(ctx,
		&ttypes.GetManagedIdentityAccessRuleInput{
			ID: state.ID.ValueString(),
		})
	if err != nil {

		// Handle the case that the access rule no longer exists.
		if t.isErrorRuleNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error reading managed identity access rule",
			err.Error(),
		)
		return
	}

	// Copy the from-Tharsis run stage to the state.
	state.RunStage = types.StringValue(string(found.RunStage))

	// When this Read method is called during a "terraform import" operation, state.ManagedIdentityID is null.
	// In that case, it is necessary to copy ManagedIdentityID from found to state.
	if state.ManagedIdentityID.Null {
		state.ManagedIdentityID = types.StringValue(found.ManagedIdentityID)
	}

	allowedUsers := []string{}
	for _, user := range found.AllowedUsers {
		allowedUsers = append(allowedUsers, user.Username)
	}
	state.AllowedUsers = t.stringValues(allowedUsers)

	allowedServiceAccounts := []string{}
	for _, serviceAccount := range found.AllowedServiceAccounts {
		allowedServiceAccounts = append(allowedServiceAccounts, serviceAccount.ResourcePath)
	}
	state.AllowedServiceAccounts = t.stringValues(allowedServiceAccounts)

	allowedTeams := []string{}
	for _, team := range found.AllowedTeams {
		allowedTeams = append(allowedTeams, team.Name)
	}
	state.AllowedTeams = t.stringValues(allowedTeams)

	// Set the refreshed state, whether or not there is an error.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (t *managedIdentityAccessRuleResource) Update(ctx context.Context,
	req resource.UpdateRequest, resp *resource.UpdateResponse) {

	// Retrieve values from plan for the fields to modify.
	var plan ManagedIdentityAccessRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update the access rule via Tharsis.
	// The ID is used to find the record to update.
	// The other fields are modified.
	updated, err := t.client.ManagedIdentity.UpdateManagedIdentityAccessRule(ctx,
		&ttypes.UpdateManagedIdentityAccessRuleInput{
			ID:                     plan.ID.ValueString(),
			RunStage:               ttypes.JobType(plan.RunStage.ValueString()),
			AllowedUsers:           t.valueStrings(plan.AllowedUsers),
			AllowedServiceAccounts: t.valueStrings(plan.AllowedServiceAccounts),
			AllowedTeams:           t.valueStrings(plan.AllowedTeams),
		})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating managed identity access rule",
			err.Error(),
		)
		return
	}

	// Copy fields returned by Tharsis to the plan.  Apparently, must copy all fields, not just the computed fields.
	plan.RunStage = types.StringValue(string(updated.RunStage))

	allowedUsers := []string{}
	for _, user := range updated.AllowedUsers {
		allowedUsers = append(allowedUsers, user.Username)
	}
	plan.AllowedUsers = t.stringValues(allowedUsers)

	allowedServiceAccounts := []string{}
	for _, serviceAccount := range updated.AllowedServiceAccounts {
		allowedServiceAccounts = append(allowedServiceAccounts, serviceAccount.ResourcePath)
	}
	plan.AllowedServiceAccounts = t.stringValues(allowedServiceAccounts)

	allowedTeams := []string{}
	for _, team := range updated.AllowedTeams {
		allowedTeams = append(allowedTeams, team.Name)
	}
	plan.AllowedTeams = t.stringValues(allowedTeams)

	// Set the response state to the fully-populated plan, error or not.
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (t *managedIdentityAccessRuleResource) Delete(ctx context.Context,
	req resource.DeleteRequest, resp *resource.DeleteResponse) {

	// Get the current state.
	var state ManagedIdentityAccessRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the access rule via Tharsis.
	// The ID is used to find the record to delete.
	err := t.client.ManagedIdentity.DeleteManagedIdentityAccessRule(ctx,
		&ttypes.DeleteManagedIdentityAccessRuleInput{
			ID: state.ID.ValueString(),
		})
	if err != nil {

		// Handle the case that the access rule no longer exists.
		if t.isErrorRuleNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error deleting managed identity access rule",
			err.Error(),
		)
	}
}

// ImportState helps the provider implement the ResourceWithImportState interface.
func (t *managedIdentityAccessRuleResource) ImportState(ctx context.Context,
	req resource.ImportStateRequest, resp *resource.ImportStateResponse) {

	// Retrieve import ID and save to id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

	// Setting of the ManagedIdentityID field during import is handled in the Read method.

}

// valueStrings converts a slice of types.String to a slice of strings.
func (t *managedIdentityAccessRuleResource) valueStrings(arg []types.String) []string {
	result := make([]string, len(arg))
	for ix, bigSValue := range arg {
		result[ix] = bigSValue.ValueString()
	}
	return result
}

// stringValues converts a slice of strings to a slice of types.String
func (t *managedIdentityAccessRuleResource) stringValues(arg []string) []types.String {
	argLength := len(arg)

	if argLength == 0 {
		// Terraform wants nil rather than an empty slice.
		return nil
	}

	result := make([]types.String, argLength)
	for ix, bigSValue := range arg {
		result[ix] = types.StringValue(bigSValue)
	}
	return result
}

// isErrorRuleNotFound returns true iff the error message is that an access rule was not found.
// Don't check the ID, because the available ID is the global id, while the ID in the message is a local ID.
// In theory, we should never see a message that some other ID was not found.
func (t *managedIdentityAccessRuleResource) isErrorRuleNotFound(e error) bool {
	// Omission of the leading 'M' is intentional in case the SDK changes to lowercase.
	return strings.Contains(e.Error(), "anaged identity access rule with ID ") &&
		strings.Contains(e.Error(), " not found")
}

// The End.
