package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	moduleSource = "registry.terraform.io/martian-cloud/module/null"
)

func TestApplyModule(t *testing.T) {
	ws1Name := "workspace-1"
	ws1Desc := "this is workspace 1"
	ws1Path := testGroupPath + "/" + ws1Name
	ws2Name := "workspace-2"
	ws2Desc := "this is workspace 2"
	wsPreventDestroyPlan := false
	varValueBase := "some variable value "
	varKey := "trigger_name"
	varCategory := "terraform"
	varHCL := false

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a root group and two workspaces.
			{
				Config: testApplyModuleConfigurationCreate(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_group.root-group", "name", testGroupPath),
					resource.TestCheckResourceAttr("tharsis_workspace.tw1", "name", ws1Name),
					resource.TestCheckResourceAttr("tharsis_workspace.tw1", "description", ws1Desc),
					resource.TestCheckResourceAttr("tharsis_workspace.tw1", "group_path", testGroupPath),
					resource.TestCheckResourceAttr("tharsis_workspace.tw1", "prevent_destroy_plan",
						strconv.FormatBool(wsPreventDestroyPlan)),
					resource.TestCheckResourceAttr("tharsis_workspace.tw2", "name", ws2Name),
					resource.TestCheckResourceAttr("tharsis_workspace.tw2", "description", ws2Desc),
					resource.TestCheckResourceAttr("tharsis_workspace.tw2", "group_path", testGroupPath),
					resource.TestCheckResourceAttr("tharsis_workspace.tw2", "prevent_destroy_plan",
						strconv.FormatBool(wsPreventDestroyPlan)),
				),
			},

			// Do the apply/create run.
			{
				Config: testApplyModuleConfigurationCreate() + testDoApplyCreateRun(1),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					testAccCheckTharsisApplyModuleExists("tharsis_apply_module.tam", true),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "workspace_path", ws1Path),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "module_source", moduleSource),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "refresh", strconv.FormatBool(true)),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "variables.0.value", varValueBase+"1"),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "variables.0.key", varKey),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "variables.0.category", varCategory),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "variables.0.hcl", strconv.FormatBool(varHCL)),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "resolved_variables.0.value", varValueBase+"1"),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "resolved_variables.0.namespace_path", ""),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "resolved_variables.0.key", varKey),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "resolved_variables.0.category", varCategory),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "resolved_variables.0.hcl", strconv.FormatBool(varHCL)),
				),
			},

			// Repeat the apply/create run with no changes.
			{
				Config: testApplyModuleConfigurationCreate() + testDoApplyCreateRun(1),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					testAccCheckTharsisApplyModuleExists("tharsis_apply_module.tam", true),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "workspace_path", ws1Path),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "module_source", moduleSource),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "refresh", strconv.FormatBool(true)),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "variables.0.value", varValueBase+"1"),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "variables.0.key", varKey),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "variables.0.category", varCategory),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "variables.0.hcl", strconv.FormatBool(varHCL)),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "resolved_variables.0.value", varValueBase+"1"),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "resolved_variables.0.namespace_path", ""),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "resolved_variables.0.key", varKey),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "resolved_variables.0.category", varCategory),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "resolved_variables.0.hcl", strconv.FormatBool(varHCL)),
				),
			},

			// Do an apply/create run with changes to the variable's value.
			{
				Config: testApplyModuleConfigurationCreate() + testDoApplyCreateRun(2),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					testAccCheckTharsisApplyModuleExists("tharsis_apply_module.tam", true),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "workspace_path", ws1Path),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "module_source", moduleSource),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "refresh", strconv.FormatBool(true)),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "variables.0.value", varValueBase+"2"),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "variables.0.key", varKey),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "variables.0.category", varCategory),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "variables.0.hcl", strconv.FormatBool(varHCL)),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "resolved_variables.0.value", varValueBase+"2"),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "resolved_variables.0.namespace_path", ""),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "resolved_variables.0.key", varKey),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "resolved_variables.0.category", varCategory),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "resolved_variables.0.hcl", strconv.FormatBool(varHCL)),
				),
			},

			// Do a destroy/delete run.
			{
				Config: testApplyModuleConfigurationCreate(), // Remove the tharsis_apply_module resource.
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify the removal/absence of the resource that should have been destroyed/deleted.
					testAccCheckTharsisApplyModuleExists("tharsis_apply_module.tam", false),
				),
			},

			// The rest of the destruction should be covered automatically by TestCase.

		},
	})
}

// testAccCheckTharsisApplyModuleExists returns a checker function to verify that a specified
// TharsisApplyModule resource does or does not exist in the state.
// This example verifies via the API whether the object really exists,
// but did not find documentation on how to get a handle to our provider, which would give us handle to the SDK.
// See https://github.com/hashicorp/terraform-plugin-testing/blob/main/website/docs/plugin/testing/testing-patterns.mdx
func testAccCheckTharsisApplyModuleExists(tfName string, shouldExist bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Look up the object in the state, if it's there.
		_, ok := s.RootModule().Resources[tfName]

		switch {
		case shouldExist && !ok:
			return fmt.Errorf("Resource should exist but was not found in state: %s", tfName)
		case !shouldExist && ok:
			return fmt.Errorf("Resource should not exist but was found in state: %s", tfName)
		}

		return nil
	}
}

func testApplyModuleConfigurationCreate() string {
	ws1Name := "workspace-1"
	ws1Desc := "this is workspace 1"
	ws2Name := "workspace-2"
	ws2Desc := "this is workspace 2"
	wsPreventDestroyPlan := false

	return fmt.Sprintf(`

%s

resource "tharsis_workspace" "tw1" {
	name                 = "%s"
	description          = "%s"
	group_path           = tharsis_group.root-group.full_path
	prevent_destroy_plan = "%v"
}

resource "tharsis_workspace" "tw2" {
	name                 = "%s"
	description          = "%s"
	group_path           = tharsis_group.root-group.full_path
	prevent_destroy_plan = "%v"
}

	`, createRootGroup(testGroupPath, "this is a test root group"),
		ws1Name, ws1Desc, wsPreventDestroyPlan,
		ws2Name, ws2Desc, wsPreventDestroyPlan,
	)
}

func testDoApplyCreateRun(val int) string {
	ws1Name := "workspace-1"
	ws1Path := testGroupPath + "/" + ws1Name
	varValueBase := "some variable value "
	varKey := "trigger_name"
	varCategory := "terraform"
	varHCL := false

	return fmt.Sprintf(`

resource "tharsis_apply_module" "tam" {
  workspace_path = "%s"
  module_source  = "%s"
  variables      = [
    {
      value = "%s%d"
      key = "%s"
      category = "%s"
      hcl = %v
    }
  ]
}

	`,
		ws1Path, moduleSource, varValueBase, val, varKey, varCategory, varHCL,
	)
}
