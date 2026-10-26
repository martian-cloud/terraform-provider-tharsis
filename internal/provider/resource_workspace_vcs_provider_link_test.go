package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// Environment variables that supply the VCS provider this test links to. All
// three must be set for the test to run.
const (
	// testVCSProviderIDEnvVar is a VCS provider whose OAuth authorization has
	// already been completed, given as an ID or a TRN.
	testVCSProviderIDEnvVar = "THARSIS_TEST_VCS_PROVIDER_ID"

	// testVCSProviderGroupPathEnvVar is the path of the group that VCS provider
	// belongs to. The test workspace is created inside it.
	testVCSProviderGroupPathEnvVar = "THARSIS_TEST_VCS_PROVIDER_GROUP_PATH"

	// testVCSRepositoryPathEnvVar is a repository the VCS provider's credentials
	// can read, in "<namespace>/<repository>" form.
	testVCSRepositoryPathEnvVar = "THARSIS_TEST_VCS_REPOSITORY_PATH"
)

// TestWorkspaceVCSProviderLink tests creation, reading, importing, updating, and
// deletion of a workspace VCS provider link resource.
//
// The test takes an existing VCS provider from the environment rather than
// creating one. Tharsis only links a workspace to a provider whose OAuth
// authorization has been completed through a browser, and it then calls the VCS
// host to refresh the token and read the repository, so the provider and the
// repository both have to be real.
//
// The referenced provider must have auto_create_webhooks set to false. When it
// is true, linking creates a webhook on the repository and unlinking calls the
// host to remove it, which makes the test depend on write access to the
// repository.
//
// Two further details shape the configuration below:
//
// The workspace is created inside the VCS provider's group, because Tharsis
// requires the workspace path to sit within the provider's group hierarchy.
//
// The configuration supplies workspace_path rather than workspace_id, because
// the provider always populates workspace_path from the link's TRN on read.
// Omitting it from the configuration would leave Terraform comparing a null
// configuration value against a populated result and rejecting the apply.
func TestWorkspaceVCSProviderLink(t *testing.T) {
	vcsProviderID := os.Getenv(testVCSProviderIDEnvVar)
	vcsGroupPath := os.Getenv(testVCSProviderGroupPathEnvVar)
	repositoryPath := os.Getenv(testVCSRepositoryPathEnvVar)

	if vcsProviderID == "" || vcsGroupPath == "" || repositoryPath == "" {
		t.Skipf("set %s, %s, and %s to run this test against a VCS provider "+
			"whose OAuth authorization has already been completed",
			testVCSProviderIDEnvVar, testVCSProviderGroupPathEnvVar, testVCSRepositoryPathEnvVar)
	}

	createWorkspaceName := "twvpl-workspace"
	createWorkspacePath := vcsGroupPath + "/" + createWorkspaceName
	createModuleDirectory := "modules/first"
	createBranch := "main"
	createAutoSpeculativePlan := false
	createWebhookDisabled := true

	updateModuleDirectory := "modules/second"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and read back a workspace VCS provider link.
			{
				Config: testWorkspaceVCSProviderLinkConfiguration(
					vcsProviderID, vcsGroupPath, createWorkspaceName,
					repositoryPath, createModuleDirectory, createBranch),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_workspace_vcs_provider_link.twvpl",
						"repository_path", repositoryPath),
					resource.TestCheckResourceAttr("tharsis_workspace_vcs_provider_link.twvpl",
						"module_directory", createModuleDirectory),
					resource.TestCheckResourceAttr("tharsis_workspace_vcs_provider_link.twvpl",
						"vcs_provider_id", vcsProviderID),
					resource.TestCheckResourceAttr("tharsis_workspace_vcs_provider_link.twvpl",
						"auto_speculative_plan", strconv.FormatBool(createAutoSpeculativePlan)),
					resource.TestCheckResourceAttr("tharsis_workspace_vcs_provider_link.twvpl",
						"webhook_disabled", strconv.FormatBool(createWebhookDisabled)),
					resource.TestCheckResourceAttr("tharsis_workspace_vcs_provider_link.twvpl",
						"workspace_path", createWorkspacePath),

					// Verify the link points at the workspace it was given.
					resource.TestCheckResourceAttrPair("tharsis_workspace.twvpl_ws", "id",
						"tharsis_workspace_vcs_provider_link.twvpl", "workspace_id"),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_workspace_vcs_provider_link.twvpl", "id"),
					resource.TestCheckResourceAttrSet("tharsis_workspace_vcs_provider_link.twvpl", "last_updated"),
				),
			},

			// Import the state by GID.
			// With no ImportStateId set, the framework reuses the id already in
			// state, which is the GID returned by the API during create.
			//
			// A link has no fully human-readable identifier. Its TRN is
			// "<workspace_path>/<link_GID>", whose final segment is itself a
			// GID, so the plain GID is the practical import form.
			{
				ResourceName:      "tharsis_workspace_vcs_provider_link.twvpl",
				ImportState:       true,
				ImportStateVerify: true,
			},

			// Update and read.
			{
				Config: testWorkspaceVCSProviderLinkConfiguration(
					vcsProviderID, vcsGroupPath, createWorkspaceName,
					repositoryPath, updateModuleDirectory, createBranch),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_workspace_vcs_provider_link.twvpl",
						"repository_path", repositoryPath),
					resource.TestCheckResourceAttr("tharsis_workspace_vcs_provider_link.twvpl",
						"module_directory", updateModuleDirectory),
					resource.TestCheckResourceAttr("tharsis_workspace_vcs_provider_link.twvpl",
						"workspace_path", createWorkspacePath),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_workspace_vcs_provider_link.twvpl", "id"),
					resource.TestCheckResourceAttrSet("tharsis_workspace_vcs_provider_link.twvpl", "last_updated"),
				),
			},

			// Destroy should be covered automatically by TestCase.
		},
	})
}

// testWorkspaceVCSProviderLinkConfiguration builds a workspace inside the VCS
// provider's group and links it to that provider.
//
// The VCS provider is referenced by ID rather than declared, so Terraform never
// manages it and tearing the test down leaves it untouched.
func testWorkspaceVCSProviderLinkConfiguration(
	vcsProviderID, vcsGroupPath, workspaceName, repositoryPath, moduleDirectory, branch string,
) string {
	return fmt.Sprintf(`

resource "tharsis_workspace" "twvpl_ws" {
	name = "%s"
	description = "this is twvpl_ws, a test workspace for a VCS provider link"
	group_path = "%s"
}

resource "tharsis_workspace_vcs_provider_link" "twvpl" {
	workspace_path = tharsis_workspace.twvpl_ws.full_path
	vcs_provider_id = "%s"
	repository_path = "%s"
	module_directory = "%s"
	branch = "%s"
	glob_patterns = ["**/*.tf"]
	auto_speculative_plan = false
	webhook_disabled = true
}
	`, workspaceName, vcsGroupPath, vcsProviderID, repositoryPath, moduleDirectory, branch)
}
