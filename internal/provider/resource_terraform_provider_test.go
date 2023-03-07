package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestTerraformProvider(t *testing.T) {
	createName := "ttp_name"
	createGroupPath := testGroupPath
	createResourcePath := createGroupPath + "/" + createName
	createRegistryNamespace := testGroupPath
	createRepositoryURL := "https://invalid.example/some/repository/url"
	createPrivate := true
	updateRepositoryURL := "https://invalid.example/updated/repository/url"
	updatePrivate := false

	resource.Test(t, resource.TestCase{

		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{

			// Create and read back a Terraform provider.
			{
				Config: testTerraformProviderConfigurationCreate(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_terraform_provider.ttp", "name", createName),
					resource.TestCheckResourceAttr("tharsis_terraform_provider.ttp", "group_path", createGroupPath),
					resource.TestCheckResourceAttr("tharsis_terraform_provider.ttp", "resource_path", createResourcePath),
					resource.TestCheckResourceAttr("tharsis_terraform_provider.ttp", "registry_namespace", createRegistryNamespace),
					resource.TestCheckResourceAttr("tharsis_terraform_provider.ttp", "repository_url", createRepositoryURL),
					resource.TestCheckResourceAttr("tharsis_terraform_provider.ttp", "private", strconv.FormatBool(createPrivate)),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_terraform_provider.ttp", "id"),
					resource.TestCheckResourceAttrSet("tharsis_terraform_provider.ttp", "last_updated"),
				),
			},

			// Import the state.
			{
				ResourceName:      "tharsis_terraform_provider.ttp",
				ImportState:       true,
				ImportStateVerify: true,
			},

			// Update and read.
			{
				Config: testTerraformProviderConfigurationUpdate(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_terraform_provider.ttp", "name", createName),
					resource.TestCheckResourceAttr("tharsis_terraform_provider.ttp", "group_path", createGroupPath),
					resource.TestCheckResourceAttr("tharsis_terraform_provider.ttp", "resource_path", createResourcePath),
					resource.TestCheckResourceAttr("tharsis_terraform_provider.ttp", "registry_namespace", createRegistryNamespace),
					resource.TestCheckResourceAttr("tharsis_terraform_provider.ttp", "repository_url", updateRepositoryURL),
					resource.TestCheckResourceAttr("tharsis_terraform_provider.ttp", "private", strconv.FormatBool(updatePrivate)),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_terraform_provider.ttp", "id"),
					resource.TestCheckResourceAttrSet("tharsis_terraform_provider.ttp", "last_updated"),
				),
			},

			// Destroy should be covered automatically by TestCase.

		},
	})
}

func testTerraformProviderConfigurationCreate() string {
	createName := "ttp_name"
	createRepositoryURL := "https://invalid.example/some/repository/url"
	createPrivate := true

	return fmt.Sprintf(`

%s

resource "tharsis_terraform_provider" "ttp" {
	name = "%s"
	group_path = tharsis_group.root-group.full_path
	repository_url = "%s"
	private = %v
}
	`, createRootGroup(), createName, createRepositoryURL, createPrivate)
}

func testTerraformProviderConfigurationUpdate() string {
	createName := "ttp_name"
	updateRepositoryURL := "https://invalid.example/updated/repository/url"
	updatePrivate := false

	return fmt.Sprintf(`

%s

resource "tharsis_terraform_provider" "ttp" {
	name = "%s"
	group_path = tharsis_group.root-group.full_path
	repository_url = "%s"
	private = %v
}
`, createRootGroup(), createName, updateRepositoryURL, updatePrivate)
}

// The End.
