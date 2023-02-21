package provider

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// FIXME: Asked Brandon: might have to scrap this test for now due to the requirement to
// go through the OAuth flow before creating a workspace VCS provider link.

func TestWorkspaceVCSProviderLink(t *testing.T) {
	createModuleDirectory := "twvpl-module-directory-1"
	createRepositoryPath := "twvpl-repository-path-1"
	createWorkspacePath := "twvpl-workspace-path-1"
	createProviderID := "tharsis_vcs_provider.wvpl_test_vcs_provider.id"
	createBranch := "twvpl-branch-1"
	createTagRegex := "twvpl-tag-regex-1"
	createGlobPatterns := []string{"twvpl-glob-patterns-1a", "twvpl-glob-patterns-1b"}
	createAutoSpeculativePlan := true
	createWebhookDisabled := false

	updateModuleDirectory := "twvpl-updated-module-directory-1"
	updateBranch := "twvpl-updated-branch-1"
	updateTagRegex := "twvpl-updated-tag-regex-1"
	updateGlobPatterns := []string{"twvpl-updated-glob-patterns-1a", "twvpl-updated-glob-patterns-1b",
		"twvpl-updated-glob-patterns-1c"}
	updateAutoSpeculativePlan := false
	updateWebhookDisabled := true

	resource.Test(t, resource.TestCase{

		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{

			// Create and read back a workspace VCS provider link.
			{
				Config: testWorkspaceVCSProviderLinkConfigurationCreate(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_workspace_vcs_provider_link.twvpl",
						"module_directory", createModuleDirectory),
					resource.TestCheckResourceAttr("tharsis_workspace_vcs_provider_link.twvpl",
						"repository_path", createRepositoryPath),
					resource.TestCheckResourceAttr("tharsis_workspace_vcs_provider_link.twvpl",
						"workspace_path", createWorkspacePath),
					resource.TestCheckResourceAttr("tharsis_workspace_vcs_provider_link.twvpl",
						"vcs_provider_id", createProviderID),
					resource.TestCheckResourceAttr("tharsis_workspace_vcs_provider_link.twvpl",
						"branch", createBranch),
					resource.TestCheckResourceAttr("tharsis_workspace_vcs_provider_link.twvpl",
						"tag_regex", createTagRegex),
					resource.TestCheckResourceAttr("tharsis_workspace_vcs_provider_link.twvpl",
						"glob_patterns.#", strconv.Itoa(len(createGlobPatterns))),
					resource.TestCheckResourceAttr("tharsis_workspace_vcs_provider_link.twvpl",
						"glob_patterns.0", createGlobPatterns[0]),
					resource.TestCheckResourceAttr("tharsis_workspace_vcs_provider_link.twvpl",
						"glob_patterns.1", createGlobPatterns[1]),
					resource.TestCheckResourceAttr("tharsis_workspace_vcs_provider_link.twvpl",
						"auto_speculative_plan", strconv.FormatBool(createAutoSpeculativePlan)),
					resource.TestCheckResourceAttr("tharsis_workspace_vcs_provider_link.twvpl",
						"webhook_disable", strconv.FormatBool(createWebhookDisabled)),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_workspace_vcs_provider_link.twvpl",
						"id"),
					resource.TestCheckResourceAttrSet("tharsis_workspace_vcs_provider_link.twvpl",
						"last_updated"),
				),
			},

			// Import the state.
			{
				ResourceName:      "tharsis_workspace_vcs_provider_link.twvpl",
				ImportStateId:     createWorkspacePath,
				ImportState:       true,
				ImportStateVerify: true,
			},

			// Update and read.
			{
				Config: testWorkspaceVCSProviderLinkConfigurationUpdate(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_workspace_vcs_provider_link.twvpl",
						"module_directory", updateModuleDirectory),
					resource.TestCheckResourceAttr("tharsis_workspace_vcs_provider_link.twvpl",
						"repository_path", createRepositoryPath),
					resource.TestCheckResourceAttr("tharsis_workspace_vcs_provider_link.twvpl",
						"workspace_path", createWorkspacePath),
					resource.TestCheckResourceAttr("tharsis_workspace_vcs_provider_link.twvpl",
						"vcs_provider_id", createProviderID),
					resource.TestCheckResourceAttr("tharsis_workspace_vcs_provider_link.twvpl",
						"branch", updateBranch),
					resource.TestCheckResourceAttr("tharsis_workspace_vcs_provider_link.twvpl",
						"tag_regex", updateTagRegex),
					resource.TestCheckResourceAttr("tharsis_workspace_vcs_provider_link.twvpl",
						"glob_patterns.#", strconv.Itoa(len(updateGlobPatterns))),
					resource.TestCheckResourceAttr("tharsis_workspace_vcs_provider_link.twvpl",
						"glob_patterns.0", updateGlobPatterns[0]),
					resource.TestCheckResourceAttr("tharsis_workspace_vcs_provider_link.twvpl",
						"glob_patterns.1", updateGlobPatterns[1]),
					resource.TestCheckResourceAttr("tharsis_workspace_vcs_provider_link.twvpl",
						"glob_patterns.2", updateGlobPatterns[2]),
					resource.TestCheckResourceAttr("tharsis_workspace_vcs_provider_link.twvpl",
						"auto_speculative_plan", strconv.FormatBool(updateAutoSpeculativePlan)),
					resource.TestCheckResourceAttr("tharsis_workspace_vcs_provider_link.twvpl",
						"webhook_disable", strconv.FormatBool(updateWebhookDisabled)),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_workspace_vcs_provider_link.twvpl",
						"id"),
					resource.TestCheckResourceAttrSet("tharsis_workspace_vcs_provider_link.twvpl",
						"last_updated"),
				),
			},

			// Destroy should be covered automatically by TestCase.

		},
	})
}

