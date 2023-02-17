package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestGPGKey(t *testing.T) {

	// Use a made-up GPG key.
	createASCIIArmor := `-----BEGIN PGP PUBLIC KEY BLOCK-----

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
=HJT6
-----END PGP PUBLIC KEY BLOCK-----
`
	createFingerprint := "C48F7F19B14E59433DF9402341F57AB6741F92F1"
	createGPGKeyID := "41F57AB6741F92F1"
	createGroupPath := testGroupPath

	resource.Test(t, resource.TestCase{

		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{

			// Create and read back a GPG key.
			{
				Config: testGPGKeyConfigurationCreate(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_gpg_key.tgk", "ascii_armor", createASCIIArmor),
					resource.TestCheckResourceAttr("tharsis_gpg_key.tgk", "fingerprint", createFingerprint),
					resource.TestCheckResourceAttr("tharsis_gpg_key.tgk", "gpg_key_id", createGPGKeyID),
					resource.TestCheckResourceAttr("tharsis_gpg_key.tgk", "group_path", createGroupPath),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_gpg_key.tgk", "id"),
					resource.TestCheckResourceAttrSet("tharsis_gpg_key.tgk", "last_updated"),
					resource.TestCheckResourceAttrSet("tharsis_gpg_key.tgk", "created_by"),
				),
			},

			// Import the state.
			{
				ResourceName:      "tharsis_gpg_key.tgk",
				ImportStateId:     "FIXME: What/which ID should this be?", // FIXME: What/which ID should this be?
				ImportState:       true,
				ImportStateVerify: true,
			},

			// Update is allowed _ONLY_ if nothing changes, so use the same configuration as create.
			{
				Config: testGPGKeyConfigurationCreate(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_gpg_key.tgk", "ascii_armor", createASCIIArmor),
					resource.TestCheckResourceAttr("tharsis_gpg_key.tgk", "fingerprint", createFingerprint),
					resource.TestCheckResourceAttr("tharsis_gpg_key.tgk", "gpg_key_id", createGPGKeyID),
					resource.TestCheckResourceAttr("tharsis_gpg_key.tgk", "group_path", createGroupPath),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_gpg_key.tgk", "id"),
					resource.TestCheckResourceAttrSet("tharsis_gpg_key.tgk", "last_updated"),
					resource.TestCheckResourceAttrSet("tharsis_gpg_key.tgk", "created_by"),
				),
			},

			// Destroy should be covered automatically by TestCase.

		},
	})
}

func testGPGKeyConfigurationCreate() string {
	createASCIIArmor := `-----BEGIN PGP PUBLIC KEY BLOCK-----

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
=HJT6
-----END PGP PUBLIC KEY BLOCK-----
`

	// Using %#v for the ASCII armor field to escape the embedded newlines.  It supplies double-quotation marks.
	return fmt.Sprintf(`

%s

resource "tharsis_gpg_key" "tgk" {
	ascii_armor = %#v
	group_path = tharsis_group.root-group.full_path
}
	`, createRootGroup(), createASCIIArmor)
}

// The End.
