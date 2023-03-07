package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestVCSProvider(t *testing.T) {
	createName := "tvp_name"
	createDescription := "this is tvp, a test VCS provider"
	createHostname := "test-vcs-provider-hostname"
	createGroupPath := testGroupPath
	createResourcePath := testGroupPath + "/" + createName
	createType := "gitlab"
	createAutoCreateWebhooks := true

	updateDescription := "this is tvp's updated description"

	resource.Test(t, resource.TestCase{

		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{

			// Create and read back a VCS provider.
			{
				Config: testVCSProviderConfigurationCreate(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_vcs_provider.tvp", "name", createName),
					resource.TestCheckResourceAttr("tharsis_vcs_provider.tvp", "description", createDescription),
					resource.TestCheckResourceAttr("tharsis_vcs_provider.tvp", "hostname", createHostname),
					resource.TestCheckResourceAttr("tharsis_vcs_provider.tvp", "group_path", createGroupPath),
					resource.TestCheckResourceAttr("tharsis_vcs_provider.tvp", "resource_path", createResourcePath),
					resource.TestCheckResourceAttr("tharsis_vcs_provider.tvp", "type", createType),
					resource.TestCheckResourceAttr("tharsis_vcs_provider.tvp", "auto_create_webhooks",
						strconv.FormatBool(createAutoCreateWebhooks)),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_vcs_provider.tvp", "id"),
					resource.TestCheckResourceAttrSet("tharsis_vcs_provider.tvp", "last_updated"),
					resource.TestCheckResourceAttrSet("tharsis_vcs_provider.tvp", "created_by"),
					resource.TestCheckResourceAttrSet("tharsis_vcs_provider.tvp", "resource_path"),

					// The only time this can be checked is immediately after a create operation.
					resource.TestCheckResourceAttrSet("tharsis_vcs_provider.tvp", "oauth_authorization_url"),

					// OAuthClientID and OAuthClientSecret are write-only, so there's nothing to verify here.
				),
			},

			// Import the state.
			// The OAuthClientID and OAuthClientSecret fields are write-only,
			// so they cannot be verified during import.
			{
				ResourceName:            "tharsis_vcs_provider.tvp",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"oauth_client_id", "oauth_client_secret", "oauth_authorization_url"},
			},

			// Update and read.
			{
				Config: testVCSProviderConfigurationUpdate(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_vcs_provider.tvp", "name", createName),
					resource.TestCheckResourceAttr("tharsis_vcs_provider.tvp", "description", updateDescription),
					resource.TestCheckResourceAttr("tharsis_vcs_provider.tvp", "hostname", createHostname),
					resource.TestCheckResourceAttr("tharsis_vcs_provider.tvp", "group_path", createGroupPath),
					resource.TestCheckResourceAttr("tharsis_vcs_provider.tvp", "resource_path", createResourcePath),
					resource.TestCheckResourceAttr("tharsis_vcs_provider.tvp", "type", createType),
					resource.TestCheckResourceAttr("tharsis_vcs_provider.tvp", "auto_create_webhooks",
						strconv.FormatBool(createAutoCreateWebhooks)),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_vcs_provider.tvp", "id"),
					resource.TestCheckResourceAttrSet("tharsis_vcs_provider.tvp", "last_updated"),
					resource.TestCheckResourceAttrSet("tharsis_vcs_provider.tvp", "created_by"),
					resource.TestCheckResourceAttrSet("tharsis_vcs_provider.tvp", "resource_path"),

					// OAuthClientID and OAuthClientSecret are write-only, so there's nothing to verify here.
				),
			},

			// Destroy should be covered automatically by TestCase.

		},
	})
}

func testVCSProviderConfigurationCreate() string {
	createName := "tvp_name"
	createDescription := "this is tvp, a test VCS provider"
	createHostname := "test-vcs-provider-hostname"
	createType := "gitlab"
	createAutoCreateWebhooks := true
	createOAuthClientID := "tvp-oauth-client-id"
	createOAuthClientSecret := "tvp-oauth-client-secret"

	return fmt.Sprintf(`

%s

resource "tharsis_vcs_provider" "tvp" {
	name = "%s"
	description = "%s"
	hostname = "%s"
	group_path = tharsis_group.root-group.full_path
	type = "%s"
	auto_create_webhooks = %s
	oauth_client_id = "%s"
	oauth_client_secret = "%s"
}
	`, createRootGroup(), createName, createDescription,
		createHostname, createType, strconv.FormatBool(createAutoCreateWebhooks),
		createOAuthClientID, createOAuthClientSecret)
}

func testVCSProviderConfigurationUpdate() string {
	createName := "tvp_name"
	createHostname := "test-vcs-provider-hostname"
	createType := "gitlab"
	createAutoCreateWebhooks := true

	updateDescription := "this is tvp's updated description"
	updateOAuthClientID := "tvp-oauth-client-updated-id"
	updateOAuthClientSecret := "tvp-oauth-client-updated-secret"

	return fmt.Sprintf(`

%s

resource "tharsis_vcs_provider" "tvp" {
	name = "%s"
	description = "%s"
	hostname = "%s"
	group_path = tharsis_group.root-group.full_path
	type = "%s"
	auto_create_webhooks = %s
	oauth_client_id = "%s"
	oauth_client_secret = "%s"
}
	`, createRootGroup(), createName, updateDescription,
		createHostname, createType, strconv.FormatBool(createAutoCreateWebhooks),
		updateOAuthClientID, updateOAuthClientSecret)
}

// The End.
