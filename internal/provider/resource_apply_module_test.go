package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	runpkg "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/run"
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

// TestParseWaitTimeout covers the wait_timeout attribute parsing: an unset, unknown or empty value
// falls back to the follower's default, a valid Go duration is parsed, and a non-positive or
// unparseable value is an error.
func TestParseWaitTimeout(t *testing.T) {
	tests := []struct {
		name    string
		input   types.String
		want    time.Duration
		wantErr bool
	}{
		{name: "null defaults", input: types.StringNull(), want: runpkg.DefaultWaitTimeout},
		{name: "unknown defaults", input: types.StringUnknown(), want: runpkg.DefaultWaitTimeout},
		{name: "empty defaults", input: types.StringValue(""), want: runpkg.DefaultWaitTimeout},
		{name: "valid minutes", input: types.StringValue("30m"), want: 30 * time.Minute},
		{name: "valid hours", input: types.StringValue("2h"), want: 2 * time.Hour},
		{name: "valid composite", input: types.StringValue("1h30m"), want: 90 * time.Minute},
		{name: "unparseable errors", input: types.StringValue("nope"), wantErr: true},
		{name: "zero errors", input: types.StringValue("0s"), wantErr: true},
		{name: "negative errors", input: types.StringValue("-5m"), wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseWaitTimeout(test.input)
			if test.wantErr {
				if err == nil {
					t.Fatalf("parseWaitTimeout(%q) expected an error, got none", test.input.ValueString())
				}
				return
			}
			if err != nil {
				t.Fatalf("parseWaitTimeout(%q) unexpected error: %v", test.input.ValueString(), err)
			}
			if got != test.want {
				t.Errorf("parseWaitTimeout(%q) = %s, want %s", test.input.ValueString(), got, test.want)
			}
		})
	}
}

// TestWaitTimeoutSchemaIsBackwardsCompatible asserts that wait_timeout is an optional attribute and
// is neither required nor computed, so adding it does not break existing configurations or state:
// configs that omit it keep working, and it produces no "known after apply" churn on upgrade.
func TestWaitTimeoutSchemaIsBackwardsCompatible(t *testing.T) {
	var resp fwresource.SchemaResponse
	(&applyModuleResource{}).Schema(context.Background(), fwresource.SchemaRequest{}, &resp)

	attr, ok := resp.Schema.Attributes["wait_timeout"]
	if !ok {
		t.Fatal("wait_timeout attribute is missing from the schema")
	}
	if !attr.IsOptional() {
		t.Error("wait_timeout must be optional")
	}
	if attr.IsRequired() {
		t.Error("wait_timeout must not be required (would break existing configs)")
	}
	if attr.IsComputed() {
		t.Error("wait_timeout must not be computed (would cause plan churn on upgrade)")
	}
}

func TestRunFailureMessage(t *testing.T) {
	planError := "Error: Invalid reference"
	applyError := "Error: creating bucket: AccessDenied"

	tests := []struct {
		name    string
		outcome *runpkg.Outcome
		want    string
	}{
		{
			name: "plan error",
			outcome: &runpkg.Outcome{
				Run: &pb.Run{
					Status: pb.RunStatus_ERRORED,
					Plan:   &pb.Plan{ErrorMessage: &planError},
					Apply:  &pb.Apply{},
				},
				Stage: runpkg.StageRef{Name: "Plan", Kind: runpkg.StagePlan},
			},
			want: `Run "run-1" ended with status ERRORED: plan failed: Error: Invalid reference`,
		},
		{
			name: "apply error",
			outcome: &runpkg.Outcome{
				Run: &pb.Run{
					Status: pb.RunStatus_ERRORED,
					Plan:   &pb.Plan{},
					Apply:  &pb.Apply{ErrorMessage: &applyError},
				},
				Stage: runpkg.StageRef{Name: "Apply", Kind: runpkg.StageApply},
			},
			want: `Run "run-1" ended with status ERRORED: apply failed: Error: creating bucket: AccessDenied`,
		},
		{
			name: "policy stage failure without a plan or apply error",
			outcome: &runpkg.Outcome{
				Run:   &pb.Run{Status: pb.RunStatus_ERRORED, Plan: &pb.Plan{}},
				Stage: runpkg.StageRef{Name: "Post-plan stage", Kind: runpkg.StageTask},
			},
			want: `Run "run-1" ended with status ERRORED during Post-plan stage.`,
		},
		{
			name: "no plan, apply or stage",
			outcome: &runpkg.Outcome{
				Run: &pb.Run{Status: pb.RunStatus_CANCELED},
			},
			want: `Run "run-1" ended with status CANCELED.`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := runFailureMessage("run-1", test.outcome); got != test.want {
				t.Errorf("runFailureMessage() = %q, want %q", got, test.want)
			}
		})
	}
}
