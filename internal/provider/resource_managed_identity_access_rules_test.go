package provider

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	ttypes "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

// TestManagedIdentityAccessRules tests creation, reading, updating, and deletion
// of managed identity access rule resources.
func TestManagedIdentityAccessRules(t *testing.T) {

	// Configuration for the parent managed identity.
	parentName := "tmiar_parent_name"

	// TODO: When we have the ability to create the parent group, users, service accounts, and teams, add them.

	// Configuration to create the access rule(s).
	ruleStage := "plan"

	// Configuration to update the access rule(s).
	// Only the run stage can be changed.
	updateStage := "apply"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{

			// Create the parent managed identity and the access rule in one step.
			// If done in separate steps, the access rule can't find its parent.
			{
				Config: testSharedProviderConfiguration() +
					testManagedIdentityAccessRulesConfigurationParent() +
					testManagedIdentityAccessRulesConfigurationRule(),
				Check: resource.ComposeAggregateTestCheckFunc(

					// Verify a few key values of the parent that should be known.
					resource.TestCheckResourceAttr("tharsis_managed_identity.tmiar_parent", "name", parentName),
					resource.TestCheckResourceAttr("tharsis_managed_identity.tmiar_parent", "group_path", testGroupPath),
					resource.TestCheckResourceAttrSet("tharsis_managed_identity.tmiar_parent", "id"),

					// Verify access rule values that should be known.
					resource.TestCheckResourceAttr("tharsis_managed_identity_access_rule.rule01",
						"run_stage", ruleStage),
					resource.TestCheckResourceAttrPair("tharsis_managed_identity.tmiar_parent", "id",
						"tharsis_managed_identity_access_rule.rule01", "managed_identity_id"),

					// Verify dynamic values have some value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_managed_identity_access_rule.rule01", "id"),
				),
			},

			// Import state.
			{
				ResourceName:      "tharsis_managed_identity_access_rule.rule01",
				ImportState:       true,
				ImportStateVerify: true,
			},

			// Attempt to update and read.
			// This is some indication this might be doing a fresh creation rather than an update.
			{
				Config: testSharedProviderConfiguration() +
					testManagedIdentityAccessRulesConfigurationParent() +
					testManagedIdentityAccessRulesConfigurationUpdate(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_managed_identity_access_rule.rule01",
						"run_stage", updateStage),
					resource.TestCheckResourceAttrPair("tharsis_managed_identity.tmiar_parent", "id",
						"tharsis_managed_identity_access_rule.rule01", "managed_identity_id"),

					// Verify dynamic values have some value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_managed_identity_access_rule.rule01", "id"),
				),
			},

			// Destroy should be covered automatically by TestCase.
		},
	})
}

func testManagedIdentityAccessRulesConfigurationParent() string {
	parentType := string(ttypes.ManagedIdentityAWSFederated)
	parentName := "tmiar_parent_name"
	parentDescription := "this is tmiar_parent, a Tharsis managed identity"
	parentAWSRole := "some-iam-aws-role"
	return fmt.Sprintf(`

%s

resource "tharsis_managed_identity" "tmiar_parent" {
	type        = "%s"
	name        = "%s"
	description = "%s"
	group_path  = tharsis_group.root-group.full_path
	aws_role    = "%s"
}

	`, createRootGroup(), parentType, parentName, parentDescription, parentAWSRole)
}

func testManagedIdentityAccessRulesConfigurationRule() string {
	ruleStage := "plan"
	ruleParentID := "tharsis_managed_identity.tmiar_parent.id"
	return fmt.Sprintf(`

resource "tharsis_managed_identity_access_rule" "rule01" {
	run_stage                = "%s"
	managed_identity_id      = %s
	allowed_users            = []
	allowed_service_accounts = []
	allowed_teams            = []
}

`, ruleStage, ruleParentID)
}

func testManagedIdentityAccessRulesConfigurationUpdate() string {
	// Only the run stage can be changed.
	ruleStage := "plan"
	updateStage := "apply"
	return strings.Replace(testManagedIdentityAccessRulesConfigurationRule(), ruleStage, updateStage, 1)
}

// The End.
