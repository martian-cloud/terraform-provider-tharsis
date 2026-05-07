package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestWorkspace(t *testing.T) {
	createName := "tw_name"
	createDescription := "this is tw, a test workspace"
	createGroupPath := testGroupPath
	createFullPath := testGroupPath + "/" + createName
	updatedDescription := "this is an updated description for tw, a test workspace"

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

	return fmt.Sprintf(`

%s

resource "tharsis_workspace" "tw" {
	name = "%s"
	description = "%s"
	group_path = tharsis_group.root-group.full_path
	max_job_duration = 720
	terraform_version = "1.5.0"
	prevent_destroy_plan = false
}
	`, createRootGroup(testGroupPath, "this is a test root group"), createName, createDescription)
}

func testWorkspaceConfigurationUpdate() string {
	createName := "tw_name"
	updatedDescription := "this is an updated description for tw, a test workspace"

	return fmt.Sprintf(`

%s

resource "tharsis_workspace" "tw" {
	name = "%s"
	description = "%s"
	group_path = tharsis_group.root-group.full_path
	max_job_duration = 1440
	terraform_version = "1.6.0"
	prevent_destroy_plan = true
}
	`, createRootGroup(testGroupPath, "this is a test root group"), createName, updatedDescription)
}
