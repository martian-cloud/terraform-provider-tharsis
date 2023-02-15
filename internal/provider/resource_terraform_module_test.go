package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestTerraformModule(t *testing.T) {
	createName := "ttm_name"
	createSystem := "aws"
	createGroupPath := testGroupPath
	createRepositoryURL := "http://somewhere.example.invalid/somewhere" // optional
	createPrivate := true                                               // optional

	updateSystem := "azure"
	updateRepositoryURL := "http://somewhere.example.invalid/else"
	updatePrivate := false

	resource.Test(t, resource.TestCase{

		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{

			// Create and read back a terraform module.
			{
				Config: testTerraformModuleConfigurationCreate(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_terraform_module.ttm", "name", createName),
					resource.TestCheckResourceAttr("tharsis_terraform_module.ttm", "system", createSystem),
					resource.TestCheckResourceAttr("tharsis_terraform_module.ttm", "group_path", createGroupPath),
					resource.TestCheckResourceAttr("tharsis_terraform_module.ttm", "repository_url", createRepositoryURL),
					resource.TestCheckResourceAttr("tharsis_terraform_module.ttm", "private", strconv.FormatBool(createPrivate)),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_terraform_module.ttm", "id"),
					resource.TestCheckResourceAttrSet("tharsis_terraform_module.ttm", "resource_path"),
					resource.TestCheckResourceAttrSet("tharsis_terraform_module.ttm", "registry_namespace"),
					resource.TestCheckResourceAttrSet("tharsis_terraform_module.ttm", "last_updated"),
				),
			},

			// Import the state.
			// The import state ID is the resource path, which is group path / module name / system.
			{
				ResourceName:      "tharsis_terraform_module.ttm",
				ImportStateId:     createGroupPath + "/" + createName + "/" + createSystem,
				ImportState:       true,
				ImportStateVerify: true,
			},

			// Update and read.
			{
				Config: testTerraformModuleConfigurationUpdate(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_terraform_module.ttm", "name", createName),
					resource.TestCheckResourceAttr("tharsis_terraform_module.ttm", "system", updateSystem),
					resource.TestCheckResourceAttr("tharsis_terraform_module.ttm", "group_path", createGroupPath),
					resource.TestCheckResourceAttr("tharsis_terraform_module.ttm", "repository_url", updateRepositoryURL),
					resource.TestCheckResourceAttr("tharsis_terraform_module.ttm", "private", strconv.FormatBool(updatePrivate)),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_terraform_module.ttm", "id"),
					resource.TestCheckResourceAttrSet("tharsis_terraform_module.ttm", "resource_path"),
					resource.TestCheckResourceAttrSet("tharsis_terraform_module.ttm", "registry_namespace"),
					resource.TestCheckResourceAttrSet("tharsis_terraform_module.ttm", "last_updated"),
				),
			},

			// Destroy should be covered automatically by TestCase.

		},
	})
}

func testTerraformModuleConfigurationCreate() string {
	createName := "ttm_name"
	createSystem := "aws"
	createGroupPath := testGroupPath
	createRepositoryURL := "http://somewhere.example.invalid/somewhere"
	createPrivate := true

	return fmt.Sprintf(`

	resource "tharsis_terraform_module" "ttm" {
		name = "%s"
		system = "%s"
		group_path = "%s"
		repository_url = "%s"
		private = "%v"

}
	`, createName, createSystem, createGroupPath, createRepositoryURL, createPrivate)
}

func testTerraformModuleConfigurationUpdate() string {
	createName := "ttm_name"
	updateSystem := "azure"
	createGroupPath := testGroupPath
	updateRepositoryURL := "http://somewhere.example.invalid/else"
	updatePrivate := false

	return fmt.Sprintf(`

	resource "tharsis_terraform_module" "ttm" {
		name = "%s"
		system = "%s"
		group_path = "%s"
		repository_url = "%s"
		private = "%v"
	}
		`, createName, updateSystem, createGroupPath, updateRepositoryURL, updatePrivate)
}

// The End.
