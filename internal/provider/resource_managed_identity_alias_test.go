package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
)

func TestManagedIdentityAliasWithSourceID(t *testing.T) {
	createName := "tmi_test_alias"
	createAliasRootGroupPath := "provider-test-managed-identity-alias-group"
	createResourcePath := createAliasRootGroupPath + "/" + createName

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and read back a managed identity alias.
			{
				Config: testSharedProviderConfiguration() + testManagedIdentityAliasConfiguration("tmi_aws", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_managed_identity_alias.tmi_alias", "resource_path", createResourcePath),
					resource.TestCheckResourceAttr("tharsis_managed_identity_alias.tmi_alias", "name", createName),
					resource.TestCheckResourceAttr("tharsis_managed_identity_alias.tmi_alias", "group_path", createAliasRootGroupPath),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_managed_identity_alias.tmi_alias", "id"),
					resource.TestCheckResourceAttrSet("tharsis_managed_identity_alias.tmi_alias", "last_updated"),
					resource.TestCheckResourceAttrSet("tharsis_managed_identity_alias.tmi_alias", "alias_source_id"),
				),
			},

			// Import state by GID.
			// With no ImportStateId set, the framework reuses the id already in
			// state, which is the GID returned by the API during create.
			{
				ResourceName:      "tharsis_managed_identity_alias.tmi_alias",
				ImportState:       true,
				ImportStateVerify: true,
			},

			// Import state by TRN. Aliases are managed identities on the API
			// side, so they carry the managed identity TRN type.
			{
				ResourceName:      "tharsis_managed_identity_alias.tmi_alias",
				ImportStateId:     "trn:managed_identity:" + createResourcePath,
				ImportState:       true,
				ImportStateVerify: true,
			},

			// Update (which requires replacement) and read back.
			{
				// Update and read back a managed identity alias (doesn't change fields).
				Config: testSharedProviderConfiguration() + testManagedIdentityAliasConfiguration("tmi_azure", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_managed_identity_alias.tmi_alias", "resource_path", createResourcePath),
					resource.TestCheckResourceAttr("tharsis_managed_identity_alias.tmi_alias", "name", createName),
					resource.TestCheckResourceAttr("tharsis_managed_identity_alias.tmi_alias", "group_path", createAliasRootGroupPath),

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

func TestManagedIdentityAliasWithSourcePath(t *testing.T) {
	createName := "tmi_test_alias"
	createAliasRootGroupPath := "provider-test-managed-identity-alias-group"
	createResourcePath := createAliasRootGroupPath + "/" + createName

	resource.Test(t, resource.TestCase{

		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{

			// Create and read back a managed identity alias.
			{
				Config: testSharedProviderConfiguration() + testManagedIdentityAliasConfiguration("tmi_aws", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_managed_identity_alias.tmi_alias", "resource_path", createResourcePath),
					resource.TestCheckResourceAttr("tharsis_managed_identity_alias.tmi_alias", "name", createName),
					resource.TestCheckResourceAttr("tharsis_managed_identity_alias.tmi_alias", "group_path", createAliasRootGroupPath),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_managed_identity_alias.tmi_alias", "id"),
					resource.TestCheckResourceAttrSet("tharsis_managed_identity_alias.tmi_alias", "last_updated"),
					resource.TestCheckResourceAttrSet("tharsis_managed_identity_alias.tmi_alias", "alias_source_id"),
				),
			},

			// Import state by GID.
			// With no ImportStateId set, the framework reuses the id already in
			// state, which is the GID returned by the API during create.
			//
			// alias_source_path is ignored here because the API identifies the
			// alias source by ID rather than path, so the path is not available
			// to compare against.
			{
				ResourceName:            "tharsis_managed_identity_alias.tmi_alias",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"alias_source_path"},
			},

			// Import state by TRN. Aliases are managed identities on the API
			// side, so they carry the managed identity TRN type.
			{
				ResourceName:            "tharsis_managed_identity_alias.tmi_alias",
				ImportStateId:           "trn:managed_identity:" + createResourcePath,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"alias_source_path"},
			},

			// Update (which requires replacement) and read back.
			{
				// Update and read back a managed identity alias (doesn't change fields).
				Config: testSharedProviderConfiguration() + testManagedIdentityAliasConfiguration("tmi_azure", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_managed_identity_alias.tmi_alias", "resource_path", createResourcePath),
					resource.TestCheckResourceAttr("tharsis_managed_identity_alias.tmi_alias", "name", createName),
					resource.TestCheckResourceAttr("tharsis_managed_identity_alias.tmi_alias", "group_path", createAliasRootGroupPath),

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

// TestManagedIdentityAliasSourceFormSwitch: naming the same alias source by its ID, its TRN or
// alias_source_path is an in-place update with no write calls, and a different source replaces.
func TestManagedIdentityAliasSourceFormSwitch(t *testing.T) {
	const addr = "tharsis_managed_identity_alias.tmi_alias"
	sourcePath := testGroupPath + "/tmi_aws_name"
	var id string
	calls := newWriteCallLog(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testSharedProviderConfiguration() + testManagedIdentityAliasSourceConfiguration("alias_source_id = tharsis_managed_identity.tmi_aws.id"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCaptureAttr(addr, "id", &id),
					calls.mark(),
				),
			},
			{
				Config: testSharedProviderConfiguration() + testManagedIdentityAliasSourceConfiguration("alias_source_id = "+quoted("trn:managed_identity:"+sourcePath)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckAttrUnchanged(addr, "id", &id),
					resource.TestCheckResourceAttr(addr, "alias_source_id", "trn:managed_identity:"+sourcePath),
					calls.checkNoneSinceMark(),
				),
			},
			{
				Config: testSharedProviderConfiguration() + testManagedIdentityAliasSourceConfiguration("alias_source_path = "+quoted(sourcePath)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckAttrUnchanged(addr, "id", &id),
					resource.TestCheckResourceAttr(addr, "alias_source_path", sourcePath),
					resource.TestCheckResourceAttr(addr, "alias_source_id", "trn:managed_identity:"+sourcePath),
					calls.checkNoneSinceMark(),
				),
			},
			{
				Config: testSharedProviderConfiguration() + testManagedIdentityAliasSourceConfiguration("alias_source_id = tharsis_managed_identity.tmi_aws.id"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckAttrUnchanged(addr, "id", &id),
					resource.TestCheckResourceAttrPair(addr, "alias_source_id", "tharsis_managed_identity.tmi_aws", "id"),
					resource.TestCheckNoResourceAttr(addr, "alias_source_path"),
					calls.checkNoneSinceMark(),
				),
			},
			{
				Config: testSharedProviderConfiguration() + testManagedIdentityAliasSourceConfiguration("alias_source_path = "+quoted(testGroupPath+"/tmi_azure_name")),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckAttrChanged(addr, "id", &id),
					resource.TestCheckResourceAttrPair(addr, "alias_source_id", "tharsis_managed_identity.tmi_azure", "id"),
				),
			},
		},
	})
}

func testManagedIdentityAliasConfiguration(sourceName string, withPath bool) string {
	// Decide whether we're using path or ID for the alias source.
	aliasSource := fmt.Sprintf("alias_source_id = tharsis_managed_identity.%s.id", sourceName)
	if withPath {
		aliasSource = fmt.Sprintf("alias_source_path = tharsis_managed_identity.%s.resource_path", sourceName)
	}
	return testManagedIdentityAliasSourceConfiguration(aliasSource)
}

// testManagedIdentityAliasSourceConfiguration names the alias source with the given line. A
// literal TRN or path carries no dependency, so depends_on keeps the sources created first.
func testManagedIdentityAliasSourceConfiguration(aliasSource string) string {
	sourceIdentityAWSType := pb.ManagedIdentityType_aws_federated.String()
	sourceIdentityAWSName := "tmi_aws_name"
	sourceIdentityAWSDescription := "this is tmi_aws, a Tharsis managed identity of AWS type"
	sourceIdentityAWSRole := "some-iam-role"

	sourceIdentityAzureType := pb.ManagedIdentityType_azure_federated.String()
	sourceIdentityAzureName := "tmi_azure_name"
	sourceIdentityAzureDescription := "this is tmi_azure, a Tharsis managed identity of Azure type"
	sourceIdentityAzureClient := "some-azure-client"
	sourceIdentityAzureTenant := "some-azure-tenant"

	// Managed identity alias must be created under a different namespace.
	createAliasRootGroupPath := "provider-test-managed-identity-alias-group"
	createAliasRootGroupDescription := "this is a test root group for managed identity alias"

	// Alias fields.
	createAliasName := "tmi_test_alias"

	return fmt.Sprintf(`

%s

resource "tharsis_managed_identity" "tmi_aws" {
	type        = "%s"
	name        = "%s"
	description = "%s"
	group_path  = tharsis_group.root-group.full_path
	aws_role    = "%s"
}

resource "tharsis_managed_identity" "tmi_azure" {
	type            = "%s"
	name            = "%s"
	description     = "%s"
	group_path      = tharsis_group.root-group.full_path
	azure_client_id = "%s"
	azure_tenant_id = "%s"
}

resource "tharsis_group" "alias-group" {
	name = "%s"
	description = "%s"
}

resource "tharsis_managed_identity_alias" "tmi_alias" {
	name = "%s"
	group_path = tharsis_group.alias-group.full_path
	%s
	depends_on = [tharsis_managed_identity.tmi_aws, tharsis_managed_identity.tmi_azure]
}

	`, createRootGroup(testGroupPath, "this is a test root group"),
		sourceIdentityAWSType,
		sourceIdentityAWSName,
		sourceIdentityAWSDescription,
		sourceIdentityAWSRole,
		sourceIdentityAzureType,
		sourceIdentityAzureName,
		sourceIdentityAzureDescription,
		sourceIdentityAzureClient,
		sourceIdentityAzureTenant,
		createAliasRootGroupPath,
		createAliasRootGroupDescription,
		createAliasName,
		aliasSource,
	)
}
