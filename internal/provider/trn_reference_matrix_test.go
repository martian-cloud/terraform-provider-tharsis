package provider

import (
	"context"
	"fmt"
	"os"
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
	// withParts builds a case's config from the objects it uses. A part marked with
	// onMove is the case's alternate object; it is declared only in the step whose
	// reference names it, so the earlier steps do not read it on every command.
	withParts := func(body string, parts ...string) func(string) string {
		return func(ref string) string {
			var use []string
			for _, p := range parts {
				if alt, ok := strings.CutPrefix(p, "move:"); ok {
					if !strings.HasSuffix(ref, "/trn_"+alt+`"`) {
						continue
					}
					p = alt
				}
				use = append(use, p)
			}
			return trnRefBase(use...) + fmt.Sprintf(body, ref)
		}
	}
	onMove := func(part string) string { return "move:" + part }
	groupRef := func(body string) func(string) string { return withParts(body, onMove("other")) }
	ra := "trn:runner:" + trnRefGroupPath + "/trn_ra"
	ra2 := "trn:runner:" + trnRefGroupPath + "/trn_ra2"
	sa := "trn:service_account:" + trnRefGroupPath + "/trn_sa"
	sa2 := "trn:service_account:" + trnRefGroupPath + "/trn_sa2"
	return []trnRefCase{
		{"AssignedIdentity", "tharsis_assigned_managed_identity.trn_target", "managed_identity_id", "tharsis_managed_identity.trn_mi.id",
			mi, mi2, "tharsis_managed_identity.trn_mi", withParts(`
resource "tharsis_assigned_managed_identity" "trn_target" {
	managed_identity_id = %s
	workspace_id        = tharsis_workspace.trn_ws.id
}`, "mi", onMove("mi2"), "ws")},
		{"AssignedWorkspace", "tharsis_assigned_managed_identity.trn_target", "workspace_id", "tharsis_workspace.trn_ws.id",
			ws, ws2, "tharsis_workspace.trn_ws", withParts(`
resource "tharsis_assigned_managed_identity" "trn_target" {
	managed_identity_id = tharsis_managed_identity.trn_mi.id
	workspace_id        = %s
}`, "mi", "ws", onMove("ws2"))},
		{"RunnerAssignmentRunner", "tharsis_runner_service_account_assignment.trn_target", "runner_id", "tharsis_runner.trn_ra.id",
			ra, ra2, "tharsis_runner.trn_ra", withParts(`
resource "tharsis_runner_service_account_assignment" "trn_target" {
	runner_id    = %s
	service_account_id = tharsis_service_account.trn_sa.id
}`, "ra", onMove("ra2"), "sa")},
		{"RunnerAssignmentServiceAccount", "tharsis_runner_service_account_assignment.trn_target", "service_account_id", "tharsis_service_account.trn_sa.id",
			sa, sa2, "tharsis_service_account.trn_sa", withParts(`
resource "tharsis_runner_service_account_assignment" "trn_target" {
	runner_id    = tharsis_runner.trn_ra.id
	service_account_id = %s
}`, "ra", "sa", onMove("sa2"))},
		{"Runner", "tharsis_runner.trn_target", "group_id", "tharsis_group.root-group.id", root, other,
			"tharsis_group.root-group", groupRef(`
resource "tharsis_runner" "trn_target" {
	name        = "trn_target"
	description = "TRN reference test"
	group_id    = %s
	tags        = ["trn"]
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
			"tharsis_group.trn_alias_group", withParts(`
resource "tharsis_managed_identity_alias" "trn_target" {
	name            = "trn_target"
	group_id        = %s
	alias_source_id = tharsis_managed_identity.trn_mi.id
}`, "mi", "alias", onMove("other_alias"))},
		{"AliasSource", "tharsis_managed_identity_alias.trn_target", "alias_source_id", "tharsis_managed_identity.trn_mi.id",
			mi, mi2, "tharsis_managed_identity.trn_mi", withParts(`
resource "tharsis_managed_identity_alias" "trn_target" {
	name            = "trn_target"
	group_path      = tharsis_group.trn_alias_group.full_path
	alias_source_id = %s
}`, "mi", onMove("mi2"), "alias")},
		{"AccessRule", "tharsis_managed_identity_access_rule.trn_target", "managed_identity_id", "tharsis_managed_identity.trn_mi.id",
			mi, mi2, "tharsis_managed_identity.trn_mi", withParts(`
resource "tharsis_managed_identity_access_rule" "trn_target" {
	type                     = "eligible_principals"
	run_stage                = "plan"
	managed_identity_id      = %s
	allowed_users            = []
	allowed_service_accounts = []
	allowed_teams            = []
	allowed_user_ids            = []
	allowed_service_account_ids = []
	allowed_team_ids            = []
	verify_state_lineage     = false
	module_attestation_policies = []
}`, "mi", onMove("mi2"))},
		{"RoleBinding", "tharsis_workspace_role_binding.trn_target", "workspace_id", "tharsis_workspace.trn_ws.id",
			ws, ws2, "tharsis_workspace.trn_ws", withParts(`
resource "tharsis_workspace_role_binding" "trn_target" {
	workspace_id = %s
	role_name    = "`+testDeployerRoleName+`"
}`, "ws", onMove("ws2"))},
	}
}

// trnRefAliasGroup is the root group aliases live in; an alias must be in another namespace.
const trnRefAliasGroup = "provider-test-trn-reference-alias"

// trnRefBase declares the root group plus the named objects a case uses: "other" (a second
// group), "mi"/"mi2" (managed identities), "ws"/"ws2" (workspaces), "ra"/"ra2" (runners),
// "sa"/"sa2" (service accounts), "alias" and "other_alias" (the alias namespace and a group
// in it). Every Terraform command reads every declared object, so a case that declares
// more than it uses sends bursts of reads that approach the API's 60 requests a second.
func trnRefBase(parts ...string) string {
	var b strings.Builder
	b.WriteString(createRootGroup(trnRefGroupPath, "root group for the TRN reference tests"))
	for _, part := range parts {
		switch part {
		case "other":
			b.WriteString(`
resource "tharsis_group" "trn_other" {
	name        = "trn_other"
	description = "second group for the TRN reference tests"
	parent_path = tharsis_group.root-group.full_path
}
`)
		case "mi", "mi2":
			fmt.Fprintf(&b, `
resource "tharsis_managed_identity" "trn_%[1]s" {
	type        = "%[2]s"
	name        = "trn_%[1]s"
	description = "managed identity for the TRN reference tests"
	group_path  = tharsis_group.root-group.full_path
	aws_role    = "some-iam-role"
}
`, part, pb.ManagedIdentityType_aws_federated.String())
		case "ws", "ws2":
			fmt.Fprintf(&b, `
resource "tharsis_workspace" "trn_%[1]s" {
	name                 = "trn_%[1]s"
	description          = "workspace for the TRN reference tests"
	group_path           = tharsis_group.root-group.full_path
	prevent_destroy_plan = false
}
`, part)
		case "ra", "ra2":
			fmt.Fprintf(&b, `
resource "tharsis_runner" "trn_%[1]s" {
	name     = "trn_%[1]s"
	group_id = tharsis_group.root-group.id
	tags     = ["trn"]
}
`, part)
		case "sa", "sa2":
			fmt.Fprintf(&b, `
resource "tharsis_service_account" "trn_%[1]s" {
	name                = "trn_%[1]s"
	description         = "service account for the TRN reference tests"
	group_path          = tharsis_group.root-group.full_path
	oidc_trust_policies = [{bound_claims = {"key" = "value"}, issuer = "https://trn-test-issuer/"}]
}
`, part)
		case "alias":
			fmt.Fprintf(&b, `
resource "tharsis_group" "trn_alias_group" {
	name        = "%s"
	description = "alias group for the TRN reference tests"
}
`, trnRefAliasGroup)
		case "other_alias":
			b.WriteString(`
resource "tharsis_group" "trn_other_alias" {
	name        = "trn_other_alias"
	description = "second alias group for the TRN reference tests"
	parent_path = tharsis_group.trn_alias_group.full_path
}
`)
		default:
			panic("trnRefBase: unknown part " + part)
		}
	}
	return b.String()
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
// TRN -> Global ID -> TRN without replacement and without any write call to the API, and a
// move to a different object.
// trnRefNoUpdateRPC lists the cases whose resource has no update RPC: a Global ID <-> TRN
// switch is absorbed into state with no API call, so those steps must make zero write calls.
// Every other resource runs a normal (no-op) update on a form switch and may call the API.
var trnRefNoUpdateRPC = map[string]bool{
	"AssignedIdentity": true, "AssignedWorkspace": true, // tharsis_assigned_managed_identity
	"RunnerAssignmentRunner": true, "RunnerAssignmentServiceAccount": true, // tharsis_runner_service_account_assignment
	"AliasGroup": true, "AliasSource": true, // tharsis_managed_identity_alias
	"GPGKey": true, // tharsis_gpg_key
}

func TestTRNReferenceMatrix(t *testing.T) {
	for _, c := range trnRefCases() {
		t.Run(c.name, func(t *testing.T) {
			var id string
			calls := newWriteCallLog(t)
			noWrite := trnRefNoUpdateRPC[c.name]
			formSwitchChecks := func(wantAttr resource.TestCheckFunc) resource.TestCheckFunc {
				checks := []resource.TestCheckFunc{testCheckAttrUnchanged(c.resource, "id", &id), wantAttr}
				// A resource with no update RPC must make no write call on a form switch; one with
				// an update RPC runs a normal no-op update, so writes are not asserted there.
				if noWrite {
					checks = append(checks, calls.checkNoneSinceMark())
				}
				return resource.ComposeAggregateTestCheckFunc(checks...)
			}
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: c.withDependsOn(quoted(c.trn)),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr(c.resource, c.attr, c.trn),
							testCaptureAttr(c.resource, "id", &id),
							calls.mark(),
						),
					},
					{
						Config: c.withDependsOn(c.gid),
						Check:  formSwitchChecks(resource.TestCheckResourceAttrPair(c.resource, c.attr, c.gidAddr, "id")),
					},
					{
						Config: c.withDependsOn(quoted(c.trn)),
						Check:  formSwitchChecks(resource.TestCheckResourceAttr(c.resource, c.attr, c.trn)),
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

// writeCallLog counts the provider's write calls to the Tharsis API. The gRPC client logs
// every request at debug level; the test sets Terraform's log for the whole test case to a
// file of its own and reads the "gRPC request" lines back.
type writeCallLog struct {
	path   string
	offset int64
}

// writeMethod matches the gRPC methods that change something in Tharsis.
var writeMethod = regexp.MustCompile(`gRPC request: .*method=((?:Create|Update|Delete|Set|Assign|Remove|Unassign)[A-Za-z]+)`)

func newWriteCallLog(t *testing.T) *writeCallLog {
	t.Helper()
	p := t.TempDir() + "/terraform.log"
	t.Setenv("TF_ACC_LOG_PATH", p)
	t.Setenv("TF_LOG_PROVIDER", "DEBUG")
	return &writeCallLog{path: p}
}

// mark remembers how far the log has been read, so the next check sees only later steps.
func (l *writeCallLog) mark() resource.TestCheckFunc {
	return func(*terraform.State) error {
		st, err := os.Stat(l.path)
		if err != nil {
			return fmt.Errorf("reading the Terraform log: %w", err)
		}
		l.offset = st.Size()
		return nil
	}
}

// checkNoneSinceMark fails if any write call was made since the last mark, then marks again.
func (l *writeCallLog) checkNoneSinceMark() resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		b, err := os.ReadFile(l.path)
		if err != nil {
			return fmt.Errorf("reading the Terraform log: %w", err)
		}
		var writes []string
		for _, m := range writeMethod.FindAllSubmatch(b[l.offset:], -1) {
			writes = append(writes, string(m[1]))
		}
		l.offset = int64(len(b))
		if len(writes) > 0 {
			return fmt.Errorf("a form-only change made API write calls: %v", writes)
		}
		return nil
	}
}

// TestTRNReferenceFormSwitchWithRemoval switches group_id between forms in the same apply
// that removes the description. Terraform plans an unset Optional+Computed input as
// unknown, which looks the same as a value the user is not touching, so the provider must
// run the normal update rather than skip it: the description must be cleared. A later form
// switch with the description still unset also runs the normal update and changes nothing.
func TestTRNReferenceFormSwitchWithRemoval(t *testing.T) {
	var id string
	cfg := func(groupRef, descLine string) string {
		return trnRefBase() + fmt.Sprintf(`
resource "tharsis_runner" "trn_target" {
	name     = "trn_target"
	group_id = %s
	tags     = ["trn"]
	%s
	depends_on = [tharsis_group.root-group]
}`, groupRef, descLine)
	}
	trnRef := quoted("trn:group:" + trnRefGroupPath)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg("tharsis_group.root-group.id", `description = "to be removed"`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCaptureAttr("tharsis_runner.trn_target", "id", &id),
					resource.TestCheckResourceAttr("tharsis_runner.trn_target", "description", "to be removed"),
				),
			},
			{
				Config: cfg(trnRef, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckAttrUnchanged("tharsis_runner.trn_target", "id", &id),
					resource.TestCheckResourceAttr("tharsis_runner.trn_target", "group_id", "trn:group:"+trnRefGroupPath),
					resource.TestCheckResourceAttr("tharsis_runner.trn_target", "description", ""),
				),
			},
			{
				Config: cfg("tharsis_group.root-group.id", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckAttrUnchanged("tharsis_runner.trn_target", "id", &id),
					resource.TestCheckResourceAttrPair("tharsis_runner.trn_target", "group_id",
						"tharsis_group.root-group", "id"),
					resource.TestCheckResourceAttr("tharsis_runner.trn_target", "description", ""),
				),
			},
		},
	})
}

// trnRefImportIDs gives the import ID form for each matrix case; cases not listed have no
// import (the VCS provider link) or import a different object than they create.
var trnRefImportIDs = map[string]func(map[string]string) string{
	"AssignedIdentity":               func(a map[string]string) string { return a["workspace_id"] + "/" + a["managed_identity_id"] },
	"AssignedWorkspace":              func(a map[string]string) string { return a["workspace_id"] + "/" + a["managed_identity_id"] },
	"RoleBinding":                    func(a map[string]string) string { return a["workspace_id"] },
	"RunnerAssignmentRunner":         func(a map[string]string) string { return a["runner_id"] + "/" + a["service_account_id"] },
	"RunnerAssignmentServiceAccount": func(a map[string]string) string { return a["runner_id"] + "/" + a["service_account_id"] },
}

func trnRefImportID(c trnRefCase) func(map[string]string) string {
	if f, ok := trnRefImportIDs[c.name]; ok {
		return f
	}
	switch c.name {
	case "AliasGroup":
		return nil // one config would hold two aliases of one name
	}
	return func(a map[string]string) string { return a["id"] }
}

// TestTRNReferenceAfterImport covers the import case: an object imported by ID has its
// reference stored as a Global ID. A configuration that then uses the TRN of the same
// object must be an in-place update with no replacement and no write call to the API.
//
// The object is created by Terraform under one address and imported into a second one,
// which keeps the test on the import blocks of Terraform 1.0. Only the imported copy then
// switches to the TRN. Teardown deletes the object twice; every Delete ignores NotFound.
func TestTRNReferenceAfterImport(t *testing.T) {
	for _, c := range trnRefCases() {
		importID := trnRefImportID(c)
		if importID == nil {
			continue
		}
		t.Run(c.name, func(t *testing.T) {
			var id, impID, impStateID string
			calls := newWriteCallLog(t)
			noWrite := trnRefNoUpdateRPC[c.name]
			kind, name := splitAddr(c.resource)
			imported := kind + ".trn_import"
			// twice declares the resource under its own address and again as trn_import, the
			// latter with the reference set to impRef.
			twice := func(ref, impRef string) string {
				own := c.withDependsOn(ref)
				block := c.withDependsOn(impRef)
				block = block[strings.Index(block, `resource "`+kind+`" "`+name+`"`):]
				return own + "\n" + strings.Replace(block, `"`+name+`"`, `"trn_import"`, 1)
			}
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: c.withDependsOn(c.gid),
						Check: resource.ComposeAggregateTestCheckFunc(
							testCaptureAttr(c.resource, "id", &id),
							func(s *terraform.State) error {
								impID = importID(s.RootModule().Resources[c.resource].Primary.Attributes)
								return nil
							},
						),
					},
					{
						Config:             twice(c.gid, c.gid),
						ResourceName:       imported,
						ImportState:        true,
						ImportStatePersist: true,
						ImportStateIdFunc:  func(*terraform.State) (string, error) { return impID, nil },
					},
					{
						Config: twice(c.gid, c.gid),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttrPair(imported, c.attr, c.gidAddr, "id"),
							testCaptureAttr(imported, "id", &impStateID),
							calls.mark(),
						),
					},
					{
						Config: twice(c.gid, quoted(c.trn)),
						Check: func() resource.TestCheckFunc {
							// Unchanged id = no replacement. (The assignment resources mint a fresh id on
							// import, so the comparison is with the imported copy, not the original.)
							checks := []resource.TestCheckFunc{
								testCheckAttrUnchanged(imported, "id", &impStateID),
								resource.TestCheckResourceAttr(imported, c.attr, c.trn),
							}
							// Only resources with no update RPC absorb a form switch without an API call.
							if noWrite {
								checks = append(checks, calls.checkNoneSinceMark())
							}
							return resource.ComposeAggregateTestCheckFunc(checks...)
						}(),
					},
				},
			})
		})
	}
}

func splitAddr(addr string) (kind, name string) {
	i := strings.Index(addr, ".")
	return addr[:i], addr[i+1:]
}
