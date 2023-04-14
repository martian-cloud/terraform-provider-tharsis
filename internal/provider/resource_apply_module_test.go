package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	moduleSource = "registry.terraform.io/vancluever/module/null"
	// FIXME: Switch to this:
	// moduleSource := "registry.terraform.io/martian-cloud/module/terraform-null-module"
)

func TestApplyModule(t *testing.T) {
	ws1Name := "workspace-1"
	ws1Desc := "this is workspace 1"
	ws1Path := testGroupPath + "/" + ws1Name
	ws2Name := "workspace-2"
	ws2Desc := "this is workspace 2"
	wsPreventDestroyPlan := false
	varValueBase := "some variable value "
	varKey := "a-variable-name"
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
				Config: testDoApplyCreateRun(1),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "workspace_path", ws1Path),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "module_source", moduleSource),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "variables.0.value", varValueBase+"1"),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "variables.0.key", varKey),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "variables.0.category", varCategory),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "variables.0.hcl", strconv.FormatBool(varHCL)),
				),
			},

			// Repeat the apply/create run with no changes.
			{
				Config: testDoApplyCreateRun(1),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "workspace_path", ws1Path),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "module_source", moduleSource),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "variables.0.value", varValueBase+"1"),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "variables.0.key", varKey),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "variables.0.category", varCategory),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "variables.0.hcl", strconv.FormatBool(varHCL)),
				),
			},

			// Do an apply/create run with changes to the variable's value.
			{
				Config: testDoApplyCreateRun(2),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "workspace_path", ws1Path),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "module_source", moduleSource),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "variables.0.value", varValueBase+"2"),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "variables.0.key", varKey),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "variables.0.category", varCategory),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "variables.0.hcl", strconv.FormatBool(varHCL)),
				),
			},

			// Do a destroy/delete run.
			{
				Config:  testDoApplyCreateRun(2),
				Destroy: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckNoResourceAttr("tharsis_apply_module.tam", "workspace_path"),
					resource.TestCheckNoResourceAttr("tharsis_apply_module.tam", "module_source"),
				),
			},

			// Destroy should be covered automatically by TestCase.

		},
	})
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

	`, createRootGroup(),
		ws1Name, ws1Desc, wsPreventDestroyPlan,
		ws2Name, ws2Desc, wsPreventDestroyPlan,
	)
}

func testDoApplyCreateRun(val int) string {
	ws1Name := "workspace-1"
	ws1Path := testGroupPath + "/" + ws1Name
	varValueBase := "some variable value "
	varKey := "a-variable-name"
	varCategory := "terraform"
	varHCL := false

	return fmt.Sprintf(`

%s

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

	`, testApplyModuleConfigurationCreate(),
		ws1Path, moduleSource, varValueBase, val, varKey, varCategory, varHCL,
	)
}

// The End.
