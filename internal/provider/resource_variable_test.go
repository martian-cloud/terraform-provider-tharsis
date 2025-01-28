package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestVariable(t *testing.T) {
	createNamespacePath := testGroupPath
	createCategory := "terraform"
	createKey := "first-key"
	createValue := "first-value"
	updateKey := "updated-key"
	updateValue := "updated-value"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and read back a variable.
			{
				Config: testVariableConfigurationCreate(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_variable.tnv", "namespace_path", createNamespacePath),
					resource.TestCheckResourceAttr("tharsis_variable.tnv", "category", createCategory),
					resource.TestCheckResourceAttr("tharsis_variable.tnv", "key", createKey),
					resource.TestCheckResourceAttr("tharsis_variable.tnv", "value", createValue),

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
					resource.TestCheckResourceAttr("tharsis_variable.tnv", "namespace_path", createNamespacePath),
					resource.TestCheckResourceAttr("tharsis_variable.tnv", "category", createCategory),
					resource.TestCheckResourceAttr("tharsis_variable.tnv", "key", updateKey),
					resource.TestCheckResourceAttr("tharsis_variable.tnv", "value", updateValue),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_variable.tnv", "id"),
				),
			},

			// Destroy should be covered automatically by TestCase.

		},
	})
}

func testVariableConfigurationCreate() string {
	createCategory := "terraform"
	createKey := "first-key"
	createValue := "first-value"

	return fmt.Sprintf(`

%s

resource "tharsis_variable" "tnv" {
	namespace_path = tharsis_group.root-group.full_path
	category = "%s"
	key = "%s"
	value = "%s"
}
	`, createRootGroup(testGroupPath, "this is a test root group"), createCategory, createKey, createValue)
}

func testVariableConfigurationUpdate() string {
	createCategory := "terraform"
	updateKey := "updated-key"
	updateValue := "updated-value"

	return fmt.Sprintf(`

%s

resource "tharsis_variable" "tnv" {
	namespace_path = tharsis_group.root-group.full_path
	category = "%s"
	key = "%s"
	value = "%s"
}
	`, createRootGroup(testGroupPath, "this is a test root group"), createCategory, updateKey, updateValue)
}
