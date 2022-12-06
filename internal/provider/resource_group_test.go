package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// Test only a nested group, because top-level group creation is privileged.
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

func testGroupNestedConfigurationCreate() string {
	createName := "tng_name"
	createDescription := "this is tng, a test nested group"
	createParentPath := testGroupPath

	return fmt.Sprintf(`

resource "tharsis_group" "tng" {
	name = "%s"
	description = "%s"
	parent_path = "%s"
}
	`, createName, createDescription, createParentPath)
}

func testGroupNestedConfigurationUpdate() string {
	createName := "tng_name"
	updatedDescription := "this is an updated description for tng, a test nested group"
	createParentPath := testGroupPath

	return fmt.Sprintf(`

	resource "tharsis_group" "tng" {
		name = "%s"
		description = "%s"
		parent_path = "%s"
	}
		`, createName, updatedDescription, createParentPath)
}

// The End.
