package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
)

// Reference attributes accept either a Global ID or a TRN. For each one these tests check
// that a TRN works on create; that switching between the two forms of the same object
// neither replaces the resource nor leaves a diff, and that state keeps the configured
// form; and that moving the reference to a different object still replaces the resource.
// The Global ID step also stands in for an import, which stores the reference as a Global
// ID: the next step, back to the TRN, is the update a user sees after importing.
//
// The resource framework runs a follow-up plan after every step and fails on a non-empty
// one, so every step also proves the configuration has settled.

// trnRefGroupPath is the root group these tests create, kept apart from testGroupPath.
const trnRefGroupPath = "provider-test-trn-reference"

// trnRefCase describes one reference attribute under test.
type trnRefCase struct {
	name     string // test name suffix
	resource string // resource address, e.g. tharsis_workspace.trn_target
	attr     string // reference attribute
	// gid and trn are the attribute's value in configuration: an expression for the Global
	// ID form, and the TRN form as a string. other is a TRN of a different object.
	gid, trn, other string
	// gidAddr names the resource whose id equals the reference in its Global ID form.
	gidAddr string
	// config returns the full configuration with the reference set to ref.
	config func(ref string) string
}

func trnRefCases() []trnRefCase {
	root := "trn:group:" + trnRefGroupPath
	other := root + "/trn_other"
	mi := "trn:managed_identity:" + trnRefGroupPath + "/trn_mi"
	mi2 := "trn:managed_identity:" + trnRefGroupPath + "/trn_mi2"
	ws := "trn:workspace:" + trnRefGroupPath + "/trn_ws"
	ws2 := "trn:workspace:" + trnRefGroupPath + "/trn_ws2"
	withParts := func(body string, parts ...string) func(string) string {
		return func(ref string) string { return trnRefBase(parts...) + fmt.Sprintf(body, ref) }
	}
	groupRef := func(body string) func(string) string { return withParts(body) }
	ra := "trn:runner:" + trnRefGroupPath + "/trn_ra"
	ra2 := "trn:runner:" + trnRefGroupPath + "/trn_ra2"
	sa := "trn:service_account:" + trnRefGroupPath + "/trn_sa"
	sa2 := "trn:service_account:" + trnRefGroupPath + "/trn_sa2"
	assignment := func(body string) func(string) string { return withParts(body, "runner") }
	miRef := func(body string) func(string) string { return withParts(body, "mi") }
	wsRef := func(body string) func(string) string { return withParts(body, "ws") }
	miWSRef := func(body string) func(string) string { return withParts(body, "mi", "ws") }
	return []trnRefCase{
		{"AssignedIdentity", "tharsis_assigned_managed_identity.trn_target", "managed_identity_id", "tharsis_managed_identity.trn_mi.id",
			mi, mi2, "tharsis_managed_identity.trn_mi", miWSRef(`
resource "tharsis_assigned_managed_identity" "trn_target" {
	managed_identity_id = %s
	workspace_id        = tharsis_workspace.trn_ws.id
}`)},
		{"AssignedWorkspace", "tharsis_assigned_managed_identity.trn_target", "workspace_id", "tharsis_workspace.trn_ws.id",
			ws, ws2, "tharsis_workspace.trn_ws", miWSRef(`
resource "tharsis_assigned_managed_identity" "trn_target" {
	managed_identity_id = tharsis_managed_identity.trn_mi.id
	workspace_id        = %s
}`)},
		{"RunnerAssignmentRunner", "tharsis_runner_agent_service_account.trn_target", "runner_agent_id", "tharsis_runner_agent.trn_ra.id",
			ra, ra2, "tharsis_runner_agent.trn_ra", assignment(`
resource "tharsis_runner_agent_service_account" "trn_target" {
	runner_agent_id    = %s
	service_account_id = tharsis_service_account.trn_sa.id
}`)},
		{"RunnerAssignmentServiceAccount", "tharsis_runner_agent_service_account.trn_target", "service_account_id", "tharsis_service_account.trn_sa.id",
			sa, sa2, "tharsis_service_account.trn_sa", assignment(`
resource "tharsis_runner_agent_service_account" "trn_target" {
	runner_agent_id    = tharsis_runner_agent.trn_ra.id
	service_account_id = %s
}`)},
		{"RunnerAgent", "tharsis_runner_agent.trn_target", "group_id", "tharsis_group.root-group.id", root, other,
			"tharsis_group.root-group", groupRef(`
resource "tharsis_runner_agent" "trn_target" {
	name     = "trn_target"
	group_id = %s
	tags     = ["trn"]
}`)},
		{"ServiceAccount", "tharsis_service_account.trn_target", "group_id", "tharsis_group.root-group.id", root, other,
			"tharsis_group.root-group", groupRef(`
resource "tharsis_service_account" "trn_target" {
	name                = "trn_target"
	description         = "TRN reference test"
	group_id            = %s
	oidc_trust_policies = [{bound_claims = {"key" = "value"}, issuer = "https://trn-test-issuer/"}]
}`)},
		{"Workspace", "tharsis_workspace.trn_target", "group_id", "tharsis_group.root-group.id", root, other,
			"tharsis_group.root-group", groupRef(`
resource "tharsis_workspace" "trn_target" {
	name                 = "trn_target"
	description          = "TRN reference test"
	group_id             = %s
	prevent_destroy_plan = false
}`)},
		{"NestedGroup", "tharsis_group.trn_target", "parent_id", "tharsis_group.root-group.id", root, other,
			"tharsis_group.root-group", groupRef(`
resource "tharsis_group" "trn_target" {
	name        = "trn_target"
	description = "TRN reference test"
	parent_id   = %s
}`)},
		{"ManagedIdentity", "tharsis_managed_identity.trn_target", "group_id", "tharsis_group.root-group.id", root, other,
			"tharsis_group.root-group", groupRef(`
resource "tharsis_managed_identity" "trn_target" {
	type        = "` + pb.ManagedIdentityType_aws_federated.String() + `"
	name        = "trn_target"
	description = "TRN reference test"
	group_id    = %s
	aws_role    = "some-iam-role"
}`)},
		{"TerraformModule", "tharsis_terraform_module.trn_target", "group_id", "tharsis_group.root-group.id", root, other,
			"tharsis_group.root-group", groupRef(`
resource "tharsis_terraform_module" "trn_target" {
	name     = "trn-target"
	system   = "aws"
	group_id = %s
	repository_url = "https://example.com/trn-target"
	private        = true
}`)},
		{"TerraformProvider", "tharsis_terraform_provider.trn_target", "group_id", "tharsis_group.root-group.id", root, other,
			"tharsis_group.root-group", groupRef(`
resource "tharsis_terraform_provider" "trn_target" {
	name     = "trn-target"
	group_id = %s
	repository_url = "https://example.com/trn-target"
	private        = true
}`)},
		{"VCSProvider", "tharsis_vcs_provider.trn_target", "group_id", "tharsis_group.root-group.id", root, other,
			"tharsis_group.root-group", groupRef(`
resource "tharsis_vcs_provider" "trn_target" {
	name                 = "trn_target"
	description          = "TRN reference test"
	url                  = "https://vcs-provider.domain"
	group_id             = %s
	type                 = "gitlab"
	auto_create_webhooks = false
	oauth_client_id      = "trn-oauth-client-id"
	oauth_client_secret  = "trn-oauth-client-secret" # nosec G101 -- test value
}`)},
		{"GPGKey", "tharsis_gpg_key.trn_target", "group_id", "tharsis_group.root-group.id", root, other,
			"tharsis_group.root-group", groupRef(`
resource "tharsis_gpg_key" "trn_target" {
	ascii_armor = ` + fmt.Sprintf("%#v", testGPGKeyASCIIArmor) + `
	group_id    = %s
}`)},
		{"AliasGroup", "tharsis_managed_identity_alias.trn_target", "group_id", "tharsis_group.trn_alias_group.id",
			"trn:group:" + trnRefAliasGroup, "trn:group:" + trnRefAliasGroup + "/trn_other_alias",
			"tharsis_group.trn_alias_group", func(ref string) string {
				return trnRefBase("mi", "alias") + fmt.Sprintf(`
resource "tharsis_managed_identity_alias" "trn_target" {
	name            = "trn_target"
	group_id        = %s
	alias_source_id = tharsis_managed_identity.trn_mi.id
}`, ref)
			}},
		{"AliasSource", "tharsis_managed_identity_alias.trn_target", "alias_source_id", "tharsis_managed_identity.trn_mi.id",
			mi, mi2, "tharsis_managed_identity.trn_mi", func(ref string) string {
				return trnRefBase("mi", "alias") + fmt.Sprintf(`
resource "tharsis_managed_identity_alias" "trn_target" {
	name            = "trn_target"
	group_path      = tharsis_group.trn_alias_group.full_path
	alias_source_id = %s
}`, ref)
			}},
		{"AccessRule", "tharsis_managed_identity_access_rule.trn_target", "managed_identity_id", "tharsis_managed_identity.trn_mi.id",
			mi, mi2, "tharsis_managed_identity.trn_mi", miRef(`
resource "tharsis_managed_identity_access_rule" "trn_target" {
	type                     = "eligible_principals"
	run_stage                = "plan"
	managed_identity_id      = %s
	allowed_users            = []
	allowed_service_accounts = []
	allowed_teams            = []
}`)},
		{"RoleBinding", "tharsis_workspace_role_binding.trn_target", "workspace_id", "tharsis_workspace.trn_ws.id",
			ws, ws2, "tharsis_workspace.trn_ws", wsRef(`
resource "tharsis_workspace_role_binding" "trn_target" {
	workspace_id = %s
	role_name    = "` + testDeployerRoleName + `"
}`)},
	}
}

