package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestTerraformModule(t *testing.T) {
	createName := "ttm_name"
	createSystem := "aws"
	createGroupPath := testGroupPath
	createRepositoryURL := "http://somewhere.example.invalid/somewhere" // optional
	createPrivate := true                                               // optional

	updateRepositoryURL := "http://somewhere.example.invalid/else"
	updatePrivate := false

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and read back a Terraform module.
			{
				Config: testTerraformModuleConfigurationCreate(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_terraform_module.ttm", "name", createName),
					resource.TestCheckResourceAttr("tharsis_terraform_module.ttm", "system", createSystem),
					resource.TestCheckResourceAttr("tharsis_terraform_module.ttm", "group_path", createGroupPath),
					resource.TestCheckResourceAttr("tharsis_terraform_module.ttm", "repository_url", createRepositoryURL),
					resource.TestCheckResourceAttr("tharsis_terraform_module.ttm", "private", strconv.FormatBool(createPrivate)),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_terraform_module.ttm", "id"),
					resource.TestCheckResourceAttrSet("tharsis_terraform_module.ttm", "resource_path"),
					resource.TestCheckResourceAttrSet("tharsis_terraform_module.ttm", "registry_namespace"),
					resource.TestCheckResourceAttrSet("tharsis_terraform_module.ttm", "last_updated"),
				),
			},

			// Import the state by GID.
			// With no ImportStateId set, the framework reuses the id already in
			// state, which is the GID returned by the API during create.
			{
				ResourceName:      "tharsis_terraform_module.ttm",
				ImportState:       true,
				ImportStateVerify: true,
			},

			// Import the state by TRN.
			{
				ResourceName:      "tharsis_terraform_module.ttm",
				ImportStateId:     "trn:terraform_module:" + createGroupPath + "/" + createName + "/" + createSystem,
				ImportState:       true,
				ImportStateVerify: true,
			},

			// Update and read.
			{
				Config: testTerraformModuleConfigurationUpdate(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_terraform_module.ttm", "name", createName),
					resource.TestCheckResourceAttr("tharsis_terraform_module.ttm", "system", createSystem),
					resource.TestCheckResourceAttr("tharsis_terraform_module.ttm", "group_path", createGroupPath),
					resource.TestCheckResourceAttr("tharsis_terraform_module.ttm", "repository_url", updateRepositoryURL),
					resource.TestCheckResourceAttr("tharsis_terraform_module.ttm", "private", strconv.FormatBool(updatePrivate)),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_terraform_module.ttm", "id"),
					resource.TestCheckResourceAttrSet("tharsis_terraform_module.ttm", "resource_path"),
					resource.TestCheckResourceAttrSet("tharsis_terraform_module.ttm", "registry_namespace"),
					resource.TestCheckResourceAttrSet("tharsis_terraform_module.ttm", "last_updated"),
				),
			},

			// Destroy should be covered automatically by TestCase.

		},
	})
}

func testTerraformModuleConfigurationCreate() string {
	createName := "ttm_name"
	createSystem := "aws"
	createRepositoryURL := "http://somewhere.example.invalid/somewhere"
	createPrivate := true

	return fmt.Sprintf(`

%s

resource "tharsis_terraform_module" "ttm" {
	name = "%s"
	system = "%s"
	group_path = tharsis_group.root-group.full_path
	repository_url = "%s"
	private = "%v"
}
	`, createRootGroup(testGroupPath, "this is a test root group"), createName, createSystem, createRepositoryURL, createPrivate)
}

func testTerraformModuleConfigurationUpdate() string {
	createName := "ttm_name"
	createSystem := "aws"
	updateRepositoryURL := "http://somewhere.example.invalid/else"
	updatePrivate := false

	return fmt.Sprintf(`

%s

resource "tharsis_terraform_module" "ttm" {
	name = "%s"
	system = "%s"
	group_path = tharsis_group.root-group.full_path
	repository_url = "%s"
	private = "%v"
}
	`, createRootGroup(testGroupPath, "this is a test root group"), createName, createSystem, updateRepositoryURL, updatePrivate)
}

// TestTerraformModuleUnsetOptional leaves repository_url and private unset. Tharsis returns "" and false for them, which
// once failed the apply with "Provider produced inconsistent result after apply" and left
// the resource tainted. It then sets them and removes them again: removing a value from the
// configuration must clear it in Tharsis. resource.Test re-plans after each step, so this
// also proves there is no perpetual diff.
func TestTerraformModuleUnsetOptional(t *testing.T) {
	const addr = "tharsis_terraform_module.ttm_unset"
	unset := resource.ComposeAggregateTestCheckFunc(
		resource.TestCheckResourceAttr(addr, "repository_url", ""),
		resource.TestCheckResourceAttr(addr, "private", "false"),
	)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: testTerraformModuleUnsetOptionalConfiguration(""), Check: unset},
			{
				Config: testTerraformModuleUnsetOptionalConfiguration("repository_url = \"https://example.invalid/module\"\n\tprivate        = true"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "repository_url", "https://example.invalid/module"),
					resource.TestCheckResourceAttr(addr, "private", "true"),
				),
			},
			{Config: testTerraformModuleUnsetOptionalConfiguration(""), Check: unset},
		},
	})
}

func testTerraformModuleUnsetOptionalConfiguration(extra string) string {
	return createRootGroup(testGroupPath, "this is a test root group") + fmt.Sprintf(`

resource "tharsis_terraform_module" "ttm_unset" {
	name       = "ttm-unset"
	system     = "aws"
	group_path = tharsis_group.root-group.full_path
	%s
}
`, extra)
}
