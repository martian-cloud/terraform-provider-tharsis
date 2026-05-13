package provider

import (
	"context"
	"fmt"

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
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/client"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	AllowedUserIDs            basetypes.SetValue  `tfsdk:"allowed_user_ids"`
	AllowedServiceAccounts    basetypes.SetValue  `tfsdk:"allowed_service_accounts"`
	AllowedServiceAccountIDs  basetypes.SetValue  `tfsdk:"allowed_service_account_ids"`
	AllowedTeams              basetypes.SetValue  `tfsdk:"allowed_teams"`
	AllowedTeamIDs            basetypes.SetValue  `tfsdk:"allowed_team_ids"`
	VerifyStateLineage        types.Bool          `tfsdk:"verify_state_lineage"`
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
	client *client.GRPCClient
}

// Metadata returns the full name of the resource, including prefix, underscore, instance name.
func (t *managedIdentityAccessRuleResource) Metadata(_ context.Context,
	_ resource.MetadataRequest, resp *resource.MetadataResponse,
) {
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
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
				DeprecationMessage:  "Use allowed_user_ids instead. This field will be removed in a future version.",
				PlanModifiers: []planmodifier.Set{
					modifiers.SetDefault([]attr.Value{}),
				},
			},
			"allowed_user_ids": schema.SetAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "List of user IDs allowed to use the managed identity associated with this rule.",
				Description:         "List of user IDs allowed to use the managed identity associated with this rule.",
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
				DeprecationMessage:  "Use allowed_service_account_ids instead. This field will be removed in a future version.",
				PlanModifiers: []planmodifier.Set{
					modifiers.SetDefault([]attr.Value{}),
				},
			},
			"allowed_service_account_ids": schema.SetAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "List of service account IDs allowed to use the managed identity associated with this rule.",
				Description:         "List of service account IDs allowed to use the managed identity associated with this rule.",
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
				DeprecationMessage:  "Use allowed_team_ids instead. This field will be removed in a future version.",
				PlanModifiers: []planmodifier.Set{
					modifiers.SetDefault([]attr.Value{}),
				},
			},
			"allowed_team_ids": schema.SetAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "List of team IDs allowed to use the managed identity associated with this rule.",
				Description:         "List of team IDs allowed to use the managed identity associated with this rule.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Set{
					modifiers.SetDefault([]attr.Value{}),
				},
				// Can be updated in place, so no RequiresReplace plan modifier.
			},
			"verify_state_lineage": schema.BoolAttribute{
				MarkdownDescription: "Whether to verify that the workspace's current state is from the same module source, default is false.",
				Description:         "Whether to verify that the workspace's current state is from the same module source, default is false.",
				Optional:            true,
				Computed:            true, // When not passed it, it needs to be set by Create.
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
	req resource.ConfigureRequest, _ *resource.ConfigureResponse,
) {
	if req.ProviderData == nil {
		return
	}
	t.client = req.ProviderData.(*client.GRPCClient)
}

func (t *managedIdentityAccessRuleResource) Create(ctx context.Context,
	req resource.CreateRequest, resp *resource.CreateResponse,
) {
	// Retrieve values from accessRule.
	var accessRule ManagedIdentityAccessRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &accessRule)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policies, err := t.copyAttestationPoliciesToProto(ctx, &accessRule.ModuleAttestationPolicies)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error while copying module attestation policies to Tharsis input",
			err.Error(),
		)
		return
	}

	allowedUsersInput, err := t.resolveIDs(ctx, accessRule.AllowedUserIDs, accessRule.AllowedUsers, trn.TypeUser)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error while copying access rule AllowedUsers to Tharsis input",
			err.Error(),
		)
		return
	}

	allowedServiceAccountsInput, err := t.resolveIDs(ctx, accessRule.AllowedServiceAccountIDs, accessRule.AllowedServiceAccounts, trn.TypeServiceAccount)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error while copying access rule AllowedServiceAccounts to Tharsis input",
			err.Error(),
		)
		return
	}

	allowedTeamsInput, err := t.resolveIDs(ctx, accessRule.AllowedTeamIDs, accessRule.AllowedTeams, trn.TypeTeam)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error while copying access rule AllowedTeams to Tharsis input",
			err.Error(),
		)
		return
	}

	// Build the access rule input.
	input := &pb.CreateManagedIdentityAccessRuleRequest{
		ManagedIdentityId:         accessRule.ManagedIdentityID.ValueString(),
		Type:                      pb.ManagedIdentityAccessRuleType(pb.ManagedIdentityAccessRuleType_value[accessRule.Type.ValueString()]),
		RunStage:                  pb.JobType(pb.JobType_value[accessRule.RunStage.ValueString()]),
		AllowedUsers:              allowedUsersInput,
		AllowedServiceAccounts:    allowedServiceAccountsInput,
		AllowedTeams:              allowedTeamsInput,
		ModuleAttestationPolicies: policies,
	}

	// Pass in VerifyStateLineage if supplied.
	if !accessRule.VerifyStateLineage.IsNull() {
		input.VerifyStateLineage = accessRule.VerifyStateLineage.ValueBool()
	}

	// Create the managed identity access rule.
	created, err := t.client.ManagedIdentitiesClient.CreateManagedIdentityAccessRule(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating managed identity access rule",
			err.Error(),
		)
		return
	}

	// Map the response body to the schema and update the plan with the computed attribute values.
	t.copyAccessRuleToState(ctx, created, &accessRule, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set the response state to the fully-populated plan, whether or not there is an error.
	resp.Diagnostics.Append(resp.State.Set(ctx, accessRule)...)
}

