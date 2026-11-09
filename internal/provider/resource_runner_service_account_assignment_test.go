package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// TestRunnerServiceAccountAssignment tests creation, reading, and deletion of a runner
// service account assignment — assigning and unassigning a service account to/from a
// runner.
//
// The service account is passed as a TRN and the runner as a Global ID, on
// purpose: the Read resolves both inputs to Global IDs at create and matches on those,
// so a TRN input must still be found on refresh. (The assigned-managed-identity
// precedent matches the raw input against Global IDs, where a TRN input never matches.)
func TestRunnerServiceAccountAssignment(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and read back the assignment.
			{
				Config: testRunnerServiceAccountAssignmentConfiguration(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_runner_service_account_assignment.trasa", "id"),
					resource.TestCheckResourceAttrSet("tharsis_runner_service_account_assignment.trasa", "runner_id"),
					resource.TestCheckResourceAttrSet("tharsis_runner_service_account_assignment.trasa", "service_account_id"),
				),
			},

			// Re-apply the same configuration and read back. With a TRN service account
			// input, this is the step that fails if Read matched the raw input against
			// Global IDs: the resource would drop from state and plan a recreate.
			{
				Config: testRunnerServiceAccountAssignmentConfiguration(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("tharsis_runner_service_account_assignment.trasa", "id"),
					resource.TestCheckResourceAttrSet("tharsis_runner_service_account_assignment.trasa", "runner_id"),
					resource.TestCheckResourceAttrSet("tharsis_runner_service_account_assignment.trasa", "service_account_id"),
				),
			},

			// Import by "<runner_id>/<service_account_id>". The configuration names the service
			// account by TRN, so the import ID takes its Global ID from the service account
			// resource; the next plan's TRN is then a form switch, not a replacement.
			{
				ResourceName:      "tharsis_runner_service_account_assignment.trasa",
				ImportState:       true,
				ImportStateIdFunc: testRunnerServiceAccountAssignmentImportID,
				ImportStateCheck:  testRunnerServiceAccountAssignmentImportStateCheck,
			},

			// Destroy should be covered automatically by TestCase.
		},
	})
}

// testRunnerServiceAccountAssignmentImportID builds the composite import ID from the Global IDs of the
// runner and service account in state.
func testRunnerServiceAccountAssignmentImportID(s *terraform.State) (string, error) {
	runner, ok := s.RootModule().Resources["tharsis_runner.tra"]
	if !ok {
		return "", fmt.Errorf("tharsis_runner.tra not found in state")
	}
	sa, ok := s.RootModule().Resources["tharsis_service_account.tsa"]
	if !ok {
		return "", fmt.Errorf("tharsis_service_account.tsa not found in state")
	}
	return runner.Primary.ID + "/" + sa.Primary.ID, nil
}

// testRunnerServiceAccountAssignmentImportStateCheck checks the imported assignment holds both IDs and a
// generated id.
func testRunnerServiceAccountAssignmentImportStateCheck(states []*terraform.InstanceState) error {
	if len(states) != 1 {
		return fmt.Errorf("expected one imported instance, got %d", len(states))
	}
	for _, name := range []string{"id", "runner_id", "service_account_id"} {
		if states[0].Attributes[name] == "" {
			return fmt.Errorf("imported state is missing %s", name)
		}
	}
	return nil
}

func testRunnerServiceAccountAssignmentConfiguration() string {
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

	resource "tharsis_runner_service_account_assignment" "trasa" {
		# runner by Global ID, service account by TRN — both input forms exercised.
		runner_id    = tharsis_runner.tra.id
		service_account_id = "trn:service_account:${tharsis_service_account.tsa.resource_path}"
	}

	`
}
