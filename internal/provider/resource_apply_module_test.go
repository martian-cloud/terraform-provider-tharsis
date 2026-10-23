package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
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
	varValueBase := "some variable value "
	varKey := "trigger_name"
	varCategory := "terraform"

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
					resource.TestCheckResourceAttr("tharsis_workspace.tw2", "name", ws2Name),
					resource.TestCheckResourceAttr("tharsis_workspace.tw2", "description", ws2Desc),
					resource.TestCheckResourceAttr("tharsis_workspace.tw2", "group_path", testGroupPath),
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
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "resolved_variables.0.value", varValueBase+"1"),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "resolved_variables.0.namespace_path", ""),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "resolved_variables.0.key", varKey),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "resolved_variables.0.category", varCategory),
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
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "resolved_variables.0.value", varValueBase+"1"),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "resolved_variables.0.namespace_path", ""),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "resolved_variables.0.key", varKey),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "resolved_variables.0.category", varCategory),
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
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "resolved_variables.0.value", varValueBase+"2"),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "resolved_variables.0.namespace_path", ""),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "resolved_variables.0.key", varKey),
					resource.TestCheckResourceAttr("tharsis_apply_module.tam", "resolved_variables.0.category", varCategory),
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

	return fmt.Sprintf(`

%s

resource "tharsis_workspace" "tw1" {
	name                 = "%s"
	description          = "%s"
	group_path           = tharsis_group.root-group.full_path
	prevent_destroy_plan = false
}

resource "tharsis_workspace" "tw2" {
	name                 = "%s"
	description          = "%s"
	group_path           = tharsis_group.root-group.full_path
	prevent_destroy_plan = false
}

	`, createRootGroup(testGroupPath, "this is a test root group"),
		ws1Name, ws1Desc, ws2Name, ws2Desc,
	)
}

func testDoApplyCreateRun(val int) string {
	ws1Name := "workspace-1"
	ws1Path := testGroupPath + "/" + ws1Name
	varValueBase := "some variable value "
	varKey := "trigger_name"
	varCategory := "terraform"

	return fmt.Sprintf(`

resource "tharsis_apply_module" "tam" {
  workspace_path = "%s"
  module_source  = "%s"
  variables      = [
    {
      value = "%s%d"
      key = "%s"
      category = "%s"
    }
  ]
}

	`,
		ws1Path, moduleSource, varValueBase, val, varKey, varCategory,
	)
}

// TestApplyModuleImportStateIsRejected verifies that importing a
// tharsis_apply_module fails with an explanation rather than appearing to
// succeed or producing Terraform's generic unsupported-import message.
//
// A tharsis_apply_module has no counterpart in the Tharsis API, so there is no
// stored object an import could read back.
func TestApplyModuleImportStateIsRejected(t *testing.T) {
	tests := []struct {
		name     string
		importID string
	}{
		{name: "a workspace path", importID: "my-group/my-workspace"},
		{name: "a manufactured UUID", importID: "f6b2c5d1-9a84-4c3e-8f17-0d5b6e9a2c41"},
		{name: "an empty string", importID: ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			underTest := &applyModuleResource{}
			resp := &fwresource.ImportStateResponse{State: newTestImportState()}

			underTest.ImportState(context.Background(),
				fwresource.ImportStateRequest{ID: test.importID}, resp)

			if !resp.Diagnostics.HasError() {
				t.Fatalf("expected an error diagnostic for import ID %q, got none", test.importID)
			}

			summary := resp.Diagnostics.Errors()[0].Summary()
			if !strings.Contains(summary, "cannot be imported") {
				t.Errorf("diagnostic summary %q does not explain that import is unsupported", summary)
			}

			// The detail should point the user at a usable alternative.
			detail := resp.Diagnostics.Errors()[0].Detail()
			if !strings.Contains(detail, "tharsis_workspace") {
				t.Errorf("diagnostic detail %q does not suggest an alternative", detail)
			}
		})
	}
}

// newTestImportState builds an empty single-attribute state suitable for
// exercising an ImportState implementation in a unit test.
func newTestImportState() tfsdk.State {
	return tfsdk.State{
		Schema: schema.Schema{
			Attributes: map[string]schema.Attribute{
				"id": schema.StringAttribute{Computed: true},
			},
		},
		Raw: tftypes.NewValue(
			tftypes.Object{AttributeTypes: map[string]tftypes.Type{"id": tftypes.String}},
			map[string]tftypes.Value{"id": tftypes.NewValue(tftypes.String, nil)},
		),
	}
}
