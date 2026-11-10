package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
)

// TestUnsetOptionalAttributes leaves the managed identity's description and the module's and
// provider's repository_url and private unset. Tharsis returns "" and false for them, which
// once failed the apply with "Provider produced inconsistent result after apply" and left the
// resource tainted. The test also sets each value, then removes it again: removing a value
// from the configuration must clear it in Tharsis, not keep the old one.
//
// resource.Test runs a plan after every step and fails on a non-empty one, so it also proves
// no perpetual diff.
func TestUnsetOptionalAttributes(t *testing.T) {
	const (
		mi  = "tharsis_managed_identity.tuoa_mi"
		mod = "tharsis_terraform_module.tuoa_mod"
		prv = "tharsis_terraform_provider.tuoa_prv"
	)
	unset := resource.ComposeAggregateTestCheckFunc(
		resource.TestCheckResourceAttr(mi, "description", ""),
		resource.TestCheckResourceAttr(mod, "repository_url", ""),
		resource.TestCheckResourceAttr(mod, "private", "false"),
		resource.TestCheckResourceAttr(prv, "repository_url", ""),
		resource.TestCheckResourceAttr(prv, "private", "false"),
	)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Created with all five unset.
			{Config: testUnsetOptionalAttributesConfiguration(false), Check: unset},
			// Set them.
			{
				Config: testUnsetOptionalAttributesConfiguration(true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(mi, "description", "set"),
					resource.TestCheckResourceAttr(mod, "repository_url", "https://example.invalid/module"),
					resource.TestCheckResourceAttr(mod, "private", "true"),
					resource.TestCheckResourceAttr(prv, "repository_url", "https://example.invalid/provider"),
					resource.TestCheckResourceAttr(prv, "private", "true"),
				),
			},
			// Removed again: back to Tharsis's defaults.
			{Config: testUnsetOptionalAttributesConfiguration(false), Check: unset},
		},
	})
}

func testUnsetOptionalAttributesConfiguration(set bool) string {
	miExtra, modExtra, prvExtra := "", "", ""
	if set {
		miExtra = `description = "set"`
		modExtra = "repository_url = \"https://example.invalid/module\"\n\tprivate        = true"
		prvExtra = "repository_url = \"https://example.invalid/provider\"\n\tprivate        = true"
	}
	return createRootGroup(testGroupPath, "this is a test root group") + fmt.Sprintf(`

resource "tharsis_managed_identity" "tuoa_mi" {
	type       = "%s"
	name       = "tuoa_mi"
	group_path = tharsis_group.root-group.full_path
	aws_role   = "some-iam-role"
	%s
}

resource "tharsis_terraform_module" "tuoa_mod" {
	name       = "tuoa-mod"
	system     = "aws"
	group_path = tharsis_group.root-group.full_path
	%s
}

resource "tharsis_terraform_provider" "tuoa_prv" {
	name       = "tuoa-prv"
	group_path = tharsis_group.root-group.full_path
	%s
}
`, pb.ManagedIdentityType_aws_federated.String(), miExtra, modExtra, prvExtra)
}
