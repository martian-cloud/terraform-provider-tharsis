package tharsis

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ttypes "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ resource.Resource = managedIdentityAccessRuleResource{}
)

// Metadata returns the full name of the resource, including prefix, underscore, instance name.
func (t managedIdentityAccessRuleResource) Metadata(ctx context.Context,
	req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "tharsis_managed_identity_access_rule"
}

// The diagnostics return value is required by the interface even though this function returns only nil.
func (t managedIdentityAccessRuleResource) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
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
				MarkdownDescription: "List of email addresses of users allowed to use this rule.",
				Description:         "List of email addresses of users allowed to use this rule.",
				Optional:            true,
			},
			"allowed_service_accounts": {
				Type: types.SetType{
					ElemType: types.StringType,
				},
				MarkdownDescription: "List of resource paths of service accounts allowed to use this rule.",
				Description:         "List of resource paths of service accounts allowed to use this rule.",
				Optional:            true,
			},
			"allowed_teams": {
				Type: types.SetType{
					ElemType: types.StringType,
				},
				MarkdownDescription: "List of names of teams allowed to use this rule.",
				Description:         "List of names of teams allowed to use this rule.",
				Optional:            true,
			},
		},
	}, nil
}

type managedIdentityAccessRuleResource struct {
	provider tharsisProvider
}

func (t managedIdentityAccessRuleResource) Create(ctx context.Context,
	req resource.CreateRequest, resp *resource.CreateResponse) {

	// Retrieve values from plan.
	var plan ManagedIdentityAccessRuleModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build the access rule input.
	accessRuleInput := ttypes.CreateManagedIdentityAccessRuleInput{
		ManagedIdentityID:      plan.ManagedIdentityID.ValueString(),
		RunStage:               ttypes.JobType(plan.RunStage.ValueString()),
		AllowedUsers:           valueStrings(plan.AllowedUsers),
		AllowedServiceAccounts: valueStrings(plan.AllowedServiceAccounts),
		AllowedTeams:           valueStrings(plan.AllowedTeams),
	}

	// Create the managed identity access rule.
	created, err := t.provider.client.ManagedIdentity.CreateManagedIdentityAccessRule(ctx,
		&accessRuleInput)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating managed identity",
			err.Error(),
		)
		return
	}

	// Map the response body to the schema and update the plan with the computed attribute values.
	// Because the schema uses the Set type rather than the List type, make sure to set all fields.
	plan.ID = types.StringValue(created.Metadata.ID)
	plan.RunStage = types.StringValue(string(created.RunStage))
	plan.ManagedIdentityID = types.StringValue(created.ManagedIdentityID)

	allowedUsers := []types.String{}
	for _, user := range created.AllowedUsers {
		allowedUsers = append(allowedUsers, types.StringValue(user.Username))
	}
	plan.AllowedUsers = allowedUsers

	allowedServiceAccounts := []types.String{}
	for _, serviceAccount := range created.AllowedServiceAccounts {
		allowedServiceAccounts = append(allowedServiceAccounts, types.StringValue(serviceAccount.ResourcePath))
	}
	plan.AllowedServiceAccounts = allowedServiceAccounts

	allowedTeams := []types.String{}
	for _, team := range created.AllowedTeams {
		allowedTeams = append(allowedTeams, types.StringValue(team.Name))
	}
	plan.AllowedTeams = allowedTeams

	// Set the response state to the fully-populated plan.
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (t managedIdentityAccessRuleResource) Read(ctx context.Context,
	req resource.ReadRequest, resp *resource.ReadResponse) {

	// Get the current state.
	var state ManagedIdentityAccessRuleModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the managed identity access rule from Tharsis.
	found, err := t.provider.client.ManagedIdentity.GetManagedIdentityAccessRule(ctx,
		&ttypes.GetManagedIdentityAccessRuleInput{
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
	state.ID = types.StringValue(found.Metadata.ID)
	state.RunStage = types.StringValue(string(found.RunStage))
	state.ManagedIdentityID = types.StringValue(found.ManagedIdentityID)

	state.AllowedUsers = []types.String{}
	for _, user := range found.AllowedUsers {
		state.AllowedUsers = append(state.AllowedUsers, types.StringValue(user.Username))
	}

	state.AllowedServiceAccounts = []types.String{}
	for _, serviceAccount := range found.AllowedServiceAccounts {
		state.AllowedServiceAccounts = append(state.AllowedServiceAccounts,
			types.StringValue(serviceAccount.ResourcePath))
	}

	state.AllowedTeams = []types.String{}
	for _, team := range found.AllowedTeams {
		state.AllowedTeams = append(state.AllowedTeams, types.StringValue(team.Name))
	}

	// Set the refreshed state.
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (t managedIdentityAccessRuleResource) Update(ctx context.Context,
	req resource.UpdateRequest, resp *resource.UpdateResponse) {

	// Get the current state for its ID.
	var state ManagedIdentityAccessRuleModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Retrieve values from plan for the fields to modify.
	var plan ManagedIdentityAccessRuleModel
	diags = req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update the access rule via Tharsis.
	// The ID is used to find the record to update.
	// The other fields are modified.
	updated, err := t.provider.client.ManagedIdentity.UpdateManagedIdentityAccessRule(ctx,
		&ttypes.UpdateManagedIdentityAccessRuleInput{
			ID:                     state.ID.ValueString(),
			RunStage:               ttypes.JobType(plan.RunStage.ValueString()),
			AllowedUsers:           valueStrings(plan.AllowedUsers),
			AllowedServiceAccounts: valueStrings(plan.AllowedServiceAccounts),
			AllowedTeams:           valueStrings(plan.AllowedTeams),
		})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating managed identity",
			err.Error(),
		)
		return
	}

	// Copy fields returned by Tharsis to the plan.  Apparently, must copy all fields, not just the computed fields.
	plan.ID = types.StringValue(updated.Metadata.ID)
	plan.RunStage = types.StringValue(string(updated.RunStage))
	plan.ManagedIdentityID = types.StringValue(updated.ManagedIdentityID)

	plan.AllowedUsers = []types.String{}
	for _, user := range updated.AllowedUsers {
		plan.AllowedUsers = append(plan.AllowedUsers, types.StringValue(user.Username))
	}

	plan.AllowedServiceAccounts = []types.String{}
	for _, serviceAccount := range updated.AllowedServiceAccounts {
		plan.AllowedServiceAccounts = append(plan.AllowedServiceAccounts, types.StringValue(serviceAccount.ResourcePath))
	}

	plan.AllowedTeams = []types.String{}
	for _, team := range updated.AllowedTeams {
		plan.AllowedTeams = append(plan.AllowedTeams, types.StringValue(team.Name))
	}

	// Set the response state to the fully-populated plan.
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (t managedIdentityAccessRuleResource) Delete(ctx context.Context,
	req resource.DeleteRequest, resp *resource.DeleteResponse) {

	// Get the current state.
	var state ManagedIdentityAccessRuleModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the managed identity via Tharsis.
	// The ID is used to find the record to delete.
	err := t.provider.client.ManagedIdentity.DeleteManagedIdentityAccessRule(ctx,
		&ttypes.DeleteManagedIdentityAccessRuleInput{
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

// valueStrings converts a slice of types.String to a slice of strings.
func valueStrings(arg []types.String) []string {
	result := make([]string, len(arg))
	for ix, bigSValue := range arg {
		result[ix] = bigSValue.ValueString()
	}
	return result
}

// The End.
