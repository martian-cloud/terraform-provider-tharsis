package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestWorkspace(t *testing.T) {
	createName := "tw_name"
	createDescription := "this is tw, a test workspace"
	createGroupPath := testGroupPath
	createFullPath := testGroupPath + "/" + createName
	createMaxJobDuration := 1234      // must not exceed 1440
	createTerraformVersion := "1.2.3" // must be a valid version
	createPreventDestroyPlan := true
	updatedDescription := "this is an updated description for tw, a test workspace"
	updatedMaxJobDuration := 1357      // must not exceed 1440
	updatedTerraformVersion := "1.3.5" // must be a valid version
	updatedPreventDestroyPlan := false

	resource.Test(t, resource.TestCase{

		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{

			// Create and read back a workspace.
			{
				Config: testWorkspaceConfigurationCreate(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_workspace.tw", "name", createName),
					resource.TestCheckResourceAttr("tharsis_workspace.tw", "description", createDescription),
					resource.TestCheckResourceAttr("tharsis_workspace.tw", "full_path", createFullPath),
					resource.TestCheckResourceAttr("tharsis_workspace.tw", "group_path", createGroupPath),
					resource.TestCheckResourceAttr("tharsis_workspace.tw", "max_job_duration", strconv.Itoa(createMaxJobDuration)),
					resource.TestCheckResourceAttr("tharsis_workspace.tw", "terraform_version", createTerraformVersion),
					resource.TestCheckResourceAttr("tharsis_workspace.tw", "prevent_destroy_plan", strconv.FormatBool(createPreventDestroyPlan)),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_workspace.tw", "id"),
					resource.TestCheckResourceAttrSet("tharsis_workspace.tw", "last_updated"),
				),
			},

			// Import the state.
			{
				ResourceName:      "tharsis_workspace.tw",
				ImportStateId:     createFullPath,
				ImportState:       true,
				ImportStateVerify: true,
			},

			// Update and read.
			{
				Config: testWorkspaceConfigurationUpdate(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_workspace.tw", "name", createName),
					resource.TestCheckResourceAttr("tharsis_workspace.tw", "description", updatedDescription),
					resource.TestCheckResourceAttr("tharsis_workspace.tw", "full_path", createFullPath),
					resource.TestCheckResourceAttr("tharsis_workspace.tw", "group_path", createGroupPath),
					resource.TestCheckResourceAttr("tharsis_workspace.tw", "max_job_duration", strconv.Itoa(updatedMaxJobDuration)),
					resource.TestCheckResourceAttr("tharsis_workspace.tw", "terraform_version", updatedTerraformVersion),
					resource.TestCheckResourceAttr("tharsis_workspace.tw", "prevent_destroy_plan", strconv.FormatBool(updatedPreventDestroyPlan)),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_workspace.tw", "id"),
					resource.TestCheckResourceAttrSet("tharsis_workspace.tw", "last_updated"),
				),
			},

			// Destroy should be covered automatically by TestCase.

		},
	})
}

func testWorkspaceConfigurationCreate() string {
	createName := "tw_name"
	createDescription := "this is tw, a test workspace"
	createMaxJobDuration := 1234      // must not exceed 1440
	createTerraformVersion := "1.2.3" // must be a valid version
	createPreventDestroyPlan := true

	return fmt.Sprintf(`

%s

resource "tharsis_workspace" "tw" {
	name = "%s"
	description = "%s"
	group_path = tharsis_group.root-group.full_path
	max_job_duration = "%d"
	terraform_version = "%s"
	prevent_destroy_plan = "%v"
}
	`, createRootGroup(), createName, createDescription,
		createMaxJobDuration, createTerraformVersion, createPreventDestroyPlan)
}

func testWorkspaceConfigurationUpdate() string {
	createName := "tw_name"
	updatedDescription := "this is an updated description for tw, a test workspace"
	updatedMaxJobDuration := 1357      // must not exceed 1440
	updatedTerraformVersion := "1.3.5" // must be a valid version
	updatedPreventDestroyPlan := false

	return fmt.Sprintf(`

%s

resource "tharsis_workspace" "tw" {
	name = "%s"
	description = "%s"
	group_path = tharsis_group.root-group.full_path
	max_job_duration = "%d"
	terraform_version = "%s"
	prevent_destroy_plan = "%v"
}
	`, createRootGroup(), createName, updatedDescription,
		updatedMaxJobDuration, updatedTerraformVersion, updatedPreventDestroyPlan)
}

// The End.
