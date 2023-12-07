package provider

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	ttypes "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

const (
	dummyPublicKey = `-----BEGIN PUBLIC KEY-----\nMIIBITANBgkqhkiG9w0BAQEFAAOCAQ4AMIIBCQKCAQBbtSCD0EYwujE7O/VfB5e0\nLSeHAP1dYAgOjjRPdu3K4FT0ugJkUhjCqpdqnrFQGmeBOLW2BQbvoVfuiC+VaPqW\nIuyb0DfE2PAtxBZjc7kZkxIxVcITk2bUWiXQH/+Es0Qn85o3rdBC8tBb2wUE3rQ8\nNU3Qbmnl5epnqyGjuBpD9DCJofaK0juPMbB16m1z7GXPPBc8vxg4r/CWrff5yAEu\n3Nwq9NaoL9DKlv2GTUtgm4+3oHPUq45kSD+DSLdzoLEsTHeoQblWEiZ4eBCHDmdq\ne6nxeRj2n+n0YT7mIkZVdvlrrtSfZTYyLjHFTTBgUnv9j2Tof46VDIDgVGYWpGIl\nAgMBAAE=\n-----END PUBLIC KEY-----`
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

	ruleType := "eligible_principals"

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
					resource.TestCheckResourceAttr("tharsis_managed_identity_access_rule.rule01", "type", ruleType),

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
					resource.TestCheckResourceAttr("tharsis_managed_identity_access_rule.rule01",
						"type", ruleType),

					// Verify dynamic values have some value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_managed_identity_access_rule.rule01", "id"),
				),
			},

			{
				Config: testSharedProviderConfiguration() +
					testManagedIdentityAccessRulesConfigurationParent() +
					testManagedIdentityAccessRulesConfigurationRule2(),
				Check: resource.ComposeAggregateTestCheckFunc(

					// Verify a few key values of the parent that should be known.
					resource.TestCheckResourceAttr("tharsis_managed_identity.tmiar_parent", "name", parentName),
					resource.TestCheckResourceAttr("tharsis_managed_identity.tmiar_parent", "group_path", testGroupPath),
					resource.TestCheckResourceAttrSet("tharsis_managed_identity.tmiar_parent", "id"),

					// Verify access rule values that should be known.
					resource.TestCheckResourceAttr("tharsis_managed_identity_access_rule.rule02",
						"run_stage", ruleStage),
					resource.TestCheckResourceAttrPair("tharsis_managed_identity.tmiar_parent", "id",
						"tharsis_managed_identity_access_rule.rule02", "managed_identity_id"),

					// Verify dynamic values have some value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_managed_identity_access_rule.rule02", "id"),
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

	`, createRootGroup(testGroupPath, "this is a test root group"), parentType, parentName, parentDescription, parentAWSRole)
}

func testManagedIdentityAccessRulesConfigurationRule() string {
	ruleStage := "plan"
	ruleParentID := "tharsis_managed_identity.tmiar_parent.id"
	ruleType := "eligible_principals"
	return fmt.Sprintf(`

resource "tharsis_managed_identity_access_rule" "rule01" {
	type 					 = "%s"
	run_stage                = "%s"
	managed_identity_id      = %s
	allowed_users            = []
	allowed_service_accounts = []
	allowed_teams            = []
}

`, ruleType, ruleStage, ruleParentID)
}

func testManagedIdentityAccessRulesConfigurationRule2() string {
	ruleStage := "plan"
	ruleParentID := "tharsis_managed_identity.tmiar_parent.id"
	ruleType := "module_attestation"
	return fmt.Sprintf(`

resource "tharsis_managed_identity_access_rule" "rule02" {
	type 					    = "%s"
	run_stage                   = "%s"
	managed_identity_id         = %s
	module_attestation_policies = [{
		predicate_type = "some-predicate"
		public_key     = "%s"
	}]
}

`, ruleType, ruleStage, ruleParentID, dummyPublicKey)
}

func testManagedIdentityAccessRulesConfigurationUpdate() string {
	// Only the run stage can be changed.
	ruleStage := "plan"
	updateStage := "apply"
	return strings.Replace(testManagedIdentityAccessRulesConfigurationRule(), ruleStage, updateStage, 1)
}
