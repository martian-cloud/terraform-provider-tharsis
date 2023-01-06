package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestServiceAccount(t *testing.T) {
	createName := "tsa_name"
	createDescription := "this is tsa, a test service account"
	createGroupPath := testGroupPath
	createResourcePath := testGroupPath + "/" + createName
	updatedDescription := "this is an updated description for tsa, a test service account"

	resource.Test(t, resource.TestCase{

		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{

			// Create and read back a service account.
			{
				Config: testServiceAccountConfigurationCreate(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_service_account.tsa", "name", createName),
					resource.TestCheckResourceAttr("tharsis_service_account.tsa", "description", createDescription),
					resource.TestCheckResourceAttr("tharsis_service_account.tsa", "resource_path", createResourcePath),
					resource.TestCheckResourceAttr("tharsis_service_account.tsa", "group_path", createGroupPath),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_service_account.tsa", "id"),
				),
			},

			// Import the state.
			{
				ResourceName:      "tharsis_service_account.tsa",
				ImportState:       true,
				ImportStateVerify: true,
			},

			// Update and read.
			{
				Config: testServiceAccountConfigurationUpdate(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_service_account.tsa", "name", createName),
					resource.TestCheckResourceAttr("tharsis_service_account.tsa", "description", updatedDescription),
					resource.TestCheckResourceAttr("tharsis_service_account.tsa", "resource_path", createResourcePath),
					resource.TestCheckResourceAttr("tharsis_service_account.tsa", "group_path", createGroupPath),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_service_account.tsa", "id"),
				),
			},

			// Destroy should be covered automatically by TestCase.

		},
	})
}

func testServiceAccountConfigurationCreate() string {
	createName := "tsa_name"
	createDescription := "this is tsa, a test service account"
	createTrustPolicyIssuer := "https://first-issuer/"
	createTrustPolicyBoundClaimKey := "first-key"
	createTrustPolicyBoundClaimValue := "first-value"

	return fmt.Sprintf(`

%s

resource "tharsis_service_account" "tsa" {
	name = "%s"
	description = "%s"
	group_path = tharsis_group.root-group.full_path
	oidc_trust_policies = [{bound_claims = {"%s" = "%s"}, issuer = "%s"}]
}
	`, createRootGroup(), createName, createDescription,
		createTrustPolicyBoundClaimKey, createTrustPolicyBoundClaimValue, createTrustPolicyIssuer,
	)
}

func testServiceAccountConfigurationUpdate() string {
	createName := "tsa_name"
	updatedDescription := "this is an updated description for tsa, a test service account"
	updateTrustPolicyIssuer := "https://updated-issuer/"
	updateTrustPolicyBoundClaimKey := "updated-key"
	updateTrustPolicyBoundClaimValue := "updated-value"

	return fmt.Sprintf(`

%s

resource "tharsis_service_account" "tsa" {
	name = "%s"
	description = "%s"
	group_path = tharsis_group.root-group.full_path
	oidc_trust_policies = [{bound_claims = {"%s" = "%s"}, issuer = "%s"}]
}
	`, createRootGroup(), createName, updatedDescription,
		updateTrustPolicyBoundClaimKey, updateTrustPolicyBoundClaimValue, updateTrustPolicyIssuer,
	)
}

// The End.
