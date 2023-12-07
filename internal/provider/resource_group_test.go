package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestRootGroup(t *testing.T) {
	createName := "trg_name"
	createDescription := "this is root-group, a test root group"
	updatedDescription := "this is an updated description for root-group, a test root group"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and read back a root group.
			{
				Config: createRootGroup(createName, createDescription),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_group.root-group", "name", createName),
					resource.TestCheckResourceAttr("tharsis_group.root-group", "description", createDescription),
					resource.TestCheckResourceAttr("tharsis_group.root-group", "full_path", createName),

					// Verify that the parent path is _NOT_ set.
					resource.TestCheckNoResourceAttr("tharsis_group.root-group", "parent_path"),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_group.root-group", "id"),
					resource.TestCheckResourceAttrSet("tharsis_group.root-group", "last_updated"),
				),
			},

			// Import the state.
			{
				ResourceName:      "tharsis_group.root-group",
				ImportStateId:     createName,
				ImportState:       true,
				ImportStateVerify: true,
			},

			// Update and read.
			{
				Config: createRootGroup(createName, updatedDescription),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_group.root-group", "name", createName),
					resource.TestCheckResourceAttr("tharsis_group.root-group", "description", updatedDescription),
					resource.TestCheckResourceAttr("tharsis_group.root-group", "full_path", createName),

					// Verify that the parent path is _NOT_ set.
					resource.TestCheckNoResourceAttr("tharsis_group.root-group", "parent_path"),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_group.root-group", "id"),
					resource.TestCheckResourceAttrSet("tharsis_group.root-group", "last_updated"),
				),
			},

			// Destroy should be covered automatically by TestCase.

		},
	})
}

func TestGroupNoDescription(t *testing.T) {
	createName := "trg_name"
	computedDescription := ""
	updatedDescription := "this is an updated description for root-group, a test root group"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and read back a root group.
			{
				Config: createRootGroupOptionalDescription(createName, nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_group.root-group", "name", createName),
					resource.TestCheckResourceAttr("tharsis_group.root-group", "description", computedDescription),
					resource.TestCheckResourceAttr("tharsis_group.root-group", "full_path", createName),

					// Verify that the parent path is _NOT_ set.
					resource.TestCheckNoResourceAttr("tharsis_group.root-group", "parent_path"),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_group.root-group", "id"),
					resource.TestCheckResourceAttrSet("tharsis_group.root-group", "last_updated"),
				),
			},

			// Import the state.
			{
				ResourceName:      "tharsis_group.root-group",
				ImportStateId:     createName,
				ImportState:       true,
				ImportStateVerify: true,
			},

			// Update and read.
			{
				Config: createRootGroup(createName, updatedDescription),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_group.root-group", "name", createName),
					resource.TestCheckResourceAttr("tharsis_group.root-group", "description", updatedDescription),
					resource.TestCheckResourceAttr("tharsis_group.root-group", "full_path", createName),

					// Verify that the parent path is _NOT_ set.
					resource.TestCheckNoResourceAttr("tharsis_group.root-group", "parent_path"),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_group.root-group", "id"),
					resource.TestCheckResourceAttrSet("tharsis_group.root-group", "last_updated"),
				),
			},

			// Destroy should be covered automatically by TestCase.

		},
	})
}

func TestNestedGroup(t *testing.T) {
	createName := "tng_name"
	createDescription := "this is nested-group, a test nested group"
	createParentPath := testGroupPath
	createFullPath := createParentPath + "/" + createName
	updatedDescription := "this is an updated description for nested-group, a test nested group"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and read back a nested group.
			{
				Config: testGroupNestedConfiguration(createName, createDescription),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_group.nested-group", "name", createName),
					resource.TestCheckResourceAttr("tharsis_group.nested-group", "description", createDescription),
					resource.TestCheckResourceAttr("tharsis_group.nested-group", "parent_path", createParentPath),
					resource.TestCheckResourceAttr("tharsis_group.nested-group", "full_path", createFullPath),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_group.nested-group", "id"),
					resource.TestCheckResourceAttrSet("tharsis_group.nested-group", "last_updated"),
				),
			},

			// Import the state.
			{
				ResourceName:      "tharsis_group.nested-group",
				ImportStateId:     createFullPath,
				ImportState:       true,
				ImportStateVerify: true,
			},

			// Update and read.
			{
				Config: testGroupNestedConfiguration(createName, updatedDescription),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_group.nested-group", "name", createName),
					resource.TestCheckResourceAttr("tharsis_group.nested-group", "description", updatedDescription),
					resource.TestCheckResourceAttr("tharsis_group.nested-group", "parent_path", createParentPath),
					resource.TestCheckResourceAttr("tharsis_group.nested-group", "full_path", createFullPath),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_group.nested-group", "id"),
					resource.TestCheckResourceAttrSet("tharsis_group.nested-group", "last_updated"),
				),
			},

			// Destroy should be covered automatically by TestCase.

		},
	})
}

func createRootGroup(name, description string) string {
	return createRootGroupOptionalDescription(name, &description)
}

func createRootGroupOptionalDescription(name string, description *string) string {
	fmtDescription := ""
	if description != nil {
		fmtDescription = fmt.Sprintf("\n	description = \"%s\"", *description)
	}

	return fmt.Sprintf(`

resource "tharsis_group" "root-group" {
	name = "%s"%s
}
	`, name, fmtDescription)
}

func testGroupNestedConfiguration(name, description string) string {
	return fmt.Sprintf(`

%s

resource "tharsis_group" "nested-group" {
	name = "%s"
	description = "%s"
	parent_path = tharsis_group.root-group.full_path
}
	`, createRootGroup(testGroupPath, "this is a test root group"), name, description)
}
