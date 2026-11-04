package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// TestRunnerServiceAccount tests creation, reading, and deletion of a runner
// service account assignment — assigning and unassigning a service account to/from a
// runner.
//
// The service account is passed as a TRN and the runner as a Global ID, on
// purpose: the Read resolves both inputs to Global IDs at create and matches on those,
// so a TRN input must still be found on refresh. (The assigned-managed-identity
// precedent matches the raw input against Global IDs, where a TRN input never matches.)
func TestRunnerServiceAccount(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and read back the assignment.
			{
				Config: testRunnerServiceAccountConfiguration(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_runner_service_account.trasa", "id"),
					resource.TestCheckResourceAttrSet("tharsis_runner_service_account.trasa", "runner_id"),
					resource.TestCheckResourceAttrSet("tharsis_runner_service_account.trasa", "service_account_id"),
				),
			},

			// Re-apply the same configuration and read back. With a TRN service account
			// input, this is the step that fails if Read matched the raw input against
			// Global IDs: the resource would drop from state and plan a recreate.
			{
				Config: testRunnerServiceAccountConfiguration(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("tharsis_runner_service_account.trasa", "id"),
					resource.TestCheckResourceAttrSet("tharsis_runner_service_account.trasa", "runner_id"),
					resource.TestCheckResourceAttrSet("tharsis_runner_service_account.trasa", "service_account_id"),
				),
			},

			// Destroy should be covered automatically by TestCase.
		},
	})
}

func testRunnerServiceAccountConfiguration() string {
	return createRootGroup(testGroupPath, "this is a test root group") +
		`

	resource "tharsis_runner" "tra" {
		name     = "trasa_runner"
		group_id = tharsis_group.root-group.id
		tags     = ["trasa"]
	}

	resource "tharsis_service_account" "tsa" {
		name                = "trasa_service_account"
		description         = "service account for the runner assignment test"
		group_id            = tharsis_group.root-group.id
		oidc_trust_policies = [{bound_claims = {"sub" = "some-subject"}, issuer = "https://issuer.example.com/"}]
	}

	resource "tharsis_runner_service_account" "trasa" {
		# runner by Global ID, service account by TRN — both input forms exercised.
		runner_id    = tharsis_runner.tra.id
		service_account_id = "trn:service_account:${tharsis_service_account.tsa.resource_path}"
	}

	`
}