// FIXME: Probably need to create VCS provider via TF.

func testWorkspaceVCSProviderLinkConfigurationCreate() string {
	createModuleDirectory := "twvpl-module-directory-1"
	createRepositoryPath := "twvpl-repository-path-1"
	createBranch := "twvpl-branch-1"
	createTagRegex := "twvpl-tag-regex-1"
	createGlobPatterns := []string{"twvpl-glob-patterns-1a", "twvpl-glob-patterns-1b"}
	createAutoSpeculativePlan := true
	createWebhookDisabled := false

	return fmt.Sprintf(`

%s

%s

%s

resource "tharsis_workspace_vcs_provider_link" "twvpl" {
	module_directory = "%s"
	repository_path = "%s"
	workspace_path = tharsis_workspace.wvpl_test_workspace.full_path
	vcs_provider_id = tharsis_vcs_provider.wvpl_test_vcs_provider.id
	branch = "%s"
	tag_regex = "%s"
	glob_patterns = %s
	auto_speculative_plan = %v
	webhook_disabled = %v
}
	`, createRootGroup(), createTestWorkspace(), createTestVCSProvider(),
		createModuleDirectory, createRepositoryPath, createBranch, createTagRegex,
		formatStringSlice(createGlobPatterns), createAutoSpeculativePlan, createWebhookDisabled)
}

func testWorkspaceVCSProviderLinkConfigurationUpdate() string {
	createRepositoryPath := "twvpl-repository-path-1"

	updateModuleDirectory := "twvpl-updated-module-directory-1"
	updateBranch := "twvpl-updated-branch-1"
	updateTagRegex := "twvpl-updated-tag-regex-1"
	updateGlobPatterns := []string{"twvpl-updated-glob-patterns-1a", "twvpl-updated-glob-patterns-1b",
		"twvpl-updated-glob-patterns-1c"}
	updateAutoSpeculativePlan := false
	updateWebhookDisabled := true

	return fmt.Sprintf(`

%s

%s

%s

resource "tharsis_workspace_vcs_provider_link" "twvpl" {
	module_directory = "%s"
	repository_path = "%s"
	workspace_path = tharsis_workspace.wvpl_test_workspace.full_path
	vcs_provider_id = tharsis_vcs_provider.wvpl_test_vcs_provider.id
	branch = "%s"
	tag_regex = "%s"
	glob_patterns = %s
	auto_speculative_plan = %v
	webhook_disabled = %v
}
	`, createRootGroup(), createTestWorkspace(), createTestVCSProvider(),
		updateModuleDirectory, createRepositoryPath, updateBranch, updateTagRegex,
		formatStringSlice(updateGlobPatterns), updateAutoSpeculativePlan, updateWebhookDisabled)
}

func createTestWorkspace() string {
	createTestWorkspaceName := "wvpl-test-workspace"
	createTestWorkspaceDescription := "this is a test workspace"

	return fmt.Sprintf(`

resource "tharsis_workspace" "wvpl_test_workspace" {
	name = "%s"
	description = "%s"
	group_path = tharsis_group.root-group.full_path
}
	`, createTestWorkspaceName, createTestWorkspaceDescription)
}

func createTestVCSProvider() string {
	vcspName := "test-vcs-provider-1"
	vcspDescription := "this is a test VCS provider"
	vcspHostname := "example.invalid"
	vcspOAuthClientID := "some-client"
	vcspOAuthClientSecret := "don't tell anyone"
	vcspType := "gitlab"
	vcspAutoCreateWebhooks := false

	return fmt.Sprintf(`

resource "tharsis_vcs_provider" "wvpl_test_vcs_provider" {
	name = "%s"
	description = "%s"
	group_path = tharsis_group.root-group.full_path
	hostname = "%s"
	/*
	FIXME: Keep or remove these?
	oauth_client_id = "%s"
	oauth_client_secret = "%s"
	*/
	type = "%s"
	auto_create_webhooks = %v
}
	`, vcspName, vcspDescription, vcspHostname,
		vcspOAuthClientID, vcspOAuthClientSecret, vcspType, vcspAutoCreateWebhooks)
}

// tharsis_vcs_provider.wvpl_test_vcs_provider.id

func formatStringSlice(arg []string) string {
	if len(arg) == 0 {
		return "[]"
	}

	return "[\"" + strings.Join(arg, "\", \"") + "\"]"
}

// The End.
