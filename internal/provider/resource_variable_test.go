package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

type testVariableConstants struct {
	createNamespacePath string
	createCategory      string
	createHCL           bool
	createKey           string
	createValue         string
	updateHCL           bool
	updateKey           string
	updateValue         string
}

func TestVariable(t *testing.T) {
	c := buildTestVariableConstants()

	resource.Test(t, resource.TestCase{

		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{

			// Create and read back a variable.
			{
				Config: testVariableConfigurationCreate(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_variable.tnv", "namespace_path", c.createNamespacePath),
					resource.TestCheckResourceAttr("tharsis_variable.tnv", "category", c.createCategory),
					resource.TestCheckResourceAttr("tharsis_variable.tnv", "hcl", strconv.FormatBool(c.createHCL)),
					resource.TestCheckResourceAttr("tharsis_variable.tnv", "key", c.createKey),
					resource.TestCheckResourceAttr("tharsis_variable.tnv", "value", c.createValue),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_variable.tnv", "id"),
				),
			},

			// Import the state.
			{
				ResourceName:      "tharsis_variable.tnv",
				ImportState:       true,
				ImportStateVerify: true,
			},

			// Update and read.
			{
				Config: testVariableConfigurationUpdate(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_variable.tnv", "namespace_path", c.createNamespacePath),
					resource.TestCheckResourceAttr("tharsis_variable.tnv", "category", c.createCategory),
					resource.TestCheckResourceAttr("tharsis_variable.tnv", "hcl", strconv.FormatBool(c.updateHCL)),
					resource.TestCheckResourceAttr("tharsis_variable.tnv", "key", c.updateKey),
					resource.TestCheckResourceAttr("tharsis_variable.tnv", "value", c.updateValue),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_variable.tnv", "id"),
				),
			},

			// Destroy should be covered automatically by TestCase.

		},
	})
}

func testVariableConfigurationCreate() string {
	c := buildTestVariableConstants()
	return fmt.Sprintf(`

resource "tharsis_variable" "tnv" {
	namespace_path = "%s"
	category = "%s"
	hcl = "%v"
	key = "%s"
	value = "%s"
}
	`, c.createNamespacePath, c.createCategory, c.createHCL, c.createKey, c.createValue)
}

func testVariableConfigurationUpdate() string {
	c := buildTestVariableConstants()
	return fmt.Sprintf(`

resource "tharsis_variable" "tnv" {
	namespace_path = "%s"
	category = "%s"
	hcl = "%v"
	key = "%s"
	value = "%s"
}
	`, c.createNamespacePath, c.createCategory, c.updateHCL, c.updateKey, c.updateValue)
}

func buildTestVariableConstants() *testVariableConstants {
	return &testVariableConstants{
		createNamespacePath: testGroupPath,
		createCategory:      "terraform",
		createHCL:           true,
		createKey:           "first-key",
		createValue:         "first-value",
		updateHCL:           false,
		updateKey:           "updated-key",
		updateValue:         "updated-value",
	}
}

// The End.
