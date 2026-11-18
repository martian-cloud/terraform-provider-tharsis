package provider

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestRunner(t *testing.T) {
	createName := "tra_name"
	createDescription := "this is tra, a test runner"
	createResourcePath := testGroupPath + "/" + createName
	updatedDescription := "this is an updated description for tra, a test runner"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and read back a runner.
			{
				Config: testRunnerConfigurationCreate(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_runner.tra", "name", createName),
					resource.TestCheckResourceAttr("tharsis_runner.tra", "description", createDescription),
					resource.TestCheckResourceAttr("tharsis_runner.tra", "resource_path", createResourcePath),
					resource.TestCheckResourceAttr("tharsis_runner.tra", "disabled", "false"),
					resource.TestCheckResourceAttr("tharsis_runner.tra", "run_untagged_jobs", "false"),
					resource.TestCheckResourceAttr("tharsis_runner.tra", "tags.#", "2"),
					resource.TestCheckTypeSetElemAttr("tharsis_runner.tra", "tags.*", "alpha"),
					resource.TestCheckTypeSetElemAttr("tharsis_runner.tra", "tags.*", "beta"),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_runner.tra", "id"),
					resource.TestCheckResourceAttrSet("tharsis_runner.tra", "group_id"),
				),
			},

			// Import the state.
			{
				ResourceName:      "tharsis_runner.tra",
				ImportState:       true,
				ImportStateVerify: true,
			},

			// Update description, disabled, run_untagged_jobs and tags, then read back.
			{
				Config: testRunnerConfigurationUpdate(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("tharsis_runner.tra", "name", createName),
					resource.TestCheckResourceAttr("tharsis_runner.tra", "description", updatedDescription),
					resource.TestCheckResourceAttr("tharsis_runner.tra", "resource_path", createResourcePath),
					resource.TestCheckResourceAttr("tharsis_runner.tra", "disabled", "true"),
					resource.TestCheckResourceAttr("tharsis_runner.tra", "run_untagged_jobs", "true"),
					resource.TestCheckResourceAttr("tharsis_runner.tra", "tags.#", "1"),
					resource.TestCheckTypeSetElemAttr("tharsis_runner.tra", "tags.*", "gamma"),

					resource.TestCheckResourceAttrSet("tharsis_runner.tra", "id"),
				),
			},

			// Destroy should be covered automatically by TestCase.
		},
	})
}

// TestIsClearingNonEmptySet pins the rule that forces a runner replacement: only a known,
// non-empty tag set planned to become a known, empty one.
func TestIsClearingNonEmptySet(t *testing.T) {
	set := func(v ...string) types.Set { return types.SetValueMust(types.StringType, stringValues(v)) }
	null, unknown := types.SetNull(types.StringType), types.SetUnknown(types.StringType)
	for _, tc := range []struct {
		name        string
		state, plan types.Set
		want        bool
	}{
		{"non-empty to empty", set("a"), set(), true},
		{"non-empty to other non-empty", set("a"), set("b"), false},
		{"empty to empty", set(), set(), false},
		{"empty to non-empty", set(), set("a"), false},
		{"null state", null, set(), false},
		{"unknown state", unknown, set(), false},
		{"null plan", set("a"), null, false},
		{"unknown plan", set("a"), unknown, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := isClearingNonEmptySet(tc.state, tc.plan); got != tc.want {
				t.Fatalf("isClearingNonEmptySet = %v, want %v", got, tc.want)
			}
		})
	}
}

func stringValues(v []string) []attr.Value {
	out := make([]attr.Value, len(v))
	for i, s := range v {
		out[i] = types.StringValue(s)
	}
	return out
}

func testRunnerConfigurationCreate() string {
	createName := "tra_name"
	createDescription := "this is tra, a test runner"

	return fmt.Sprintf(`

%s

resource "tharsis_runner" "tra" {
	name        = "%s"
	description = "%s"
	group_id    = tharsis_group.root-group.id
	tags        = ["alpha", "beta"]
}
	`, createRootGroup(testGroupPath, "this is a test root group"), createName, createDescription)
}

func testRunnerConfigurationUpdate() string {
	createName := "tra_name"
	updatedDescription := "this is an updated description for tra, a test runner"

	return fmt.Sprintf(`

%s

resource "tharsis_runner" "tra" {
	name              = "%s"
	description       = "%s"
	group_id          = tharsis_group.root-group.id
	disabled          = true
	run_untagged_jobs = true
	tags              = ["gamma"]
}
	`, createRootGroup(testGroupPath, "this is a test root group"), createName, updatedDescription)
}

