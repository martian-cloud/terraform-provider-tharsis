package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/martian-cloud/terraform-provider-tharsis/internal/modifiers"
	tharsis "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg"
	ttypes "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

// ModuleAttestationPolicyModel is used in access rules to verify that a
// module has an in-toto attestation that is signed with the specified public key and an optional
// predicate type
type ModuleAttestationPolicyModel struct {
	PredicateType *string `tfsdk:"predicate_type"`
	PublicKey     string  `tfsdk:"public_key"`
}

// FromTerraform5Value converts from Terraform values to Go equivalent.
func (e *ModuleAttestationPolicyModel) FromTerraform5Value(val tftypes.Value) error {

	v := map[string]tftypes.Value{}
	err := val.As(&v)
	if err != nil {
		return err
	}

	err = v["predicate_type"].As(&e.PredicateType)
	if err != nil {
		return err
	}

	err = v["public_key"].As(&e.PublicKey)
	if err != nil {
		return err
	}

	return nil
}

// ManagedIdentityAccessRuleModel is the model for a managed identity access rule.
type ManagedIdentityAccessRuleModel struct {
	ID                        types.String        `tfsdk:"id"`
	Type                      types.String        `tfsdk:"type"`
	RunStage                  types.String        `tfsdk:"run_stage"`
	ManagedIdentityID         types.String        `tfsdk:"managed_identity_id"`
	ModuleAttestationPolicies basetypes.ListValue `tfsdk:"module_attestation_policies"`
	AllowedUsers              basetypes.SetValue  `tfsdk:"allowed_users"`
	AllowedServiceAccounts    basetypes.SetValue  `tfsdk:"allowed_service_accounts"`
	AllowedTeams              basetypes.SetValue  `tfsdk:"allowed_teams"`
}

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ resource.Resource                = (*managedIdentityAccessRuleResource)(nil)
	_ resource.ResourceWithConfigure   = (*managedIdentityAccessRuleResource)(nil)
	_ resource.ResourceWithImportState = (*managedIdentityAccessRuleResource)(nil)
)

// NewManagedIdentityAccessRuleResource is a helper function to simplify the provider implementation.
func NewManagedIdentityAccessRuleResource() resource.Resource {
	return &managedIdentityAccessRuleResource{}
}

type managedIdentityAccessRuleResource struct {
	client *tharsis.Client
}

// Metadata returns the full name of the resource, including prefix, underscore, instance name.
func (t *managedIdentityAccessRuleResource) Metadata(_ context.Context,
	_ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "tharsis_managed_identity_access_rule"
}

