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

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ resource.Resource = managedIdentityResource{}
)

// Metadata returns the full name of the resource, including prefix, underscore, instance name.
func (t managedIdentityResource) Metadata(ctx context.Context,
	req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "tharsis_managed_identity"
}

// The diagnostics return value is required by the interface even though this function returns only nil.
func (t managedIdentityResource) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
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
				Attributes: tfsdk.SetNestedAttributes(map[string]tfsdk.Attribute{
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

type managedIdentityResource struct {
	provider tharsisProvider
}

func (t managedIdentityResource) Create(ctx context.Context,
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

	// FIXME: Must take something more human-readable as input for data.
	// The provider must (marshal and) base64-encode it.
	// Brandon to decide whether to use a union type or some other arrangement.

	// Create the managed identity.
	created, err := t.provider.client.ManagedIdentity.CreateManagedIdentity(ctx,
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

	// The CreateManagedIdentity call above does not return the access rules,
	// even though the rules are written to the database.
	createdAccessRules, err := t.provider.client.ManagedIdentity.GetManagedIdentityAccessRules(ctx,
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

func (t managedIdentityResource) Read(ctx context.Context,
	req resource.ReadRequest, resp *resource.ReadResponse) {

	// Get the current state.
	var state ManagedIdentityModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the managed identity from Tharsis.
	found, err := t.provider.client.ManagedIdentity.GetManagedIdentity(ctx, &ttypes.GetManagedIdentityInput{
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
	copyManagedIdentity(*found, &state)

	// Set the refreshed state.
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (t managedIdentityResource) Update(ctx context.Context,
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

	// Update the managed identity via Tharsis.
	// The ID is used to find the record to update.
	// The description and data are modified.
	updated, err := t.provider.client.ManagedIdentity.UpdateManagedIdentity(ctx,
		&ttypes.UpdateManagedIdentityInput{
			ID:          state.ID.ValueString(),
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

	// Copy all fields returned by Tharsis back into the plan.
	copyManagedIdentity(*updated, &plan)

	// Set the response state to the fully-populated plan.
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (t managedIdentityResource) Delete(ctx context.Context,
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
	err := t.provider.client.ManagedIdentity.DeleteManagedIdentity(ctx,
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

// copyManagedIdentity copies the contents of a managed identity.
// It is intended to copy from a struct returned by Tharsis to a Terraform plan or state.
func copyManagedIdentity(src ttypes.ManagedIdentity, dest *ManagedIdentityModel) {

	dest.ID = types.StringValue(src.Metadata.ID)
	dest.Type = types.StringValue(string(src.Type))
	dest.ResourcePath = types.StringValue(src.ResourcePath)
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
	dest.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
}

// The End.
