package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	ttypes "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

// TestManagedIdentityAWS tests creation, reading, updating, and deletion of an AWS managed identity resource.
func TestManagedIdentityAWS(t *testing.T) {
	createType := string(ttypes.ManagedIdentityAWSFederated)
	createName := "tmi_aws_name"
	createDescription := "this is tmi_aws, a Tharsis managed identity of AWS type"
	createResourcePath := testGroupPath + "/" + createName
	createAWSRole := "some-iam-role"

	updatedDescription := "this is an updated description for tmi_aws"
	updatedAWSRole := "updated-iam-role"

	resource.Test(t, resource.TestCase{

		// AWS managed identities
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{

			// Create and read back a managed identity.
			{
				Config: testSharedProviderConfiguration() + testManagedIdentityAWSConfigurationCreate(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_managed_identity.tmi_aws", "type", createType),
					resource.TestCheckResourceAttr("tharsis_managed_identity.tmi_aws", "resource_path", createResourcePath),
					resource.TestCheckResourceAttr("tharsis_managed_identity.tmi_aws", "name", createName),
					resource.TestCheckResourceAttr("tharsis_managed_identity.tmi_aws", "description", createDescription),
					resource.TestCheckResourceAttr("tharsis_managed_identity.tmi_aws", "group_path", testGroupPath),
					resource.TestCheckResourceAttr("tharsis_managed_identity.tmi_aws", "aws_role", createAWSRole),
					// Azure client_id and Azure tenant_id should not be set, but we cannot check that.

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_managed_identity.tmi_aws", "id"),
					resource.TestCheckResourceAttrSet("tharsis_managed_identity.tmi_aws", "subject"),
					resource.TestCheckResourceAttrSet("tharsis_managed_identity.tmi_aws", "last_updated"),
				),
			},

			// Import state.
			{
				ResourceName:      "tharsis_managed_identity.tmi_aws",
				ImportState:       true,
				ImportStateVerify: true,
			},

			// Update and read.
			{
				// Update and read back a managed identity.
				Config: testSharedProviderConfiguration() + testManagedIdentityAWSConfigurationUpdate(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_managed_identity.tmi_aws", "type", createType),
					resource.TestCheckResourceAttr("tharsis_managed_identity.tmi_aws", "resource_path", createResourcePath),
					resource.TestCheckResourceAttr("tharsis_managed_identity.tmi_aws", "name", createName),
					resource.TestCheckResourceAttr("tharsis_managed_identity.tmi_aws", "description", updatedDescription),
					resource.TestCheckResourceAttr("tharsis_managed_identity.tmi_aws", "group_path", testGroupPath),
					resource.TestCheckResourceAttr("tharsis_managed_identity.tmi_aws", "aws_role", updatedAWSRole),
					// Azure client_id and Azure tenant_id should not be set, but we cannot check that.

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_managed_identity.tmi_aws", "id"),
					resource.TestCheckResourceAttrSet("tharsis_managed_identity.tmi_aws", "subject"),
					resource.TestCheckResourceAttrSet("tharsis_managed_identity.tmi_aws", "last_updated"),
				),
			},

			// Destroy should be covered automatically by TestCase.
		},
	})
}

