package tharsis

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ttypes "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

// Managed identity models:

type managedIdentityModel struct {
	ID           types.String                     `tfsdk:"id"`
	Type         types.String                     `tfsdk:"type"`
	ResourcePath types.String                     `tfsdk:"resource_path"`
	Name         types.String                     `tfsdk:"name"`
	Description  types.String                     `tfsdk:"description"`
	GroupPath    types.String                     `tfsdk:"group_path"`
	CreatedBy    types.String                     `tfsdk:"created_by"`
	Data         types.String                     `tfsdk:"data"` // less overhead than a types.List of int[8]s
	AccessRules  []managedIdentityAccessRuleModel `tfsdk:"access_rules"`
	LastUpdated  types.String                     `tfsdk:"last_updated"`
}

type managedIdentityAccessRuleModel struct {
	ID                     types.String   `tfsdk:"id"`
	RunStage               types.String   `tfsdk:"run_stage"`
	ManagedIdentityID      types.String   `tfsdk:"managed_identity_id"`
	AllowedUsers           []types.String `tfsdk:"allowed_users"`
	AllowedServiceAccounts []types.String `tfsdk:"allowed_service_accounts"`
	AllowedTeams           []types.String `tfsdk:"allowed_teams"`
}

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ resource.Resource = managedIdentitiesResource{}
)

// Metadata returns the full name of the resource, including prefix, underscore, instance name.
func (t managedIdentitiesResource) Metadata(ctx context.Context,
	req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "tharsis_managed_identities"
}

// The diagnostics return value is required by the interface even though this function returns only nil.
func (t managedIdentitiesResource) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	description := "Defines and manages a managed identity."

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
			"created_by": {
				Type:                types.StringType,
				MarkdownDescription: "User email address, service account path, or other identifier of creator of the managed identity.",
				Description:         "User email address, service account path, or other identifier of creator of the managed identity.",
				Optional:            true,
			},
			"data": {
				Type:                types.StringType,
				MarkdownDescription: "IAM role or tenant and client IDs of the managed identity.",
				Description:         "IAM role or tenant and client IDs of the managed identity.",
				Required:            true,
			},
			"access_rules": {
				MarkdownDescription: "List of access rules for the managed identity.",
				Description:         "List of access rules for the managed identity.",
				Optional:            true,
				Attributes: tfsdk.ListNestedAttributes(map[string]tfsdk.Attribute{
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
						Computed:            true,
					},
					"allowed_users": {
						Type: types.ListType{
							ElemType: types.StringType,
						},
						MarkdownDescription: "List of email addresses of users allowed to use this rule.",
						Description:         "List of email addresses of users allowed to use this rule.",
						Optional:            true,
					},
					"allowed_service_accounts": {
						Type: types.ListType{
							ElemType: types.StringType,
						},
						MarkdownDescription: "List of resource paths of service accounts allowed to use this rule.",
						Description:         "List of resource paths of service accounts allowed to use this rule.",
						Optional:            true,
					},
					"allowed_teams": {
						Type: types.ListType{
							ElemType: types.StringType,
						},
						MarkdownDescription: "List of names of teams allowed to use this rule.",
						Description:         "List of names of teams allowed to use this rule.",
						Optional:            true,
					},
				}),
			},
			"last_updated": {
				Type:                types.StringType,
				MarkdownDescription: "Timestamp when this managed identity was most recently updated.",
				Description:         "Timestamp when this managed identity was most recently updated.",
				Computed:            true,
			},
		},
	}, nil
}

type managedIdentitiesResource struct {
	provider tharsisProvider
}

