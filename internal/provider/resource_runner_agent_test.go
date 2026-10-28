package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestRunnerAgent(t *testing.T) {
	createName := "tra_name"
	createDescription := "this is tra, a test runner agent"
	createResourcePath := testGroupPath + "/" + createName
	updatedDescription := "this is an updated description for tra, a test runner agent"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and read back a runner agent.
			{
				Config: testRunnerAgentConfigurationCreate(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_runner_agent.tra", "name", createName),
					resource.TestCheckResourceAttr("tharsis_runner_agent.tra", "description", createDescription),
					resource.TestCheckResourceAttr("tharsis_runner_agent.tra", "resource_path", createResourcePath),
					resource.TestCheckResourceAttr("tharsis_runner_agent.tra", "disabled", "false"),
					resource.TestCheckResourceAttr("tharsis_runner_agent.tra", "run_untagged_jobs", "false"),
					resource.TestCheckResourceAttr("tharsis_runner_agent.tra", "tags.#", "2"),
					resource.TestCheckTypeSetElemAttr("tharsis_runner_agent.tra", "tags.*", "alpha"),
					resource.TestCheckTypeSetElemAttr("tharsis_runner_agent.tra", "tags.*", "beta"),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_runner_agent.tra", "id"),
					resource.TestCheckResourceAttrSet("tharsis_runner_agent.tra", "group_id"),
				),
			},

			// Import the state.
			{
				ResourceName:      "tharsis_runner_agent.tra",
				ImportState:       true,
				ImportStateVerify: true,
			},

			// Update description, disabled, run_untagged_jobs and tags, then read back.
			{
				Config: testRunnerAgentConfigurationUpdate(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("tharsis_runner_agent.tra", "name", createName),
					resource.TestCheckResourceAttr("tharsis_runner_agent.tra", "description", updatedDescription),
					resource.TestCheckResourceAttr("tharsis_runner_agent.tra", "resource_path", createResourcePath),
					resource.TestCheckResourceAttr("tharsis_runner_agent.tra", "disabled", "true"),
					resource.TestCheckResourceAttr("tharsis_runner_agent.tra", "run_untagged_jobs", "true"),
					resource.TestCheckResourceAttr("tharsis_runner_agent.tra", "tags.#", "1"),
					resource.TestCheckTypeSetElemAttr("tharsis_runner_agent.tra", "tags.*", "gamma"),

					resource.TestCheckResourceAttrSet("tharsis_runner_agent.tra", "id"),
				),
			},

			// Destroy should be covered automatically by TestCase.
		},
	})
}

func testRunnerAgentConfigurationCreate() string {
	createName := "tra_name"
	createDescription := "this is tra, a test runner agent"

	return fmt.Sprintf(`

%s

resource "tharsis_runner_agent" "tra" {
	name        = "%s"
	description = "%s"
	group_id    = tharsis_group.root-group.id
	tags        = ["alpha", "beta"]
}
	`, createRootGroup(testGroupPath, "this is a test root group"), createName, createDescription)
}

func testRunnerAgentConfigurationUpdate() string {
	createName := "tra_name"
	updatedDescription := "this is an updated description for tra, a test runner agent"

	return fmt.Sprintf(`

%s

resource "tharsis_runner_agent" "tra" {
	name              = "%s"
	description       = "%s"
	group_id          = tharsis_group.root-group.id
	disabled          = true
	run_untagged_jobs = true
	tags              = ["gamma"]
}
	`, createRootGroup(testGroupPath, "this is a test root group"), createName, updatedDescription)
}