// TestRunnerUnsetOptional is QA's finding on v0.18.0-alpha.6: removing description or tags
// from the configuration kept the old value in Tharsis. It creates a runner with both unset,
// sets them, then removes the description (an in-place update) and then the tags (a
// replacement, since the API can't clear tags through an update). resource.Test re-plans after
// each step, so this also proves there is no perpetual diff.
func TestRunnerUnsetOptional(t *testing.T) {
	const addr = "tharsis_runner.tr_unset"
	var id string
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testRunnerUnsetOptionalConfiguration(""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "description", ""),
					resource.TestCheckResourceAttr(addr, "tags.#", "0"),
					testCaptureAttr(addr, "id", &id),
				),
			},
			{
				Config: testRunnerUnsetOptionalConfiguration("description = \"set\"\n\ttags        = [\"a\"]"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckAttrUnchanged(addr, "id", &id),
					resource.TestCheckResourceAttr(addr, "description", "set"),
					resource.TestCheckResourceAttr(addr, "tags.#", "1"),
				),
			},
			{
				Config: testRunnerUnsetOptionalConfiguration("tags = [\"a\"]"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckAttrUnchanged(addr, "id", &id),
					resource.TestCheckResourceAttr(addr, "description", ""),
					resource.TestCheckResourceAttr(addr, "tags.#", "1"),
				),
			},
			{
				Config: testRunnerUnsetOptionalConfiguration(""),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckAttrChanged(addr, "id", &id),
					resource.TestCheckResourceAttr(addr, "description", ""),
					resource.TestCheckResourceAttr(addr, "tags.#", "0"),
				),
			},
		},
	})
}

func testRunnerUnsetOptionalConfiguration(extra string) string {
	return createRootGroup(testGroupPath, "this is a test root group") + fmt.Sprintf(`

resource "tharsis_runner" "tr_unset" {
	name              = "tr_unset"
	group_id          = tharsis_group.root-group.id
	run_untagged_jobs = true
	%s
}
`, extra)
}

// TestRunnerValidateConfig checks the plan-time rule: no tags (unset or empty) and
// run_untagged_jobs not true is an error; a value unknown until apply is not checked.
func TestRunnerValidateConfig(t *testing.T) {
	ctx := context.Background()
	r := &runnerResource{}
	var sresp fwresource.SchemaResponse
	r.Schema(ctx, fwresource.SchemaRequest{}, &sresp)
	objType := sresp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	tagsType := tftypes.Set{ElementType: tftypes.String}

	set := func(v ...string) tftypes.Value {
		vals := make([]tftypes.Value, len(v))
		for i, s := range v {
			vals[i] = tftypes.NewValue(tftypes.String, s)
		}
		return tftypes.NewValue(tagsType, vals)
	}
	null := tftypes.NewValue(tagsType, nil)
	unknownTags := tftypes.NewValue(tagsType, tftypes.UnknownValue)
	b := func(v bool) tftypes.Value { return tftypes.NewValue(tftypes.Bool, v) }
	nullBool := tftypes.NewValue(tftypes.Bool, nil)
	unknownBool := tftypes.NewValue(tftypes.Bool, tftypes.UnknownValue)

	for _, tc := range []struct {
		name      string
		tags, rut tftypes.Value
		wantError bool
	}{
		{"both unset", null, nullBool, true},
		{"empty tags, run_untagged_jobs false", set(), b(false), true},
		{"empty tags, run_untagged_jobs true", set(), b(true), false},
		{"unset tags, run_untagged_jobs true", null, b(true), false},
		{"a tag, run_untagged_jobs unset", set("a"), nullBool, false},
		{"tags unknown", unknownTags, nullBool, false},
		{"run_untagged_jobs unknown", null, unknownBool, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			vals := map[string]tftypes.Value{}
			for name, at := range objType.AttributeTypes {
				vals[name] = tftypes.NewValue(at, nil)
			}
			vals["name"] = tftypes.NewValue(tftypes.String, "r")
			vals["group_id"] = tftypes.NewValue(tftypes.String, "trn:group:g")
			vals["tags"] = tc.tags
			vals["run_untagged_jobs"] = tc.rut
			req := fwresource.ValidateConfigRequest{Config: tfsdk.Config{Schema: sresp.Schema, Raw: tftypes.NewValue(objType, vals)}}
			resp := &fwresource.ValidateConfigResponse{}
			r.ValidateConfig(ctx, req, resp)
			if got := resp.Diagnostics.HasError(); got != tc.wantError {
				t.Fatalf("HasError = %v, want %v: %v", got, tc.wantError, resp.Diagnostics)
			}
		})
	}
}

// TestRunnerWithoutTagsFailsAtPlan is QA's case: a runner with only name and group_id must fail
// at plan with the provider's message, before anything is created.
func TestRunnerWithoutTagsFailsAtPlan(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: createRootGroup(testGroupPath, "this is a test root group") + `

resource "tharsis_runner" "tr_no_tags" {
	name     = "tr_no_tags"
	group_id = tharsis_group.root-group.id
}
`,
			PlanOnly:    true,
			ExpectError: regexp.MustCompile(`Runner has no way to claim jobs`),
		}},
	})
}