// trnRefAliasGroup is the root group aliases live in; an alias must be in another namespace.
const trnRefAliasGroup = "provider-test-trn-reference-alias"

// trnRefBase declares the root group and a second group, plus the named extra parts a
// case references: "mi" (two managed identities), "ws" (two workspaces), "runner" (two
// runner agents and two service accounts) and "alias" (the alias namespace). Each case
// declares only what it uses, to keep refreshes under the API's rate limit.
func trnRefBase(parts ...string) string {
	var b strings.Builder
	b.WriteString(createRootGroup(trnRefGroupPath, "root group for the TRN reference tests"))
	b.WriteString(`
resource "tharsis_group" "trn_other" {
	name        = "trn_other"
	description = "second group for the TRN reference tests"
	parent_path = tharsis_group.root-group.full_path
}
`)
	for _, part := range parts {
		switch part {
		case "mi":
			for _, n := range []string{"trn_mi", "trn_mi2"} {
				fmt.Fprintf(&b, `
resource "tharsis_managed_identity" "%[1]s" {
	type        = "%[2]s"
	name        = "%[1]s"
	description = "managed identity for the TRN reference tests"
	group_path  = tharsis_group.root-group.full_path
	aws_role    = "some-iam-role"
}
`, n, pb.ManagedIdentityType_aws_federated.String())
			}
		case "ws":
			for _, n := range []string{"trn_ws", "trn_ws2"} {
				fmt.Fprintf(&b, `
resource "tharsis_workspace" "%[1]s" {
	name                 = "%[1]s"
	description          = "workspace for the TRN reference tests"
	group_path           = tharsis_group.root-group.full_path
	prevent_destroy_plan = false
}
`, n)
			}
		case "runner":
			b.WriteString(trnRefRunnerBase())
		case "alias":
			b.WriteString(trnRefAliasBase())
		}
	}
	return b.String()
}

