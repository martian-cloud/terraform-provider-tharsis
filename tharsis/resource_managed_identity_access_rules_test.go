package tharsis

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
	parentType := string(ttypes.ManagedIdentityAWSFederated)
	parentName := "tmiar_parent_name"
	parentDescription := "this is tmiar_parent, a Tharsis managed identity"
	parentAWSRole := "some-iam-aws-role"
	parentConfig := fmt.Sprintf(`

	resource "tharsis_managed_identity" "tmiar_parent" {
		type        = "%s"
		name        = "%s"
		description = "%s"
		group_path  = "%s"
		aws_role    = "%s"
	}

	`, parentType, parentName, parentDescription, testGroupPath, parentAWSRole)

	// TODO: When we have the ability to create the parent group, users, service accounts, and teams, add them.

	// Configuration to create the access rule(s).
	ruleStage := "plan"
	ruleParentID := "tharsis_managed_identity.tmiar_parent.id"
	ruleConfig := fmt.Sprintf(`

	resource "tharsis_managed_identity_access_rule" "rule01" {
		run_stage                = "%s"
		managed_identity_id      = %s
		allowed_users            = []
		allowed_service_accounts = []
		allowed_teams            = []
	}

	`, ruleStage, ruleParentID)

	// Configuration to update the access rule(s).
	// Only the run stage can be changed.
	updateStage := "apply"
	updateConfig := strings.Replace(ruleConfig, ruleStage, updateStage, 1)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{

			// Create the parent managed identity and the access rule in one step.
			// If done in separate steps, the access rule can't find its parent.
			{
				Config: providerConfig + parentConfig + ruleConfig,
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
				Config: providerConfig + parentConfig + updateConfig,
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

// The End.