func (d managedIdentitiesResource) Create(ctx context.Context,
	req resource.CreateRequest, resp *resource.CreateResponse) {

	// Retrieve values from plan.
	var plan managedIdentityModel
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
			AllowedUsers:           stringValues(thInput.AllowedUsers),
			AllowedServiceAccounts: stringValues(thInput.AllowedServiceAccounts),
			AllowedTeams:           stringValues(thInput.AllowedTeams),
		})
	}

	// Create the managed identity.
	created, err := d.provider.client.ManagedIdentity.CreateManagedIdentity(ctx,
		&ttypes.CreateManagedIdentityInput{
			Type:        ttypes.ManagedIdentityType(plan.Type.ValueString()),
			Name:        plan.Name.ValueString(),
			Description: plan.Description.ValueString(),
			GroupPath:   plan.GroupPath.ValueString(),
			Data:        plan.Data.ValueString(),
			AccessRules: accessRuleInputs,
		})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating managed identity",
			err.Error(),
		)
		return
	}

	// Map the response body to the schema and update the plan with the computed attribute values.
	plan.ID = types.StringValue(created.Metadata.ID)
	for ruleIx, rule := range created.AccessRules {

		allowedUsers := []types.String{}
		for _, user := range rule.AllowedUsers {
			allowedUsers = append(allowedUsers, types.StringValue(user.Email))
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

		plan.AccessRules[ruleIx] = managedIdentityAccessRuleModel{
			ID:                     types.StringValue(rule.Metadata.ID),
			RunStage:               types.StringValue(string(rule.RunStage)),
			ManagedIdentityID:      plan.ID,
			AllowedUsers:           allowedUsers,
			AllowedServiceAccounts: allowedServiceAccounts,
			AllowedTeams:           allowedTeams,
		}
	}
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	// Set the response state to the fully-populated plan.
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (d managedIdentitiesResource) Read(ctx context.Context,
	req resource.ReadRequest, resp *resource.ReadResponse) {

	// Get the current state.
	var state managedIdentityModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the managed identity from Tharsis.
	found, err := d.provider.client.ManagedIdentity.GetManagedIdentity(ctx, &ttypes.GetManagedIdentityInput{
		ID: state.ResourcePath.ValueString(),
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
	state.Type = types.StringValue(string(found.Type))
	state.ResourcePath = types.StringValue(found.ResourcePath)
	state.Name = types.StringValue(found.Name)
	state.Description = types.StringValue(found.Description)
	state.Data = types.StringValue(found.Data)
	for ruleIx, rule := range found.AccessRules {

		allowedUsers := []types.String{}
		for _, user := range rule.AllowedUsers {
			allowedUsers = append(allowedUsers, types.StringValue(user.Email))
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

		state.AccessRules[ruleIx] = managedIdentityAccessRuleModel{
			ID:                     types.StringValue(rule.Metadata.ID),
			RunStage:               types.StringValue(string(rule.RunStage)),
			ManagedIdentityID:      state.ID,
			AllowedUsers:           allowedUsers,
			AllowedServiceAccounts: allowedServiceAccounts,
			AllowedTeams:           allowedTeams,
		}
	}
	state.LastUpdated = types.StringValue(found.Metadata.LastUpdatedTimestamp.Format(time.RFC850))

	// Set the refreshed state.
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (d managedIdentitiesResource) Update(ctx context.Context,
	req resource.UpdateRequest, resp *resource.UpdateResponse) {

	// Retrieve values from plan.
	var plan managedIdentityModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update the managed identity via Tharsis.
	// The ID is used to find the record to update.
	// The description and data are modified.
	updated, err := d.provider.client.ManagedIdentity.UpdateManagedIdentity(ctx,
		&ttypes.UpdateManagedIdentityInput{
			ID:          plan.ID.ValueString(),
			Description: plan.Description.ValueString(),
			Data:        plan.Data.ValueString(),
		})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating managed identity",
			err.Error(),
		)
		return
	}

	// Update the resource state with updated values and timestamp.
	plan.Description = types.StringValue(updated.Description)
	plan.Data = types.StringValue(updated.Data)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	// Set the response state to the fully-populated plan.
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (d managedIdentitiesResource) Delete(ctx context.Context,
	req resource.DeleteRequest, resp *resource.DeleteResponse) {

	// Get the current state.
	var state managedIdentityModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the managed identity via Tharsis.
	// The ID is used to find the record to delete.
	err := d.provider.client.ManagedIdentity.DeleteManagedIdentity(ctx,
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

// stringValues converts a slice of types.String to a slice of strings.
func stringValues(arg []types.String) []string {
	result := make([]string, len(arg))
	for ix, bigSValue := range arg {
		result[ix] = bigSValue.ValueString()
	}
	return result
}

// The End.
