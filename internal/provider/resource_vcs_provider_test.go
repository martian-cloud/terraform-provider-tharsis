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
	createURL := "https://vcs-provider.domain"
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
					resource.TestCheckResourceAttr("tharsis_vcs_provider.tvp", "url", createURL),
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

			// Import the state by GID.
			// With no ImportStateId set, the framework reuses the id already in
			// state, which is the GID returned by the API during create.
			//
			// The OAuthClientID and OAuthClientSecret fields are write-only to
			// the Tharsis SDK and OAuthAuthorizationURL is only returned by the
			// create call, so none of the three can be read back. These ignores
			// reflect an intentional API constraint, not an incomplete import.
			{
				ResourceName:            "tharsis_vcs_provider.tvp",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"oauth_client_id", "oauth_client_secret", "oauth_authorization_url"},
			},

			// Import the state by resource path, which is "<group_path>/<name>".
			{
				ResourceName:            "tharsis_vcs_provider.tvp",
				ImportStateId:           createResourcePath,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"oauth_client_id", "oauth_client_secret", "oauth_authorization_url"},
			},

			// Import the state by TRN.
			{
				ResourceName:            "tharsis_vcs_provider.tvp",
				ImportStateId:           "trn:vcs_provider:" + createResourcePath,
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
					resource.TestCheckResourceAttr("tharsis_vcs_provider.tvp", "url", createURL),
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
	createURL := "https://vcs-provider.domain"
	createType := "gitlab"
	createAutoCreateWebhooks := true
	createOAuthClientID := "tvp-oauth-client-id"
	createOAuthClientSecret := "tvp-oauth-client-secret" // #nosec G101 -- This is a false positive

	return fmt.Sprintf(`

%s

resource "tharsis_vcs_provider" "tvp" {
	name = "%s"
	description = "%s"
	url = "%s"
	group_path = tharsis_group.root-group.full_path
	type = "%s"
	auto_create_webhooks = %s
	oauth_client_id = "%s"
	oauth_client_secret = "%s"
}
	`, createRootGroup(testGroupPath, "this is a test root group"), createName, createDescription,
		createURL, createType, strconv.FormatBool(createAutoCreateWebhooks),
		createOAuthClientID, createOAuthClientSecret)
}

func testVCSProviderConfigurationUpdate() string {
	createName := "tvp_name"
	createURL := "https://vcs-provider.domain"
	createType := "gitlab"
	createAutoCreateWebhooks := true

	updateDescription := "this is tvp's updated description"
	updateOAuthClientID := "tvp-oauth-client-updated-id"
	updateOAuthClientSecret := "tvp-oauth-client-updated-secret" // #nosec G101 -- This is a false positive

	return fmt.Sprintf(`

%s

resource "tharsis_vcs_provider" "tvp" {
	name = "%s"
	description = "%s"
	url = "%s"
	group_path = tharsis_group.root-group.full_path
	type = "%s"
	auto_create_webhooks = %s
	oauth_client_id = "%s"
	oauth_client_secret = "%s"
}
	`, createRootGroup(testGroupPath, "this is a test root group"), createName, updateDescription,
		createURL, createType, strconv.FormatBool(createAutoCreateWebhooks),
		updateOAuthClientID, updateOAuthClientSecret)
}
