package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	ttypes "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

// TestManagedIdentityTharsis tests creation, reading, updating, and deletion of a managed identity alias resource.
func TestManagedIdentityAlias(t *testing.T) {
	createName := "tmi_alias"
	createResourcePath := testGroupPath + "/" + createName

	resource.Test(t, resource.TestCase{

		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{

			// Create and read back a managed identity alias.
			{
				Config: testSharedProviderConfiguration() + testManagedIdentityAliasConfigurationCreate(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_managed_identity_alias.tmi_alias", "resource_path", createResourcePath),
					resource.TestCheckResourceAttr("tharsis_managed_identity_alias.tmi_alias", "name", createName),
					resource.TestCheckResourceAttr("tharsis_managed_identity_alias.tmi_alias", "group_path", testGroupPath),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_managed_identity_alias.tmi_alias", "id"),
					resource.TestCheckResourceAttrSet("tharsis_managed_identity_alias.tmi_alias", "last_updated"),
					resource.TestCheckResourceAttrSet("tharsis_managed_identity_alias.tmi_alias", "alias_source_id"),
				),
			},

			// Import state.
			{
				ResourceName:      "tharsis_managed_identity_alias.tmi_alias",
				ImportState:       true,
				ImportStateVerify: true,
			},

			// Update and read.
			{
				// Update and read back a managed identity alias.
				Config: testSharedProviderConfiguration() + testManagedIdentityAliasConfigurationUpdate(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_managed_identity_alias.tmi_alias", "resource_path", createResourcePath),
					resource.TestCheckResourceAttr("tharsis_managed_identity_alias.tmi_alias", "name", createName),
					resource.TestCheckResourceAttr("tharsis_managed_identity_alias.tmi_alias", "group_path", testGroupPath),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_managed_identity_alias.tmi_alias", "id"),
					resource.TestCheckResourceAttrSet("tharsis_managed_identity_alias.tmi_alias", "last_updated"),
					resource.TestCheckResourceAttrSet("tharsis_managed_identity_alias.tmi_alias", "alias_source_id"),
				),
			},

			// Destroy should be covered automatically by TestCase.
		},
	})
}

func testManagedIdentityAliasConfigurationCreate() string {
	sourceIdentityType := string(ttypes.ManagedIdentityAWSFederated)
	sourceIdentityName := "tmi_aws_name"
	sourceIdentityDescription := "this is tmi_aws, a Tharsis managed identity of AWS type"
	sourceIdentityAWSRole := "some-iam-role"

	// Managed identity alias must be created under a different namespace.
	createAliasRootGroupPath := "provider-test-managed-identity-alias-group"
	createAliasRootGroupDescription := "this is a test root group for managed identity alias"

	// Alias fields.
	createAliasName := sourceIdentityName + "_alias"
	return fmt.Sprintf(`

%s

resource "tharsis_managed_identity" "tmi_aws" {
	type        = "%s"
	name        = "%s"
	description = "%s"
	group_path  = tharsis_group.root-group.full_path
	aws_role    = "%s"
}

resource "tharsis_group" "alias-group" {
	name = "%s"
	description = "%s"
}

resource "tharsis_managed_identity_alias" "tmi_alias" {
	name = "%s"
	group_path = tharsis_group.alias-group.full_path
	alias_source_id = tharsis_managed_identity.tmi_aws.id
}

	`, createRootGroup(),
		sourceIdentityType,
		sourceIdentityName,
		sourceIdentityDescription,
		sourceIdentityAWSRole,
		createAliasRootGroupPath,
		createAliasRootGroupDescription,
		createAliasName,
	)
}

func testManagedIdentityAliasConfigurationUpdate() string {
	sourceIdentityType := string(ttypes.ManagedIdentityAWSFederated)
	sourceIdentityName := "tmi_aws_name"
	sourceIdentityDescription := "this is tmi_aws, a Tharsis managed identity of AWS type"
	sourceIdentityAWSRole := "some-iam-role"
	createName := "tmi_aws_name"
	updateAliasRootGroupPath := "provider-test-managed-identity-alias-group"
	updateAliasRootGroupDescription := "this is a test root group for managed identity alias"
	return fmt.Sprintf(`

%s

resource "tharsis_managed_identity" "tmi_aws" {
	type        = "%s"
	name        = "%s"
	description = "%s"
	group_path  = tharsis_group.root-group.full_path
	aws_role    = "%s"
}

resource "tharsis_group" "alias-group" {
	name = "%s"
	description = "%s"
}

resource "tharsis_managed_identity_alias" "tmi_alias" {
	name = "%s"
	group_path = tharsis_group.alias-group.full_path
	alias_source_id = tharsis_managed_identity.tmi_aws.id
}

	`, createRootGroup(),
		sourceIdentityType,
		sourceIdentityName,
		sourceIdentityDescription,
		sourceIdentityAWSRole,
		createName,
		updateAliasRootGroupPath,
		updateAliasRootGroupDescription,
	)
}
