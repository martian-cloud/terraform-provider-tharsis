package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
