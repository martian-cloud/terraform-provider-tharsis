package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const assignedManagedIdentityResourceName = "tharsis_assigned_managed_identity.tami1"

// TestAssignedManagedIdentity tests creation, reading, importing, updating, and deletion of an
// assigned managed identity resource, also known as assigning and unassigning a managed identity
// to/from a workspace.
func TestAssignedManagedIdentity(t *testing.T) {
	resource.Test(t, resource.TestCase{
		// Tharsis assigned managed identities
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and read back an assigned managed identity.
			{
				Config: testSharedProviderConfiguration() + testAssignedManagedIdentityConfiguration(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// There are no values in an assigned managed identity that are known.

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet(assignedManagedIdentityResourceName, "managed_identity_id"),
					resource.TestCheckResourceAttrSet(assignedManagedIdentityResourceName, "workspace_id"),
				),
			},

			// Import the state using the composite "<workspace_id>/<managed_identity_id>" ID.
			//
			// This step uses ImportStateCheck rather than ImportStateVerify.
			// Verification matches the imported resource to the existing one by
			// its id, and this resource's id is a UUID the provider mints for
			// Terraform's benefit rather than a value Tharsis stores, so the
			// imported id never equals the one created earlier. The two
			// attributes that do identify the assignment are checked instead.
			{
				ResourceName:      assignedManagedIdentityResourceName,
				ImportState:       true,
				ImportStateIdFunc: testAssignedManagedIdentityImportID,
				ImportStateCheck:  testAssignedManagedIdentityImportStateCheck,
			},

			// Update (which requires replacement) and read back.
			{
				Config: testAssignedManagedIdentityConfiguration(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// There are no values in an assigned managed identity that are known.

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet(assignedManagedIdentityResourceName, "managed_identity_id"),
					resource.TestCheckResourceAttrSet(assignedManagedIdentityResourceName, "workspace_id"),
				),
			},

			// Destroy should be covered automatically by TestCase.
		},
	})
}

// testAssignedManagedIdentityImportID builds the composite import ID from the
// workspace and managed identity GIDs recorded in state, neither of which is
// known until the resources have been created.
func testAssignedManagedIdentityImportID(state *terraform.State) (string, error) {
	res, ok := state.RootModule().Resources[assignedManagedIdentityResourceName]
	if !ok {
		return "", fmt.Errorf("resource %s not found in state", assignedManagedIdentityResourceName)
	}

	workspaceID := res.Primary.Attributes["workspace_id"]
	managedIdentityID := res.Primary.Attributes["managed_identity_id"]
	if workspaceID == "" || managedIdentityID == "" {
		return "", fmt.Errorf("workspace_id and managed_identity_id must both be set in state, got %q and %q",
			workspaceID, managedIdentityID)
	}

	return workspaceID + "/" + managedIdentityID, nil
}

// testAssignedManagedIdentityImportStateCheck asserts that importing produced a
// single assignment carrying both halves of the composite identifier, and that
// the generated id is present.
func testAssignedManagedIdentityImportStateCheck(states []*terraform.InstanceState) error {
	if len(states) != 1 {
		return fmt.Errorf("expected one imported instance, got %d", len(states))
	}

	attrs := states[0].Attributes

	for _, name := range []string{"workspace_id", "managed_identity_id", "id"} {
		if attrs[name] == "" {
			return fmt.Errorf("imported state is missing %s", name)
		}
	}

	// The import ID is "<workspace_id>/<managed_identity_id>", so the two
	// attributes must reconstruct the identifier the import was given.
	wantID := attrs["workspace_id"] + "/" + attrs["managed_identity_id"]
	if states[0].ID == "" {
		return fmt.Errorf("imported state has an empty id, expected a generated one for %s", wantID)
	}

	return nil
}

// TestParseAssignedManagedIdentityImportID covers the composite import ID parser
// without needing a Tharsis instance.
func TestParseAssignedManagedIdentityImportID(t *testing.T) {
	workspaceGID := testGID("workspace")
	managedIdentityGID := testGID("managed_identity")

	tests := []struct {
		name                  string
		importID              string
		wantWorkspaceID       string
		wantManagedIdentityID string
		wantErr               bool
	}{
		{
			name:                  "a pair of GIDs separated by a forward slash",
			importID:              workspaceGID + "/" + managedIdentityGID,
			wantWorkspaceID:       workspaceGID,
			wantManagedIdentityID: managedIdentityGID,
		},
		{
			name:     "a single GID with no separator",
			importID: workspaceGID,
			wantErr:  true,
		},
		{
			name:     "three parts",
			importID: workspaceGID + "/" + managedIdentityGID + "/extra",
			wantErr:  true,
		},
		{
			name:     "an empty string",
			importID: "",
			wantErr:  true,
		},
		{
			name:     "only a separator",
			importID: "/",
			wantErr:  true,
		},
		{
			name:     "an empty workspace ID",
			importID: "/" + managedIdentityGID,
			wantErr:  true,
		},
		{
			name:     "an empty managed identity ID",
			importID: workspaceGID + "/",
			wantErr:  true,
		},
		{
			name:     "a pair of TRNs, which cannot be split unambiguously",
			importID: "trn:workspace:my-group/my-workspace/trn:managed_identity:my-group/my-identity",
			wantErr:  true,
		},
		{
			name:     "a single TRN",
			importID: "trn:workspace:my-group/my-workspace",
			wantErr:  true,
		},
		{
			name:     "a workspace TRN paired with a managed identity GID",
			importID: "trn:workspace:my-group/my-workspace/" + managedIdentityGID,
			wantErr:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			workspaceID, managedIdentityID, err := parseAssignedManagedIdentityImportID(test.importID)

			if test.wantErr {
				if err == nil {
					t.Fatalf("expected an error for %q, got workspace %q and managed identity %q",
						test.importID, workspaceID, managedIdentityID)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error for %q: %v", test.importID, err)
			}
			if workspaceID != test.wantWorkspaceID {
				t.Errorf("workspace ID = %q, want %q", workspaceID, test.wantWorkspaceID)
			}
			if managedIdentityID != test.wantManagedIdentityID {
				t.Errorf("managed identity ID = %q, want %q", managedIdentityID, test.wantManagedIdentityID)
			}
		})
	}
}

func testAssignedManagedIdentityConfiguration() string {
	return createRootGroup(testGroupPath, "this is a test root group") +
		`

	resource "tharsis_workspace" "tw" {
		name = "tw_name"
		description = "this is tw, a test workspace"
		group_path = tharsis_group.root-group.full_path
		max_job_duration = "1234"
		terraform_version = "1.2.3"
		prevent_destroy_plan = "true"
	}

	resource "tharsis_managed_identity" "tmi_tharsis" {
		type                         = "tharsis_federated"
		name                         = "tmi_tharsis_name"
		description                  = "this is tmi_tharsis, a Tharsis managed identity of Tharsis type"
		group_path                   = tharsis_group.root-group.full_path
		tharsis_service_account_path = "some-tharsis-service-account-path"
	}

	resource "tharsis_assigned_managed_identity" "tami1" {
		managed_identity_id = tharsis_managed_identity.tmi_tharsis.id
		workspace_id = tharsis_workspace.tw.id
	}

	`
}
