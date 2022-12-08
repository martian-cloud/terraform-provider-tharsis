package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

type testServiceAccountConstants struct {
	createName                       string
	createDescription                string
	createGroupPath                  string
	createResourcePath               string
	createTrustPolicyIssuer          string
	createTrustPolicyBoundClaimKey   string
	createTrustPolicyBoundClaimValue string
	updatedDescription               string
	updateTrustPolicyIssuer          string
	updateTrustPolicyBoundClaimKey   string
	updateTrustPolicyBoundClaimValue string
}

func TestServiceAccount(t *testing.T) {
	c := buildTestServiceAccountConstants()

	resource.Test(t, resource.TestCase{

		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{

			// Create and read back a service account.
			{
				Config: testServiceAccountConfigurationCreate(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_service_account.tsa", "name", c.createName),
					resource.TestCheckResourceAttr("tharsis_service_account.tsa", "description", c.createDescription),
					resource.TestCheckResourceAttr("tharsis_service_account.tsa", "resource_path", c.createResourcePath),
					resource.TestCheckResourceAttr("tharsis_service_account.tsa", "group_path", c.createGroupPath),

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
					resource.TestCheckResourceAttr("tharsis_service_account.tsa", "name", c.createName),
					resource.TestCheckResourceAttr("tharsis_service_account.tsa", "description", c.updatedDescription),
					resource.TestCheckResourceAttr("tharsis_service_account.tsa", "resource_path", c.createResourcePath),
					resource.TestCheckResourceAttr("tharsis_service_account.tsa", "group_path", c.createGroupPath),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_service_account.tsa", "id"),
				),
			},

			// Destroy should be covered automatically by TestCase.

		},
	})
}

func testServiceAccountConfigurationCreate() string {
	c := buildTestServiceAccountConstants()
	return fmt.Sprintf(`

resource "tharsis_service_account" "tsa" {
	name = "%s"
	description = "%s"
	group_path = "%s"
	oidc_trust_policies = [{bound_claims = {"%s" = "%s"}, issuer = "%s"}]
}
	`, c.createName, c.createDescription, c.createGroupPath,
		c.createTrustPolicyBoundClaimKey, c.createTrustPolicyBoundClaimValue, c.createTrustPolicyIssuer,
	)
}

func testServiceAccountConfigurationUpdate() string {
	c := buildTestServiceAccountConstants()
	return fmt.Sprintf(`

resource "tharsis_service_account" "tsa" {
	name = "%s"
	description = "%s"
	group_path = "%s"
	oidc_trust_policies = [{bound_claims = {"%s" = "%s"}, issuer = "%s"}]
}
	`, c.createName, c.updatedDescription, c.createGroupPath,
		c.updateTrustPolicyBoundClaimKey, c.updateTrustPolicyBoundClaimValue, c.updateTrustPolicyIssuer,
	)
}

func buildTestServiceAccountConstants() *testServiceAccountConstants {
	createName := "tsa_name"
	return &testServiceAccountConstants{
		createName:                       createName,
		createDescription:                "this is tsa, a test service account",
		createGroupPath:                  testGroupPath,
		createResourcePath:               testGroupPath + "/" + createName,
		createTrustPolicyIssuer:          "https://first-issuer/",
		createTrustPolicyBoundClaimKey:   "first-key",
		createTrustPolicyBoundClaimValue: "first-value",
		updatedDescription:               "this is an updated description for tsa, a test service account",
		updateTrustPolicyIssuer:          "https://updated-issuer/",
		updateTrustPolicyBoundClaimKey:   "updated-key",
		updateTrustPolicyBoundClaimValue: "updated-value",
	}
}

// The End.