// trnRefRunnerBase declares two runner agents and two service accounts to assign.
func trnRefRunnerBase() string {
	var b strings.Builder
	for _, n := range []string{"trn_ra", "trn_ra2"} {
		fmt.Fprintf(&b, `
resource "tharsis_runner_agent" "%[1]s" {
	name     = "%[1]s"
	group_id = tharsis_group.root-group.id
	tags     = ["trn"]
}
`, n)
	}
	for _, n := range []string{"trn_sa", "trn_sa2"} {
		fmt.Fprintf(&b, `
resource "tharsis_service_account" "%[1]s" {
	name                = "%[1]s"
	description         = "service account for the TRN reference tests"
	group_path          = tharsis_group.root-group.full_path
	oidc_trust_policies = [{bound_claims = {"key" = "value"}, issuer = "https://trn-test-issuer/"}]
}
`, n)
	}
	return b.String()
}

// trnRefAliasBase declares the alias namespace and a second group inside it.
func trnRefAliasBase() string {
	return fmt.Sprintf(`
resource "tharsis_group" "trn_alias_group" {
	name        = "%s"
	description = "alias group for the TRN reference tests"
}

resource "tharsis_group" "trn_other_alias" {
	name        = "trn_other_alias"
	description = "second alias group for the TRN reference tests"
	parent_path = tharsis_group.trn_alias_group.full_path
}
`, trnRefAliasGroup)
}

// withDependsOn adds depends_on on every other declared resource to the case's resource,
// so a literal TRN reference still orders creation and destruction correctly.
func (c trnRefCase) withDependsOn(ref string) string {
	cfg := c.config(ref)
	var deps []string
	for _, m := range trnRefDeclared.FindAllStringSubmatch(cfg, -1) {
		if addr := m[1] + "." + m[2]; addr != c.resource {
			deps = append(deps, addr)
		}
	}
	i := strings.LastIndex(cfg, "}")
	return cfg[:i] + "\tdepends_on = [" + strings.Join(deps, ", ") + "]\n}" + cfg[i+1:]
}

// trnRefDeclared matches resource headers written on one line, as every config here is.
var trnRefDeclared = regexp.MustCompile(`resource "(tharsis_[a-z_]+)" "([a-z0-9_-]+)"`)

