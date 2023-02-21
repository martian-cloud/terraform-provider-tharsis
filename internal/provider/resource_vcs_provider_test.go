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
	/*
	   FIXME: Keep or remove these?
	   	createClientID := "test-client-id"
	   	createClientSecret := "test-client-secret"
	*/
	createType := "gitlab"
	createAutoCreateWebhooks := true

	updateDescription := "this is tvp's updated description"
	/*
	   FIXME: Keep or remove these?
	   	updateClientID := "updated-test-client-id"
	   	updateClientSecret := "updated-test-client-secret"
	*/

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
					/*
					   FIXME: Keep or remove these?
					   					resource.TestCheckResourceAttr("tharsis_vcs_provider.tvp", "oauth_client_id", createClientID),
					   					resource.TestCheckResourceAttr("tharsis_vcs_provider.tvp", "oauth_client_secret", createClientSecret),
					*/
					resource.TestCheckResourceAttr("tharsis_vcs_provider.tvp", "type", createType),
					resource.TestCheckResourceAttr("tharsis_vcs_provider.tvp", "auto_create_webhooks",
						strconv.FormatBool(createAutoCreateWebhooks)),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_vcs_provider.tvp", "id"),
					resource.TestCheckResourceAttrSet("tharsis_vcs_provider.tvp", "last_updated"),
					resource.TestCheckResourceAttrSet("tharsis_vcs_provider.tvp", "created_by"),
					resource.TestCheckResourceAttrSet("tharsis_vcs_provider.tvp", "resource_path"),
				),
			},

			// Import the state.
			{
				ResourceName:      "tharsis_vcs_provider.tvp",
				ImportState:       true,
				ImportStateVerify: true,
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
					/*
					   FIXME: Keep or remove these?
					   					resource.TestCheckResourceAttr("tharsis_vcs_provider.tvp", "oauth_client_id", updateClientID),
					   					resource.TestCheckResourceAttr("tharsis_vcs_provider.tvp", "oauth_client_secret", updateClientSecret),
					*/
					resource.TestCheckResourceAttr("tharsis_vcs_provider.tvp", "type", createType),
					resource.TestCheckResourceAttr("tharsis_vcs_provider.tvp", "auto_create_webhooks",
						strconv.FormatBool(createAutoCreateWebhooks)),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_vcs_provider.tvp", "id"),
					resource.TestCheckResourceAttrSet("tharsis_vcs_provider.tvp", "last_updated"),
					resource.TestCheckResourceAttrSet("tharsis_vcs_provider.tvp", "created_by"),
					resource.TestCheckResourceAttrSet("tharsis_vcs_provider.tvp", "resource_path"),
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
	createClientID := "test-client-id"
	createClientSecret := "test-client-secret"
	createType := "gitlab"
	createAutoCreateWebhooks := true

	return fmt.Sprintf(`

%s

resource "tharsis_vcs_provider" "tvp" {
	name = "%s"
	description = "%s"
	hostname = "%s"
	group_path = tharsis_group.root-group.full_path
	/*
	FIXME: Keep or remove these?
		oauth_client_id = "%s"
	oauth_client_secret = "%s"
*/
	type = "%s"
	auto_create_webhooks = %s
}
	`, createRootGroup(), createName, createDescription,
		createHostname, createClientID, createClientSecret,
		createType, strconv.FormatBool(createAutoCreateWebhooks))
}

func testVCSProviderConfigurationUpdate() string {
	createName := "tvp_name"
	createHostname := "test-vcs-provider-hostname"
	createType := "gitlab"
	createAutoCreateWebhooks := true

	updateDescription := "this is tvp's updated description"
	updateClientID := "updated-test-client-id"
	updateClientSecret := "updated-test-client-secret"

	return fmt.Sprintf(`

%s

resource "tharsis_vcs_provider" "tvp" {
	name = "%s"
	description = "%s"
	hostname = "%s"
	group_path = tharsis_group.root-group.full_path
	/*
	FIXME: Keep or remove these?
		oauth_client_id = "%s"
	oauth_client_secret = "%s"
*/
	type = "%s"
	auto_create_webhooks = %s
}
	`, createRootGroup(), createName, updateDescription,
		createHostname, updateClientID, updateClientSecret,
		createType, strconv.FormatBool(createAutoCreateWebhooks))
}

// The End.
