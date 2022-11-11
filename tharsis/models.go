package tharsis

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// WorkspacesOutputsDataSourceData represents the outputs for a workspace in Tharsis.
type WorkspacesOutputsDataSourceData struct {
	Path           types.String      `tfsdk:"path"`
	FullPath       types.String      `tfsdk:"full_path"`
	WorkspaceID    types.String      `tfsdk:"workspace_id"`
	StateVersionID types.String      `tfsdk:"state_version_id"`
	Outputs        map[string]string `tfsdk:"outputs"`
}

// ManagedIdentityModel is the model for a managed identity.
type ManagedIdentityModel struct {
	ID           types.String                     `tfsdk:"id"`
	Type         types.String                     `tfsdk:"type"`
	ResourcePath types.String                     `tfsdk:"resource_path"`
	Name         types.String                     `tfsdk:"name"`
	Description  types.String                     `tfsdk:"description"`
	GroupPath    types.String                     `tfsdk:"group_path"`
	CreatedBy    types.String                     `tfsdk:"created_by"`
	Role         types.String                     `tfsdk:"role"`
	ClientID     types.String                     `tfsdk:"client_id"`
	TenantID     types.String                     `tfsdk:"tenant_id"`
	Subject      types.String                     `tfsdk:"subject"`
	AccessRules  []ManagedIdentityAccessRuleModel `tfsdk:"access_rules"`
	LastUpdated  types.String                     `tfsdk:"last_updated"`
}

// ManagedIdentityAccessRuleModel is the model for a managed identity access rule.
type ManagedIdentityAccessRuleModel struct {
	ID                     types.String   `tfsdk:"id"`
	RunStage               types.String   `tfsdk:"run_stage"`
	ManagedIdentityID      types.String   `tfsdk:"managed_identity_id"`
	AllowedUsers           []types.String `tfsdk:"allowed_users"`
	AllowedServiceAccounts []types.String `tfsdk:"allowed_service_accounts"`
	AllowedTeams           []types.String `tfsdk:"allowed_teams"`
}