func quoted(s string) string { return fmt.Sprintf("%q", s) }

// TestTRNReferenceMatrix runs every reference attribute through create-with-TRN,
// TRN -> Global ID -> TRN without replacement, and a move to a different object.
func TestTRNReferenceMatrix(t *testing.T) {
	for _, c := range trnRefCases() {
		t.Run(c.name, func(t *testing.T) {
			var id string
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: c.withDependsOn(quoted(c.trn)),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr(c.resource, c.attr, c.trn),
							testCaptureAttr(c.resource, "id", &id),
						),
					},
					{
						Config: c.withDependsOn(c.gid),
						Check: resource.ComposeAggregateTestCheckFunc(
							testCheckAttrUnchanged(c.resource, "id", &id),
							resource.TestCheckResourceAttrPair(c.resource, c.attr, c.gidAddr, "id"),
						),
					},
					{
						Config: c.withDependsOn(quoted(c.trn)),
						Check: resource.ComposeAggregateTestCheckFunc(
							testCheckAttrUnchanged(c.resource, "id", &id),
							resource.TestCheckResourceAttr(c.resource, c.attr, c.trn),
						),
					},
					{
						Config: c.withDependsOn(quoted(c.other)),
						Check: resource.ComposeAggregateTestCheckFunc(
							testCheckAttrChanged(c.resource, "id", &id),
							resource.TestCheckResourceAttr(c.resource, c.attr, c.other),
						),
					},
				},
			})
		})
	}
}

// TestTRNReferenceGroupGainsParentReplaces gives a root group a parent. parent_id goes from
// null to a value; the API cannot move a group, so the group must be replaced.
func TestTRNReferenceGroupGainsParentReplaces(t *testing.T) {
	var groupID string
	cfg := func(parentLine string) string {
		return createRootGroup(trnRefGroupPath, "root group for the TRN reference tests") + fmt.Sprintf(`
resource "tharsis_group" "trn_child" {
	name        = "provider-test-trn-reference-child"
	description = "group whose parent the TRN reference tests change"
	%s
}`, parentLine)
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: cfg(""), Check: testCaptureAttr("tharsis_group.trn_child", "id", &groupID)},
			{
				Config: cfg("parent_id = tharsis_group.root-group.id"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckAttrChanged("tharsis_group.trn_child", "id", &groupID),
					resource.TestCheckResourceAttrPair("tharsis_group.trn_child", "parent_id",
						"tharsis_group.root-group", "id"),
				),
			},
		},
	})
}

// testCaptureAttr records an attribute's value from state into dst.
func testCaptureAttr(name, key string, dst *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("resource not found: %s", name)
		}
		*dst = rs.Primary.Attributes[key]
		return nil
	}
}

// testCheckAttrUnchanged fails if the attribute no longer equals the captured value,
// which for an id means the resource was replaced.
func testCheckAttrUnchanged(name, key string, want *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("resource not found: %s", name)
		}
		if got := rs.Primary.Attributes[key]; got != *want {
			return fmt.Errorf("%s.%s changed from %q to %q (resource was replaced)", name, key, *want, got)
		}
		return nil
	}
}

// testCheckAttrChanged fails if the attribute still equals the captured value.
func testCheckAttrChanged(name, key string, previous *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("resource not found: %s", name)
		}
		if got := rs.Primary.Attributes[key]; got == *previous {
			return fmt.Errorf("%s.%s is still %q; expected a replacement", name, key, got)
		}
		return nil
	}
}

// TestTRNReferenceAssignedIdentityGone assigns a managed identity by TRN, then deletes the
// managed identity outside Terraform. The TRN no longer resolves, so Read must drop the
// assignment from state rather than fail, and the next plan re-creates it.
func TestTRNReferenceAssignedIdentityGone(t *testing.T) {
	miTRN := "trn:managed_identity:" + trnRefGroupPath + "/trn_mi"
	cfg := trnRefBase("mi", "ws") + `
resource "tharsis_assigned_managed_identity" "trn_target" {
	managed_identity_id = "` + miTRN + `"
	workspace_id        = tharsis_workspace.trn_ws.id
	depends_on          = [tharsis_managed_identity.trn_mi]
}`
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: cfg},
			{
				// Delete the managed identity behind Terraform's back, then refresh and plan.
				PreConfig: func() {
					ctx := context.Background()
					c := testTharsisClient(ctx, t)
					force := true
					if _, err := c.ManagedIdentitiesClient.DeleteManagedIdentity(ctx,
						&pb.DeleteManagedIdentityRequest{Id: miTRN, Force: &force}); err != nil {
						t.Fatalf("deleting the managed identity: %v", err)
					}
				},
				Config:             cfg,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
