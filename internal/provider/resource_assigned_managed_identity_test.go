package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
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
	return createRootGroup(testGroupPath, "this is a test root group") +
		`

	resource "tharsis_workspace" "tw" {
		name = "tw_name"
		description = "this is tw, a test workspace"
		group_path = tharsis_group.root-group.full_path
		max_job_duration = "1234"
		terraform_version = "1.2.3"
		prevent_destroy_plan = "true"
	}

	resource "tharsis_managed_identity" "tmi_tharsis" {
		type                         = "tharsis_federated"
		name                         = "tmi_tharsis_name"
		description                  = "this is tmi_tharsis, a Tharsis managed identity of Tharsis type"
		group_path                   = tharsis_group.root-group.full_path
		tharsis_service_account_path = "some-tharsis-service-account-path"
	}

	resource "tharsis_assigned_managed_identity" "tami1" {
		managed_identity_id = tharsis_managed_identity.tmi_tharsis.id
		workspace_id = tharsis_workspace.tw.id
	}

	`
}
