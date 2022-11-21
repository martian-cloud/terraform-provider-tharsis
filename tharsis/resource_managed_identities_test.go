package tharsis

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	ttypes "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

const (

	// For now, assume this group exists.  See the TODO comment later in the code.
	testGroupPath = "provider-test-parent-group"
)

// TODO: For now, we're assuming the group exists.
// Eventually, the tests will need to use the Provider to create/destroy the group.  See this as an example:
// https://github.com/hashicorp/terraform-provider-tfe/blob/main/tfe/resource_tfe_workspace_run_task_test.go#L200

// TestManagedIdentityAWS tests creation, reading, updating, and deletion of an AWS managed identity resource.
func TestManagedIdentityAWS(t *testing.T) {
	createType := string(ttypes.ManagedIdentityAWSFederated)
	createName := "tmi_aws_name"
	createDescription := "this is tmi_aws, a Tharsis managed identity"
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
	createDescription := "this is tmi_azure, a Tharsis managed identity"
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

func testManagedIdentityAWSConfigurationCreate() string {
	createType := string(ttypes.ManagedIdentityAWSFederated)
	createName := "tmi_aws_name"
	createDescription := "this is tmi_aws, a Tharsis managed identity"
	createAWSRole := "some-iam-role"
	return fmt.Sprintf(`

resource "tharsis_managed_identity" "tmi_aws" {
	type        = "%s"
	name        = "%s"
	description = "%s"
	group_path  = "%s"
	aws_role    = "%s"
}

	`, createType, createName, createDescription, testGroupPath, createAWSRole)
}

func testManagedIdentityAWSConfigurationUpdate() string {
	createType := string(ttypes.ManagedIdentityAWSFederated)
	createName := "tmi_aws_name"
	updatedDescription := "this is an updated description for tmi_aws"
	updatedAWSRole := "updated-iam-role"
	return fmt.Sprintf(`

	resource "tharsis_managed_identity" "tmi_aws" {
		type        = "%s"
		name        = "%s"
		description = "%s"
		group_path  = "%s"
		aws_role    = "%s"
	}

	`, createType, createName, updatedDescription, testGroupPath, updatedAWSRole)
}

func testManagedIdentityAzureConfigurationCreate() string {
	createType := string(ttypes.ManagedIdentityAzureFederated)
	createName := "tmi_azure_name"
	createDescription := "this is tmi_azure, a Tharsis managed identity"
	createAzureClientID := "some-azure-client-id"
	createAzureTenantID := "some-azure-tenant-id"
	return fmt.Sprintf(`

resource "tharsis_managed_identity" "tmi_azure" {
	type            = "%s"
	name            = "%s"
	description     = "%s"
	group_path      = "%s"
	azure_client_id = "%s"
	azure_tenant_id = "%s"
}

	`, createType, createName, createDescription, testGroupPath, createAzureClientID, createAzureTenantID)
}

func testManagedIdentityAzureConfigurationUpdate() string {
	createType := string(ttypes.ManagedIdentityAzureFederated)
	createName := "tmi_azure_name"
	updatedDescription := "this is an updated description for tmi_azure"
	updatedAzureClientID := "updated-azure-client-id"
	updatedAzureTenantID := "updated-azure-tenant-id"
	return fmt.Sprintf(`

	resource "tharsis_managed_identity" "tmi_azure" {
		type            = "%s"
		name            = "%s"
		description     = "%s"
		group_path      = "%s"
		azure_client_id = "%s"
		azure_tenant_id = "%s"
	}

	`, createType, createName, updatedDescription, testGroupPath, updatedAzureClientID, updatedAzureTenantID)
}

// The End.
