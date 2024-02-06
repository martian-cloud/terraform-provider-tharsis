package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	ttypes "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

// TestAssignedManagedIdentity tests creation, reading, updating, and deletion of an assigned managed identity resource,
// also known as assigning and unassigning a managed identity to/from a workspace.
func TestAssignedManagedIdentity(t *testing.T) {
	resource.Test(t, resource.TestCase{
		// Tharsis assigned managed identities
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and read back an assigned managed identity.
			{
				Config: testSharedProviderConfiguration() + testAssignedManagedIdentityConfiguration(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// There are no values in an assigned managed identity that are known.

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_assigned_managed_identity.tami1", "managed_identity_id"),
					resource.TestCheckResourceAttrSet("tharsis_assigned_managed_identity.tami1", "workspace_id"),
				),
			},

			// Import state.
			{
				ResourceName:      "tharsis_assigned_managed_identity.tami1",
				ImportState:       true,
				ImportStateVerify: true,
			},

			// Update (which requires replacement) and read back.
			{
				Config: testAssignedManagedIdentityConfiguration(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// There are no values in an assigned managed identity that are known.

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_assigned_managed_identity.tami1", "managed_identity_id"),
					resource.TestCheckResourceAttrSet("tharsis_assigned_managed_identity.tami1", "workspace_id"),
				),
			},

			// Destroy should be covered automatically by TestCase.
		},
	})
}

func testAssignedManagedIdentityConfiguration() string {
	createWorkspaceName := "tw_name"
	createWorkspaceDescription := "this is tw, a test workspace"
	createMaxJobDuration := 1234      // must not exceed 1440
	createTerraformVersion := "1.2.3" // must be a valid version
	createPreventDestroyPlan := true

	createType := string(ttypes.ManagedIdentityTharsisFederated)
	createManagedIdentityName := "tmi_tharsis_name"
	createManagedIdentityDescription := "this is tmi_tharsis, a Tharsis managed identity of Tharsis type"
	createTharsisServiceAccountPath := "some-tharsis-service-account-path"

	return createRootGroup(testGroupPath, "this is a test root group") +
		fmt.Sprintf(`

	resource "tharsis_workspace" "tw" {
		name = "%s"
		description = "%s"
		group_path = tharsis_group.root-group.full_path
		max_job_duration = "%d"
		terraform_version = "%s"
		prevent_destroy_plan = "%v"
	}
		`, createWorkspaceName, createWorkspaceDescription,
			createMaxJobDuration, createTerraformVersion, createPreventDestroyPlan) +
		fmt.Sprintf(`

			resource "tharsis_managed_identity" "tmi_tharsis" {
				type                         = "%s"
				name                         = "%s"
				description                  = "%s"
				group_path                   = tharsis_group.root-group.full_path
				tharsis_service_account_path = "%s"
			}

				`, createType, createManagedIdentityName, createManagedIdentityDescription, createTharsisServiceAccountPath) +
		`

resource "tharsis_assigned_managed_identity" "tami1" {
	managed_identity_id = tharsis_managed_identity.tmi_tharsis.id
	workspace_id = tharsis_workspace.tw.id
}
`
}
