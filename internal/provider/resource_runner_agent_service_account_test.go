package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// TestRunnerAgentServiceAccount tests creation, reading, and deletion of a runner agent
// service account assignment — assigning and unassigning a service account to/from a
// runner agent.
//
// The service account is passed as a TRN and the runner agent as a Global ID, on
// purpose: the Read resolves both inputs to Global IDs at create and matches on those,
// so a TRN input must still be found on refresh. (The assigned-managed-identity
// precedent matches the raw input against Global IDs, where a TRN input never matches.)
func TestRunnerAgentServiceAccount(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and read back the assignment.
			{
				Config: testRunnerAgentServiceAccountConfiguration(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_runner_agent_service_account.trasa", "id"),
					resource.TestCheckResourceAttrSet("tharsis_runner_agent_service_account.trasa", "runner_agent_id"),
					resource.TestCheckResourceAttrSet("tharsis_runner_agent_service_account.trasa", "service_account_id"),
				),
			},

			// Re-apply the same configuration and read back. With a TRN service account
			// input, this is the step that fails if Read matched the raw input against
			// Global IDs: the resource would drop from state and plan a recreate.
			{
				Config: testRunnerAgentServiceAccountConfiguration(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("tharsis_runner_agent_service_account.trasa", "id"),
					resource.TestCheckResourceAttrSet("tharsis_runner_agent_service_account.trasa", "runner_agent_id"),
					resource.TestCheckResourceAttrSet("tharsis_runner_agent_service_account.trasa", "service_account_id"),
				),
			},

			// Destroy should be covered automatically by TestCase.
		},
	})
}

func testRunnerAgentServiceAccountConfiguration() string {
	return createRootGroup(testGroupPath, "this is a test root group") +
		`

	resource "tharsis_runner_agent" "tra" {
		name     = "trasa_runner_agent"
		group_id = tharsis_group.root-group.id
		tags     = ["trasa"]
	}

	resource "tharsis_service_account" "tsa" {
		name                = "trasa_service_account"
		description         = "service account for the runner agent assignment test"
		group_id            = tharsis_group.root-group.id
		oidc_trust_policies = [{bound_claims = {"sub" = "some-subject"}, issuer = "https://issuer.example.com/"}]
	}

	resource "tharsis_runner_agent_service_account" "trasa" {
		# runner agent by Global ID, service account by TRN — both input forms exercised.
		runner_agent_id    = tharsis_runner_agent.tra.id
		service_account_id = "trn:service_account:${tharsis_service_account.tsa.resource_path}"
	}

	`
}