// TestManagedIdentityAzure tests creation, reading, updating, and deletion of an Azure managed identity resource.
func TestManagedIdentityAzure(t *testing.T) {
	createType := string(ttypes.ManagedIdentityAzureFederated)
	createName := "tmi_azure_name"
	createDescription := "this is tmi_azure, a Tharsis managed identity of Azure type"
	createResourcePath := testGroupPath + "/" + createName
	createAzureClientID := "some-azure-client-id"
	createAzureTenantID := "some-azure-tenant-id"

	updatedDescription := "this is an updated description for tmi_azure"
	updatedAzureClientID := "updated-azure-client-id"
	updatedAzureTenantID := "updated-azure-tenant-id"

	resource.Test(t, resource.TestCase{

		// Azure managed identities
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{

			// Create and read back a managed identity.
			{
				Config: testSharedProviderConfiguration() + testManagedIdentityAzureConfigurationCreate(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_managed_identity.tmi_azure", "type", createType),
					resource.TestCheckResourceAttr("tharsis_managed_identity.tmi_azure", "resource_path", createResourcePath),
					resource.TestCheckResourceAttr("tharsis_managed_identity.tmi_azure", "name", createName),
					resource.TestCheckResourceAttr("tharsis_managed_identity.tmi_azure", "description", createDescription),
					resource.TestCheckResourceAttr("tharsis_managed_identity.tmi_azure", "group_path", testGroupPath),
					resource.TestCheckResourceAttr("tharsis_managed_identity.tmi_azure", "azure_client_id", createAzureClientID),
					resource.TestCheckResourceAttr("tharsis_managed_identity.tmi_azure", "azure_tenant_id", createAzureTenantID),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_managed_identity.tmi_azure", "id"),
					resource.TestCheckResourceAttrSet("tharsis_managed_identity.tmi_azure", "subject"),
					resource.TestCheckResourceAttrSet("tharsis_managed_identity.tmi_azure", "last_updated"),
				),
			},

			// Import state.
			{
				ResourceName:      "tharsis_managed_identity.tmi_azure",
				ImportState:       true,
				ImportStateVerify: true,
			},

			// Update and read.
			{
				// Update and read back a managed identity.
				Config: testSharedProviderConfiguration() + testManagedIdentityAzureConfigurationUpdate(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_managed_identity.tmi_azure", "type", createType),
					resource.TestCheckResourceAttr("tharsis_managed_identity.tmi_azure", "resource_path", createResourcePath),
					resource.TestCheckResourceAttr("tharsis_managed_identity.tmi_azure", "name", createName),
					resource.TestCheckResourceAttr("tharsis_managed_identity.tmi_azure", "description", updatedDescription),
					resource.TestCheckResourceAttr("tharsis_managed_identity.tmi_azure", "group_path", testGroupPath),
					resource.TestCheckResourceAttr("tharsis_managed_identity.tmi_azure", "azure_client_id", updatedAzureClientID),
					resource.TestCheckResourceAttr("tharsis_managed_identity.tmi_azure", "azure_tenant_id", updatedAzureTenantID),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_managed_identity.tmi_azure", "id"),
					resource.TestCheckResourceAttrSet("tharsis_managed_identity.tmi_azure", "subject"),
					resource.TestCheckResourceAttrSet("tharsis_managed_identity.tmi_azure", "last_updated"),
				),
			},

			// Destroy should be covered automatically by TestCase.
		},
	})
}

// TestManagedIdentityTharsis tests creation, reading, updating, and deletion of a Tharsis managed identity resource.
func TestManagedIdentityTharsis(t *testing.T) {
	createType := string(ttypes.ManagedIdentityTharsisFederated)
	createName := "tmi_tharsis_name"
	createDescription := "this is tmi_tharsis, a Tharsis managed identity of Tharsis type"
	createResourcePath := testGroupPath + "/" + createName
	createTharsisServiceAccountPath := "some-tharsis-service-account-path"

	updatedDescription := "this is an updated description for tmi_tharsis"
	updatedTharsisServiceAccountPath := "updated-tharsis-service-account-path"

	resource.Test(t, resource.TestCase{

		// Tharsis managed identities
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{

			// Create and read back a managed identity.
			{
				Config: testSharedProviderConfiguration() + testManagedIdentityTharsisConfigurationCreate(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_managed_identity.tmi_tharsis", "type", createType),
					resource.TestCheckResourceAttr("tharsis_managed_identity.tmi_tharsis", "resource_path", createResourcePath),
					resource.TestCheckResourceAttr("tharsis_managed_identity.tmi_tharsis", "name", createName),
					resource.TestCheckResourceAttr("tharsis_managed_identity.tmi_tharsis", "description", createDescription),
					resource.TestCheckResourceAttr("tharsis_managed_identity.tmi_tharsis", "group_path", testGroupPath),
					resource.TestCheckResourceAttr("tharsis_managed_identity.tmi_tharsis", "tharsis_service_account_path",
						createTharsisServiceAccountPath),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_managed_identity.tmi_tharsis", "id"),
					resource.TestCheckResourceAttrSet("tharsis_managed_identity.tmi_tharsis", "subject"),
					resource.TestCheckResourceAttrSet("tharsis_managed_identity.tmi_tharsis", "last_updated"),
				),
			},

			// Import state.
			{
				ResourceName:      "tharsis_managed_identity.tmi_tharsis",
				ImportState:       true,
				ImportStateVerify: true,
			},

			// Update and read.
			{
				// Update and read back a managed identity.
				Config: testSharedProviderConfiguration() + testManagedIdentityTharsisConfigurationUpdate(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_managed_identity.tmi_tharsis", "type", createType),
					resource.TestCheckResourceAttr("tharsis_managed_identity.tmi_tharsis", "resource_path", createResourcePath),
					resource.TestCheckResourceAttr("tharsis_managed_identity.tmi_tharsis", "name", createName),
					resource.TestCheckResourceAttr("tharsis_managed_identity.tmi_tharsis", "description", updatedDescription),
					resource.TestCheckResourceAttr("tharsis_managed_identity.tmi_tharsis", "group_path", testGroupPath),
					resource.TestCheckResourceAttr("tharsis_managed_identity.tmi_tharsis", "tharsis_service_account_path",
						updatedTharsisServiceAccountPath),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_managed_identity.tmi_tharsis", "id"),
					resource.TestCheckResourceAttrSet("tharsis_managed_identity.tmi_tharsis", "subject"),
					resource.TestCheckResourceAttrSet("tharsis_managed_identity.tmi_tharsis", "last_updated"),
				),
			},

			// Destroy should be covered automatically by TestCase.
		},
	})
}