func (t *managedIdentityAccessRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	description := "Defines and manages a managed identity access rule."

	resp.Schema = schema.Schema{
		Version:             1,
		MarkdownDescription: description,
		Description:         description,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "String identifier of the access rule.",
				Description:         "String identifier of the access rule.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Type of access rule: eligible_principals or module_attestation.",
				Description:         "Type of access rule: eligible_principals or module_attestation.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"run_stage": schema.StringAttribute{
				MarkdownDescription: "Type of job, plan or apply.",
				Description:         "Type of job, plan or apply.",
				Required:            true,
				// Can be updated in place, so no RequiresReplace plan modifier.
			},
			"managed_identity_id": schema.StringAttribute{
				MarkdownDescription: "String identifier of the connected managed identity.",
				Description:         "String identifier of the connected managed identity.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"allowed_users": schema.SetAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "List of usernames allowed to use the managed identity associated with this rule.",
				Description:         "List of usernames allowed to use the managed identity associated with this rule.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Set{
					modifiers.SetDefault([]attr.Value{}),
				},
				// Can be updated in place, so no RequiresReplace plan modifier.
			},
			"allowed_service_accounts": schema.SetAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "List of resource paths of service accounts allowed to use the managed identity associated with this rule.",
				Description:         "List of resource paths of service accounts allowed to use the managed identity associated with this rule.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Set{
					modifiers.SetDefault([]attr.Value{}),
				},
				// Can be updated in place, so no RequiresReplace plan modifier.
			},
			"allowed_teams": schema.SetAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "List of names of teams allowed to use the managed identity associated with this rule.",
				Description:         "List of names of teams allowed to use the managed identity associated with this rule.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Set{
					modifiers.SetDefault([]attr.Value{}),
				},
				// Can be updated in place, so no RequiresReplace plan modifier.
			},
			"module_attestation_policies": schema.ListNestedAttribute{
				MarkdownDescription: "Used to verify that a module has an in-toto attestation that is signed with the specified public key and an optional predicate type.",
				Description:         "Used to verify that a module has an in-toto attestation that is signed with the specified public key and an optional predicate type.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.List{
					modifiers.ListDefault([]attr.Value{}),
				},
				// Can be updated in place, so no RequiresReplace plan modifier.
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"predicate_type": schema.StringAttribute{
							MarkdownDescription: "Optional predicate type for this attestation policy.",
							Description:         "Optional predicate type for this attestation policy.",
							Optional:            true,
						},
						"public_key": schema.StringAttribute{
							MarkdownDescription: "Public key in PEM format for this attestation policy.",
							Description:         "Public key in PEM format for this attestation policy.",
							Required:            true,
						},
					},
				},
			},
		},
	}
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

	policies, err := t.copyAttestationPoliciesToInput(ctx, &accessRule.ModuleAttestationPolicies)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error while copying module attestation policies to Tharsis input",
			err.Error(),
		)
		return
	}

	allowedUsersInput, err := t.valueStrings(ctx, accessRule.AllowedUsers)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error while copying access rule AllowedUsers to Tharsis input",
			err.Error(),
		)
		return
	}

	allowedServiceAccountsInput, err := t.valueStrings(ctx, accessRule.AllowedServiceAccounts)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error while copying access rule AllowedServiceAccounts to Tharsis input",
			err.Error(),
		)
		return
	}

	allowedTeamsInput, err := t.valueStrings(ctx, accessRule.AllowedTeams)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error while copying access rule AllowedTeams to Tharsis input",
			err.Error(),
		)
		return
	}

	// Build the access rule input.
	accessRuleInput := ttypes.CreateManagedIdentityAccessRuleInput{
		ManagedIdentityID:         accessRule.ManagedIdentityID.ValueString(),
		Type:                      ttypes.ManagedIdentityAccessRuleType(accessRule.Type.ValueString()),
		RunStage:                  ttypes.JobType(accessRule.RunStage.ValueString()),
		AllowedUsers:              allowedUsersInput,
		AllowedServiceAccounts:    allowedServiceAccountsInput,
		AllowedTeams:              allowedTeamsInput,
		ModuleAttestationPolicies: policies,
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
	accessRule.Type = types.StringValue(string(created.Type))
	accessRule.RunStage = types.StringValue(string(created.RunStage))
	accessRule.ManagedIdentityID = types.StringValue(created.ManagedIdentityID)

	allowedUsers := []attr.Value{}
	for _, user := range created.AllowedUsers {
		allowedUsers = append(allowedUsers, types.StringValue(user.Username))
	}

	var diags diag.Diagnostics
	accessRule.AllowedUsers, diags = types.SetValue(types.StringType, allowedUsers)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	allowedServiceAccounts := []attr.Value{}
	for _, serviceAccount := range created.AllowedServiceAccounts {
		allowedServiceAccounts = append(allowedServiceAccounts, types.StringValue(serviceAccount.ResourcePath))
	}

	accessRule.AllowedServiceAccounts, diags = types.SetValue(types.StringType, allowedServiceAccounts)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	allowedTeams := []attr.Value{}
	for _, team := range created.AllowedTeams {
		allowedTeams = append(allowedTeams, types.StringValue(team.Name))
	}

	accessRule.AllowedTeams, diags = types.SetValue(types.StringType, allowedTeams)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	accessRule.ModuleAttestationPolicies, diags = t.toProviderAttestationPolicies(ctx, created.ModuleAttestationPolicies)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

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

		// Handle the case that the access rule no longer exists if that fact is reported by returning an error.
		if tharsis.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error reading managed identity access rule",
			err.Error(),
		)
		return
	}

	// Copy the from-Tharsis run stage to the state, but not if it no longer exists.
	state.RunStage = types.StringValue(string(found.RunStage))
	state.Type = types.StringValue(string(found.Type))

	// When this Read method is called during a "terraform import" operation, state.ManagedIdentityID is null.
	// In that case, it is necessary to copy ManagedIdentityID from found to state.
	if state.ManagedIdentityID.IsNull() {
		state.ManagedIdentityID = types.StringValue(found.ManagedIdentityID)
	}

	allowedUsers := []attr.Value{}
	for _, user := range found.AllowedUsers {
		allowedUsers = append(allowedUsers, types.StringValue(user.Username))
	}

	var diags diag.Diagnostics
	state.AllowedUsers, diags = types.SetValue(types.StringType, allowedUsers)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	allowedServiceAccounts := []attr.Value{}
	for _, serviceAccount := range found.AllowedServiceAccounts {
		allowedServiceAccounts = append(allowedServiceAccounts, types.StringValue(serviceAccount.ResourcePath))
	}

	state.AllowedServiceAccounts, diags = types.SetValue(types.StringType, allowedServiceAccounts)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	allowedTeams := []attr.Value{}
	for _, team := range found.AllowedTeams {
		allowedTeams = append(allowedTeams, types.StringValue(team.Name))
	}

	state.AllowedTeams, diags = types.SetValue(types.StringType, allowedTeams)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	state.ModuleAttestationPolicies, diags = t.toProviderAttestationPolicies(ctx, found.ModuleAttestationPolicies)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

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

	policies, err := t.copyAttestationPoliciesToInput(ctx, &plan.ModuleAttestationPolicies)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to copy module attestation policies to Tharsis input",
			err.Error(),
		)
		return
	}

	allowedUsersInput, err := t.valueStrings(ctx, plan.AllowedUsers)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error while copying access rule AllowedUsers to Tharsis input",
			err.Error(),
		)
		return
	}

	allowedServiceAccountsInput, err := t.valueStrings(ctx, plan.AllowedServiceAccounts)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error while copying access rule AllowedServiceAccounts to Tharsis input",
			err.Error(),
		)
		return
	}

	allowedTeamsInput, err := t.valueStrings(ctx, plan.AllowedTeams)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error while copying access rule AllowedTeams to Tharsis input",
			err.Error(),
		)
		return
	}

	// Update the access rule via Tharsis.
	// The ID is used to find the record to update.
	// The other fields are modified.
	updated, err := t.client.ManagedIdentity.UpdateManagedIdentityAccessRule(ctx,
		&ttypes.UpdateManagedIdentityAccessRuleInput{
			ID:                        plan.ID.ValueString(),
			RunStage:                  ttypes.JobType(plan.RunStage.ValueString()),
			AllowedUsers:              allowedUsersInput,
			AllowedServiceAccounts:    allowedServiceAccountsInput,
			AllowedTeams:              allowedTeamsInput,
			ModuleAttestationPolicies: policies,
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
	plan.Type = types.StringValue(string(updated.Type))

	allowedUsers := []attr.Value{}
	for _, user := range updated.AllowedUsers {
		allowedUsers = append(allowedUsers, types.StringValue(user.Username))
	}

	var diags diag.Diagnostics
	plan.AllowedUsers, diags = types.SetValue(types.StringType, allowedUsers)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	allowedServiceAccounts := []attr.Value{}
	for _, serviceAccount := range updated.AllowedServiceAccounts {
		allowedServiceAccounts = append(allowedServiceAccounts, types.StringValue(serviceAccount.ResourcePath))
	}

	plan.AllowedServiceAccounts, diags = types.SetValue(types.StringType, allowedServiceAccounts)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	allowedTeams := []attr.Value{}
	for _, team := range updated.AllowedTeams {
		allowedTeams = append(allowedTeams, types.StringValue(team.Name))
	}

	plan.AllowedTeams, diags = types.SetValue(types.StringType, allowedTeams)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	plan.ModuleAttestationPolicies, diags = t.toProviderAttestationPolicies(ctx, updated.ModuleAttestationPolicies)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

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
		if tharsis.IsNotFoundError(err) {
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
func (t *managedIdentityAccessRuleResource) valueStrings(ctx context.Context, arg basetypes.SetValue) ([]string, error) {
	result := make([]string, len(arg.Elements()))
	for ix, element := range arg.Elements() {
		tfValue, err := element.ToTerraformValue(ctx)
		if err != nil {
			return nil, err
		}

		var stringVal string
		if err = tfValue.As(&stringVal); err != nil {
			return nil, err
		}

		result[ix] = stringVal
	}

	return result, nil
}

// copyAttestationPoliciesToInput converts from ModuleAttestationPolicyModel to SDK equivalent.
func (t *managedIdentityAccessRuleResource) copyAttestationPoliciesToInput(ctx context.Context, list *basetypes.ListValue) ([]ttypes.ManagedIdentityAccessRuleModuleAttestationPolicy, error) {
	result := []ttypes.ManagedIdentityAccessRuleModuleAttestationPolicy{}

	for _, element := range list.Elements() {
		terraformValue, err := element.ToTerraformValue(ctx)
		if err != nil {
			return nil, err
		}

		var model ModuleAttestationPolicyModel
		if err = terraformValue.As(&model); err != nil {
			return nil, err
		}

		result = append(result, ttypes.ManagedIdentityAccessRuleModuleAttestationPolicy{
			PredicateType: model.PredicateType,
			PublicKey:     model.PublicKey,
		})
	}

	// Terraform generally wants to see nil rather than an empty list.
	if len(result) == 0 {
		result = nil
	}

	return result, nil
}

// toProviderAttestationPolicies converts from ManagedIdentityAccessRuleModuleAttestationPolicy to provider equivalent.
func (t *managedIdentityAccessRuleResource) toProviderAttestationPolicies(ctx context.Context,
	arg []ttypes.ManagedIdentityAccessRuleModuleAttestationPolicy) (basetypes.ListValue, diag.Diagnostics) {
	policies := []types.Object{}

	for _, policy := range arg {
		model := &ModuleAttestationPolicyModel{
			PredicateType: policy.PredicateType,
			PublicKey:     policy.PublicKey,
		}

		value, objectDiags := basetypes.NewObjectValueFrom(ctx, t.moduleAttestationPolicyObjectAttributes(), model)
		if objectDiags.HasError() {
			return basetypes.ListValue{}, objectDiags
		}

		policies = append(policies, value)
	}

	list, listDiags := basetypes.NewListValueFrom(ctx, basetypes.ObjectType{
		AttrTypes: t.moduleAttestationPolicyObjectAttributes(),
	}, policies)
	if listDiags.HasError() {
		return basetypes.ListValue{}, listDiags
	}

	return list, nil
}

func (t *managedIdentityAccessRuleResource) moduleAttestationPolicyObjectAttributes() map[string]attr.Type {
	return map[string]attr.Type{
		"predicate_type": types.StringType,
		"public_key":     types.StringType,
	}
}

// The End.