func (t *managedIdentityAccessRuleResource) Read(ctx context.Context,
	req resource.ReadRequest, resp *resource.ReadResponse,
) {
	// Get the current state.
	var state ManagedIdentityAccessRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the managed identity access rule from Tharsis.
	found, err := t.client.ManagedIdentitiesClient.GetManagedIdentityAccessRuleByID(ctx,
		&pb.GetManagedIdentityAccessRuleByIDRequest{
			Id: state.ID.ValueString(),
		})
	if err != nil {

		// Handle the case that the access rule no longer exists if that fact is reported by returning an error.
		if status.Code(err) == codes.NotFound {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error reading managed identity access rule",
			err.Error(),
		)
		return
	}

	// When this Read method is called during a "terraform import" operation, state.ManagedIdentityID is null.
	// In that case, it is necessary to copy ManagedIdentityID from found to state.
	if state.ManagedIdentityID.IsNull() {
		state.ManagedIdentityID = types.StringValue(found.ManagedIdentityId)
	}

	t.copyAccessRuleToState(ctx, found, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set the refreshed state, whether or not there is an error.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (t *managedIdentityAccessRuleResource) Update(ctx context.Context,
	req resource.UpdateRequest, resp *resource.UpdateResponse,
) {
	// Retrieve values from plan for the fields to modify.
	var plan ManagedIdentityAccessRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policies, err := t.copyAttestationPoliciesToProto(ctx, &plan.ModuleAttestationPolicies)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to copy module attestation policies to Tharsis input",
			err.Error(),
		)
		return
	}

	allowedUsersInput, err := t.resolveIDs(ctx, plan.AllowedUserIDs, plan.AllowedUsers, trn.TypeUser)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error while copying access rule AllowedUsers to Tharsis input",
			err.Error(),
		)
		return
	}

	allowedServiceAccountsInput, err := t.resolveIDs(ctx, plan.AllowedServiceAccountIDs, plan.AllowedServiceAccounts, trn.TypeServiceAccount)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error while copying access rule AllowedServiceAccounts to Tharsis input",
			err.Error(),
		)
		return
	}

	allowedTeamsInput, err := t.resolveIDs(ctx, plan.AllowedTeamIDs, plan.AllowedTeams, trn.TypeTeam)
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
	toUpdate := &pb.UpdateManagedIdentityAccessRuleRequest{
		Id:                        plan.ID.ValueString(),
		AllowedUsers:              allowedUsersInput,
		AllowedServiceAccounts:    allowedServiceAccountsInput,
		AllowedTeams:              allowedTeamsInput,
		ModuleAttestationPolicies: policies,
	}

	if !plan.VerifyStateLineage.IsNull() {
		toUpdate.VerifyStateLineage = new(plan.VerifyStateLineage.ValueBool())
	}

	updated, err := t.client.ManagedIdentitiesClient.UpdateManagedIdentityAccessRule(ctx, toUpdate)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating managed identity access rule",
			err.Error(),
		)
		return
	}

	t.copyAccessRuleToState(ctx, updated, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set the response state to the fully-populated plan, error or not.
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (t *managedIdentityAccessRuleResource) Delete(ctx context.Context,
	req resource.DeleteRequest, resp *resource.DeleteResponse,
) {
	// Get the current state.
	var state ManagedIdentityAccessRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the access rule via Tharsis.
	// The ID is used to find the record to delete.
	_, err := t.client.ManagedIdentitiesClient.DeleteManagedIdentityAccessRule(ctx,
		&pb.DeleteManagedIdentityAccessRuleRequest{
			Id: state.ID.ValueString(),
		})
	if err != nil {

		// Handle the case that the access rule no longer exists.
		if status.Code(err) == codes.NotFound {
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
	req resource.ImportStateRequest, resp *resource.ImportStateResponse,
) {
	// Retrieve import ID and save to id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

	// Setting of the ManagedIdentityID field during import is handled in the Read method.
}

// copyAccessRuleToState copies the proto access rule to the Terraform state model.
func (t *managedIdentityAccessRuleResource) copyAccessRuleToState(
	ctx context.Context,
	src *pb.ManagedIdentityAccessRule,
	dest *ManagedIdentityAccessRuleModel,
	diags *diag.Diagnostics,
) {
	dest.ID = types.StringValue(src.Metadata.Id)
	dest.Type = types.StringValue(src.Type)
	dest.RunStage = types.StringValue(src.RunStage)
	dest.ManagedIdentityID = types.StringValue(src.ManagedIdentityId)
	dest.VerifyStateLineage = types.BoolValue(src.VerifyStateLineage)

	var d diag.Diagnostics

	allowedUsers := []attr.Value{}
	for _, user := range src.AllowedUsers {
		allowedUsers = append(allowedUsers, types.StringValue(user))
	}
	dest.AllowedUsers, d = types.SetValue(types.StringType, allowedUsers)
	if d.HasError() {
		diags.Append(d...)
		return
	}

	dest.AllowedUserIDs, d = types.SetValue(types.StringType, allowedUsers)
	if d.HasError() {
		diags.Append(d...)
		return
	}

	allowedServiceAccounts := []attr.Value{}
	for _, sa := range src.AllowedServiceAccounts {
		allowedServiceAccounts = append(allowedServiceAccounts, types.StringValue(sa))
	}

	dest.AllowedServiceAccounts, d = types.SetValue(types.StringType, allowedServiceAccounts)
	if d.HasError() {
		diags.Append(d...)
		return
	}

	dest.AllowedServiceAccountIDs, d = types.SetValue(types.StringType, allowedServiceAccounts)
	if d.HasError() {
		diags.Append(d...)
		return
	}

	allowedTeams := []attr.Value{}
	for _, team := range src.AllowedTeams {
		allowedTeams = append(allowedTeams, types.StringValue(team))
	}

	dest.AllowedTeams, d = types.SetValue(types.StringType, allowedTeams)
	if d.HasError() {
		diags.Append(d...)
		return
	}

	dest.AllowedTeamIDs, d = types.SetValue(types.StringType, allowedTeams)
	if d.HasError() {
		diags.Append(d...)
		return
	}

	dest.ModuleAttestationPolicies, d = t.toProviderAttestationPolicies(ctx, src.ModuleAttestationPolicies)
	if d.HasError() {
		diags.Append(d...)
	}
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

// resolveIDs prefers the ID set; if empty, normalizes the deprecated path set using the given TRN type.
// Returns an error if both are specified.
func (t *managedIdentityAccessRuleResource) resolveIDs(ctx context.Context, ids, deprecated basetypes.SetValue, trnType trn.Type) ([]string, error) {
	hasIDs := len(ids.Elements()) > 0
	hasDeprecated := len(deprecated.Elements()) > 0

	if hasIDs && hasDeprecated {
		return nil, fmt.Errorf("cannot specify both ID and deprecated path fields for the same attribute")
	}

	if hasIDs {
		return t.valueStrings(ctx, ids)
	}

	paths, err := t.valueStrings(ctx, deprecated)
	if err != nil {
		return nil, err
	}

	for i, p := range paths {
		paths[i] = trnType.Normalize(p)
	}

	return paths, nil
}

// copyAttestationPoliciesToProto converts from ModuleAttestationPolicyModel to proto equivalent.
func (t *managedIdentityAccessRuleResource) copyAttestationPoliciesToProto(ctx context.Context, list *basetypes.ListValue) ([]*pb.ManagedIdentityAccessRuleModuleAttestationPolicy, error) {
	var result []*pb.ManagedIdentityAccessRuleModuleAttestationPolicy

	for _, element := range list.Elements() {
		terraformValue, err := element.ToTerraformValue(ctx)
		if err != nil {
			return nil, err
		}

		var model ModuleAttestationPolicyModel
		if err = terraformValue.As(&model); err != nil {
			return nil, err
		}

		policy := &pb.ManagedIdentityAccessRuleModuleAttestationPolicy{
			PublicKey: model.PublicKey,
		}

		if model.PredicateType != nil {
			policy.PredicateType = model.PredicateType
		}

		result = append(result, policy)
	}

	// Terraform generally wants to see nil rather than an empty list.
	if len(result) == 0 {
		result = nil
	}

	return result, nil
}

// toProviderAttestationPolicies converts from proto to provider equivalent.
func (t *managedIdentityAccessRuleResource) toProviderAttestationPolicies(ctx context.Context,
	arg []*pb.ManagedIdentityAccessRuleModuleAttestationPolicy,
) (basetypes.ListValue, diag.Diagnostics) {
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
