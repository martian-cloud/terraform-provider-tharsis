package provider

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// TestManagedIdentityValidateConfig checks the per-type data rules at plan time. Values
// unknown until apply are not checked.
func TestManagedIdentityValidateConfig(t *testing.T) {
	ctx := context.Background()
	r := &managedIdentityResource{}
	var sresp fwresource.SchemaResponse
	r.Schema(ctx, fwresource.SchemaRequest{}, &sresp)
	objType := sresp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	unknown := tftypes.NewValue(tftypes.String, tftypes.UnknownValue)

	for _, tc := range []struct {
		name     string
		fields   map[string]any
		wantAttr string // "" for no error
	}{
		{"aws valid", map[string]any{"type": "aws_federated", "aws_role": "r"}, ""},
		{"aws missing role", map[string]any{"type": "aws_federated"}, "aws_role"},
		{"aws empty role", map[string]any{"type": "aws_federated", "aws_role": ""}, "aws_role"},
		{"aws with client id", map[string]any{"type": "aws_federated", "aws_role": "r", "azure_client_id": "c"}, "azure_client_id"},
		{"aws with tenant id", map[string]any{"type": "aws_federated", "aws_role": "r", "azure_tenant_id": "t"}, "azure_tenant_id"},
		{"azure valid", map[string]any{"type": "azure_federated", "azure_client_id": "c", "azure_tenant_id": "t"}, ""},
		{"azure missing client id", map[string]any{"type": "azure_federated", "azure_tenant_id": "t"}, "azure_client_id"},
		{"azure missing tenant id", map[string]any{"type": "azure_federated", "azure_client_id": "c"}, "azure_tenant_id"},
		{"azure with role", map[string]any{"type": "azure_federated", "aws_role": "r", "azure_client_id": "c", "azure_tenant_id": "t"}, "aws_role"},
		{"tharsis valid", map[string]any{"type": "tharsis_federated", "tharsis_service_account_path": "g/sa"}, ""},
		{"tharsis missing path", map[string]any{"type": "tharsis_federated"}, "tharsis_service_account_path"},
		{"kubernetes valid", map[string]any{"type": "kubernetes_federated", "audience": "a"}, ""},
		{"kubernetes missing audience", map[string]any{"type": "kubernetes_federated"}, "audience"},
		{"unknown role", map[string]any{"type": "aws_federated", "aws_role": unknown}, ""},
		{"unknown type", map[string]any{"type": unknown}, ""},
		{"invalid type left to the type validator", map[string]any{"type": "invalid_type"}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			vals := map[string]tftypes.Value{}
			for name, at := range objType.AttributeTypes {
				vals[name] = tftypes.NewValue(at, nil)
			}
			for k, v := range tc.fields {
				if tv, ok := v.(tftypes.Value); ok {
					vals[k] = tv
				} else {
					vals[k] = tftypes.NewValue(tftypes.String, v)
				}
			}
			req := fwresource.ValidateConfigRequest{Config: tfsdk.Config{Schema: sresp.Schema, Raw: tftypes.NewValue(objType, vals)}}
			resp := &fwresource.ValidateConfigResponse{}
			r.ValidateConfig(ctx, req, resp)
			if tc.wantAttr == "" {
				if resp.Diagnostics.HasError() {
					t.Fatalf("unexpected error: %v", resp.Diagnostics)
				}
				return
			}
			if resp.Diagnostics.ErrorsCount() != 1 {
				t.Fatalf("want one error on %s, got %v", tc.wantAttr, resp.Diagnostics)
			}
			d, ok := resp.Diagnostics.Errors()[0].(diag.DiagnosticWithPath)
			if !ok || !d.Path().Equal(path.Root(tc.wantAttr)) {
				t.Fatalf("want error on %s, got %v", tc.wantAttr, resp.Diagnostics)
			}
		})
	}
}

// TestManagedIdentityInvalidTypeChangeKeepsIdentity: changing type, which replaces the
// identity, with a field the new type needs missing used to delete the old identity before
// the create failed. It must now fail at plan, and the original identity must still be there.
func TestManagedIdentityInvalidTypeChangeKeepsIdentity(t *testing.T) {
	const addr = "tharsis_managed_identity.tmi_keep"
	var id string
	config := func(body string) string {
		return testSharedProviderConfiguration() + createRootGroup(testGroupPath, "this is a test root group") + fmt.Sprintf(`
resource "tharsis_managed_identity" "tmi_keep" {
	name     = "tmi_keep"
	group_id = tharsis_group.root-group.id
	%s
}
`, body)
	}
	aws := "type     = \"aws_federated\"\n\taws_role = \"arn:aws:iam::123456789012:role/r\""
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config(aws),
				Check:  testCaptureAttr(addr, "id", &id),
			},
			{
				Config:      config("type = \"azure_federated\""),
				ExpectError: regexp.MustCompile(`(?s)Invalid managed identity configuration.*client ID is required`),
			},
			{
				Config: config(aws),
				Check:  testCheckAttrUnchanged(addr, "id", &id),
			},
		},
	})
}
