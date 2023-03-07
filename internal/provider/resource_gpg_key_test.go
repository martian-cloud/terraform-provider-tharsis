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
	createResourcePath := createGroupPath + "/" + createFingerprint

	updateASCIIArmor := `-----BEGIN PGP PUBLIC KEY BLOCK-----

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
=/yb1
-----END PGP PUBLIC KEY BLOCK-----
`
	updateFingerprint := "6A8C05A6A0D3FE76B794E81B2585A0CEA26DFCF7"
	updateGPGKeyID := "2585A0CEA26DFCF7"
	updateGroupPath := testGroupPath
	updateResourcePath := updateGroupPath + "/" + updateFingerprint

	resource.Test(t, resource.TestCase{

		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{

			// Create and read back a GPG key.
			{
				Config: testGPGKeyConfiguration(createASCIIArmor),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_gpg_key.tgk", "ascii_armor", createASCIIArmor),
					resource.TestCheckResourceAttr("tharsis_gpg_key.tgk", "fingerprint", createFingerprint),
					resource.TestCheckResourceAttr("tharsis_gpg_key.tgk", "gpg_key_id", createGPGKeyID),
					resource.TestCheckResourceAttr("tharsis_gpg_key.tgk", "group_path", createGroupPath),
					resource.TestCheckResourceAttr("tharsis_gpg_key.tgk", "resource_path", createResourcePath),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("tharsis_gpg_key.tgk", "id"),
					resource.TestCheckResourceAttrSet("tharsis_gpg_key.tgk", "last_updated"),
					resource.TestCheckResourceAttrSet("tharsis_gpg_key.tgk", "created_by"),
				),
			},

			// Import the state.
			{
				ResourceName:      "tharsis_gpg_key.tgk",
				ImportState:       true,
				ImportStateVerify: true,
			},

			// Update (which requires replacement) and read back.
			{
				Config: testGPGKeyConfiguration(updateASCIIArmor),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_gpg_key.tgk", "ascii_armor", updateASCIIArmor),
					resource.TestCheckResourceAttr("tharsis_gpg_key.tgk", "fingerprint", updateFingerprint),
					resource.TestCheckResourceAttr("tharsis_gpg_key.tgk", "gpg_key_id", updateGPGKeyID),
					resource.TestCheckResourceAttr("tharsis_gpg_key.tgk", "group_path", updateGroupPath),
					resource.TestCheckResourceAttr("tharsis_gpg_key.tgk", "resource_path", updateResourcePath),

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

func testGPGKeyConfiguration(asciiArmor string) string {

	// Using %#v for the ASCII armor field to escape the embedded newlines.  It supplies double-quotation marks.
	return fmt.Sprintf(`

%s

resource "tharsis_gpg_key" "tgk" {
	ascii_armor = %#v
	group_path = tharsis_group.root-group.full_path
}
	`, createRootGroup(), asciiArmor)
}

// The End.
