package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

type testWorkspaceConstants struct {
	createName                string
	createDescription         string
	createGroupPath           string
	createFullPath            string
	createMaxJobDuration      int
	createTerraformVersion    string
	createPreventDestroyPlan  bool
	updatedDescription        string
	updatedMaxJobDuration     int
	updatedTerraformVersion   string
	updatedPreventDestroyPlan bool
}

func TestWorkspace(t *testing.T) {
	c := buildTestWorkspaceConstants()

	resource.Test(t, resource.TestCase{

		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{

			// Create and read back a workspace.
			{
				Config: testWorkspaceConfigurationCreate(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_workspace.tw", "name", c.createName),
					resource.TestCheckResourceAttr("tharsis_workspace.tw", "description", c.createDescription),
					resource.TestCheckResourceAttr("tharsis_workspace.tw", "full_path", c.createFullPath),
					resource.TestCheckResourceAttr("tharsis_workspace.tw", "group_path", c.createGroupPath),
					resource.TestCheckResourceAttr("tharsis_workspace.tw", "max_job_duration", strconv.Itoa(c.createMaxJobDuration)),
					resource.TestCheckResourceAttr("tharsis_workspace.tw", "terraform_version", c.createTerraformVersion),
					resource.TestCheckResourceAttr("tharsis_workspace.tw", "prevent_destroy_plan", strconv.FormatBool(c.createPreventDestroyPlan)),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_workspace.tw", "id"),
					resource.TestCheckResourceAttrSet("tharsis_workspace.tw", "last_updated"),
				),
			},

			// Import the state.
			{
				ResourceName:      "tharsis_workspace.tw",
				ImportState:       true,
				ImportStateVerify: true,
			},

			// Update and read.
			{
				Config: testWorkspaceConfigurationUpdate(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_workspace.tw", "name", c.createName),
					resource.TestCheckResourceAttr("tharsis_workspace.tw", "description", c.updatedDescription),
					resource.TestCheckResourceAttr("tharsis_workspace.tw", "full_path", c.createFullPath),
					resource.TestCheckResourceAttr("tharsis_workspace.tw", "group_path", c.createGroupPath),
					resource.TestCheckResourceAttr("tharsis_workspace.tw", "max_job_duration", strconv.Itoa(c.updatedMaxJobDuration)),
					resource.TestCheckResourceAttr("tharsis_workspace.tw", "terraform_version", c.updatedTerraformVersion),
					resource.TestCheckResourceAttr("tharsis_workspace.tw", "prevent_destroy_plan", strconv.FormatBool(c.updatedPreventDestroyPlan)),

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
	c := buildTestWorkspaceConstants()
	return fmt.Sprintf(`

resource "tharsis_workspace" "tw" {
	name = "%s"
	description = "%s"
	group_path = "%s"
	max_job_duration = "%d"
	terraform_version = "%s"
	prevent_destroy_plan = "%v"
}
	`, c.createName, c.createDescription, c.createGroupPath,
		c.createMaxJobDuration, c.createTerraformVersion, c.createPreventDestroyPlan)
}

func testWorkspaceConfigurationUpdate() string {
	c := buildTestWorkspaceConstants()
	return fmt.Sprintf(`

resource "tharsis_workspace" "tw" {
	name = "%s"
	description = "%s"
	group_path = "%s"
	max_job_duration = "%d"
	terraform_version = "%s"
	prevent_destroy_plan = "%v"
}
	`, c.createName, c.updatedDescription, c.createGroupPath,
		c.updatedMaxJobDuration, c.updatedTerraformVersion, c.updatedPreventDestroyPlan)
}

func buildTestWorkspaceConstants() *testWorkspaceConstants {
	createName := "tw_name"
	return &testWorkspaceConstants{
		createName:                createName,
		createDescription:         "this is tw, a test workspace",
		createGroupPath:           testGroupPath,
		createFullPath:            testGroupPath + "/" + createName,
		createMaxJobDuration:      1234,    // must not exceed 1440
		createTerraformVersion:    "1.2.3", // must be a valid version
		createPreventDestroyPlan:  true,
		updatedDescription:        "this is an updated description for tw, a test workspace",
		updatedMaxJobDuration:     1357,    // must not exceed 1440
		updatedTerraformVersion:   "1.3.5", // must be a valid version
		updatedPreventDestroyPlan: false,
	}
}

// The End.