func testManagedIdentityAWSConfigurationCreate() string {
	createType := string(ttypes.ManagedIdentityAWSFederated)
	createName := "tmi_aws_name"
	createDescription := "this is tmi_aws, a Tharsis managed identity of AWS type"
	createAWSRole := "some-iam-role"
	return fmt.Sprintf(`

%s

resource "tharsis_managed_identity" "tmi_aws" {
	type        = "%s"
	name        = "%s"
	description = "%s"
	group_path  = tharsis_group.root-group.full_path
	aws_role    = "%s"
}

	`, createRootGroup(), createType, createName, createDescription, createAWSRole)
}

func testManagedIdentityAWSConfigurationUpdate() string {
	createType := string(ttypes.ManagedIdentityAWSFederated)
	createName := "tmi_aws_name"
	updatedDescription := "this is an updated description for tmi_aws"
	updatedAWSRole := "updated-iam-role"
	return fmt.Sprintf(`

%s

resource "tharsis_managed_identity" "tmi_aws" {
	type        = "%s"
	name        = "%s"
	description = "%s"
	group_path  = tharsis_group.root-group.full_path
	aws_role    = "%s"
}

	`, createRootGroup(), createType, createName, updatedDescription, updatedAWSRole)
}

func testManagedIdentityAzureConfigurationCreate() string {
	createType := string(ttypes.ManagedIdentityAzureFederated)
	createName := "tmi_azure_name"
	createDescription := "this is tmi_azure, a Tharsis managed identity of Azure type"
	createAzureClientID := "some-azure-client-id"
	createAzureTenantID := "some-azure-tenant-id"
	return fmt.Sprintf(`

%s

resource "tharsis_managed_identity" "tmi_azure" {
	type            = "%s"
	name            = "%s"
	description     = "%s"
	group_path      = tharsis_group.root-group.full_path
	azure_client_id = "%s"
	azure_tenant_id = "%s"
}

	`, createRootGroup(), createType, createName, createDescription, createAzureClientID, createAzureTenantID)
}

func testManagedIdentityAzureConfigurationUpdate() string {
	createType := string(ttypes.ManagedIdentityAzureFederated)
	createName := "tmi_azure_name"
	updatedDescription := "this is an updated description for tmi_azure"
	updatedAzureClientID := "updated-azure-client-id"
	updatedAzureTenantID := "updated-azure-tenant-id"
	return fmt.Sprintf(`

%s

resource "tharsis_managed_identity" "tmi_azure" {
	type            = "%s"
	name            = "%s"
	description     = "%s"
	group_path      = tharsis_group.root-group.full_path
	azure_client_id = "%s"
	azure_tenant_id = "%s"
}

	`, createRootGroup(), createType, createName, updatedDescription, updatedAzureClientID, updatedAzureTenantID)
}

func testManagedIdentityTharsisConfigurationCreate() string {
	createType := string(ttypes.ManagedIdentityTharsisFederated)
	createName := "tmi_tharsis_name"
	createDescription := "this is tmi_tharsis, a Tharsis managed identity of Tharsis type"
	createTharsisServiceAccountPath := "some-tharsis-service-account-path"
	return fmt.Sprintf(`

%s

resource "tharsis_managed_identity" "tmi_tharsis" {
	type                         = "%s"
	name                         = "%s"
	description                  = "%s"
	group_path                   = tharsis_group.root-group.full_path
	tharsis_service_account_path = "%s"
}

	`, createRootGroup(), createType, createName, createDescription, createTharsisServiceAccountPath)
}

func testManagedIdentityTharsisConfigurationUpdate() string {
	createType := string(ttypes.ManagedIdentityTharsisFederated)
	createName := "tmi_tharsis_name"
	updatedDescription := "this is an updated description for tmi_tharsis"
	updatedTharsisServiceAccountPath := "updated-tharsis-service-account-path"
	return fmt.Sprintf(`

%s

resource "tharsis_managed_identity" "tmi_tharsis" {
	type                         = "%s"
	name                         = "%s"
	description                  = "%s"
	group_path                   = tharsis_group.root-group.full_path
	tharsis_service_account_path = "%s"
}

	`, createRootGroup(), createType, createName, updatedDescription, updatedTharsisServiceAccountPath)
}

// The End.
