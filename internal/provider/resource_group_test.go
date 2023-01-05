package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestRootGroup(t *testing.T) {
	createName := "trg_name"
	createDescription := "this is trg, a test root group"
	updatedDescription := "this is an updated description for trg, a test root group"

	resource.Test(t, resource.TestCase{

		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{

			// Create and read back a root group.
			{
				Config: testGroupRootConfigurationCreate(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_group.trg", "name", createName),
					resource.TestCheckResourceAttr("tharsis_group.trg", "description", createDescription),
					resource.TestCheckResourceAttr("tharsis_group.trg", "full_path", createName),

					// Verify that the parent path is _NOT_ set.
					resource.TestCheckNoResourceAttr("tharsis_group.trg", "parent_path"),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_group.trg", "id"),
					resource.TestCheckResourceAttrSet("tharsis_group.trg", "last_updated"),
				),
			},

			// Import the state.
			{
				ResourceName:      "tharsis_group.trg",
				ImportStateId:     createName,
				ImportState:       true,
				ImportStateVerify: true,
			},

			// Update and read.
			{
				Config: testGroupRootConfigurationUpdate(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_group.trg", "name", createName),
					resource.TestCheckResourceAttr("tharsis_group.trg", "description", updatedDescription),
					resource.TestCheckResourceAttr("tharsis_group.trg", "full_path", createName),

					// Verify that the parent path is _NOT_ set.
					resource.TestCheckNoResourceAttr("tharsis_group.trg", "parent_path"),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_group.trg", "id"),
					resource.TestCheckResourceAttrSet("tharsis_group.trg", "last_updated"),
				),
			},

			// Destroy should be covered automatically by TestCase.

		},
	})
}

func TestNestedGroup(t *testing.T) {
	createName := "tng_name"
	createDescription := "this is tng, a test nested group"
	createParentPath := testGroupPath
	createFullPath := createParentPath + "/" + createName
	updatedDescription := "this is an updated description for tng, a test nested group"

	resource.Test(t, resource.TestCase{

		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{

			// Create and read back a nested group.
			{
				Config: testGroupNestedConfigurationCreate(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_group.tng", "name", createName),
					resource.TestCheckResourceAttr("tharsis_group.tng", "description", createDescription),
					resource.TestCheckResourceAttr("tharsis_group.tng", "parent_path", createParentPath),
					resource.TestCheckResourceAttr("tharsis_group.tng", "full_path", createFullPath),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_group.tng", "id"),
					resource.TestCheckResourceAttrSet("tharsis_group.tng", "last_updated"),
				),
			},

			// Import the state.
			{
				ResourceName:      "tharsis_group.tng",
				ImportStateId:     createFullPath,
				ImportState:       true,
				ImportStateVerify: true,
			},

			// Update and read.
			{
				Config: testGroupNestedConfigurationUpdate(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_group.tng", "name", createName),
					resource.TestCheckResourceAttr("tharsis_group.tng", "description", updatedDescription),
					resource.TestCheckResourceAttr("tharsis_group.tng", "parent_path", createParentPath),
					resource.TestCheckResourceAttr("tharsis_group.tng", "full_path", createFullPath),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_group.tng", "id"),
					resource.TestCheckResourceAttrSet("tharsis_group.tng", "last_updated"),
				),
			},

			// Destroy should be covered automatically by TestCase.

		},
	})
}

func testGroupRootConfigurationCreate() string {
	createName := "trg_name"
	createDescription := "this is trg, a test root group"

	return fmt.Sprintf(`

resource "tharsis_group" "trg" {
	name = "%s"
	description = "%s"
}
	`, createName, createDescription)
}

func testGroupRootConfigurationUpdate() string {
	createName := "trg_name"
	updatedDescription := "this is an updated description for trg, a test root group"

	return fmt.Sprintf(`

	resource "tharsis_group" "trg" {
		name = "%s"
		description = "%s"
	}
		`, createName, updatedDescription)
}

func testGroupNestedConfigurationCreate() string {
	createName := "tng_name"
	createDescription := "this is tng, a test nested group"

	return fmt.Sprintf(`

%s

resource "tharsis_group" "tng" {
	name = "%s"
	description = "%s"
	parent_path = tharsis_group.root-group.full_path
}
	`, createRootGroup(), createName, createDescription)
}

func testGroupNestedConfigurationUpdate() string {
	createName := "tng_name"
	updatedDescription := "this is an updated description for tng, a test nested group"

	return fmt.Sprintf(`

	resource "tharsis_group" "tng" {
		name = "%s"
		description = "%s"
		parent_path = tharsis_group.root-group.full_path
	}
		`, createName, updatedDescription)
}

func createRootGroup() string {
	createRootGroupPath := testGroupPath
	createRootGroupDescription := "this is a test root group"

	return fmt.Sprintf(`

resource "tharsis_group" "root-group" {
		name = "%s"
		description = "%s"
}

	`, createRootGroupPath, createRootGroupDescription)
}

// The End.
