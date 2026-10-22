package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// Default Tharsis role names, used here so the test does not depend on a role-management resource.
const (
	testDeployerRoleName = "deployer"
	testViewerRoleName   = "viewer"
)

// TestWorkspaceRoleBinding tests creation, reading, updating, and deletion of a workspace role binding resource.
func TestWorkspaceRoleBinding(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and read back a workspace role binding.
			{
				Config: testWorkspaceRoleBindingConfiguration(testDeployerRoleName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("tharsis_workspace_role_binding.twrb", "role_name", testDeployerRoleName),
					resource.TestCheckResourceAttrPair("tharsis_workspace_role_binding.twrb", "workspace_id",
						"tharsis_workspace.tw", "id"),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_workspace_role_binding.twrb", "id"),
				),
			},

			// Import state.
			{
				ResourceName:      "tharsis_workspace_role_binding.twrb",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: testAccWorkspaceRoleBindingImportStateIDFunc,
			},

			// Update and read.
			{
				Config: testWorkspaceRoleBindingConfiguration(testViewerRoleName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("tharsis_workspace_role_binding.twrb", "role_name", testViewerRoleName),
					resource.TestCheckResourceAttrPair("tharsis_workspace_role_binding.twrb", "workspace_id",
						"tharsis_workspace.tw", "id"),

					resource.TestCheckResourceAttrSet("tharsis_workspace_role_binding.twrb", "id"),
				),
			},

			// Destroy should be covered automatically by TestCase.
		},
	})
}

// testAccWorkspaceRoleBindingImportStateIDFunc returns the workspace ID to import by,
// since a workspace role binding is identified by its workspace, not its own ID.
func testAccWorkspaceRoleBindingImportStateIDFunc(state *terraform.State) (string, error) {
	rs, ok := state.RootModule().Resources["tharsis_workspace_role_binding.twrb"]
	if !ok {
		return "", fmt.Errorf("resource not found: tharsis_workspace_role_binding.twrb")
	}
	return rs.Primary.Attributes["workspace_id"], nil
}

// testWorkspaceRoleBindingConfiguration returns a configuration that binds the named role to a test workspace.
func testWorkspaceRoleBindingConfiguration(roleName string) string {
	createName := "tw_for_role_binding"
	createDescription := "this is tw_for_role_binding, a test workspace for a role binding"

	return fmt.Sprintf(`

%s

resource "tharsis_workspace" "tw" {
	name                 = "%s"
	description          = "%s"
	group_path           = tharsis_group.root-group.full_path
	prevent_destroy_plan = false
}

resource "tharsis_workspace_role_binding" "twrb" {
	workspace_id = tharsis_workspace.tw.id
	role_name    = "%s"
}

	`, createRootGroup(testGroupPath, "this is a test root group"), createName, createDescription, roleName)
}
