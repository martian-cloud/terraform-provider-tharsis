package provider

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
)

// newTestImportState builds an empty single-attribute state suitable for
// exercising an ImportState implementation in a unit test.
func newTestImportState() tfsdk.State {
	return tfsdk.State{
		Schema: schema.Schema{
			Attributes: map[string]schema.Attribute{
				"id": schema.StringAttribute{Computed: true},
			},
		},
		Raw: tftypes.NewValue(
			tftypes.Object{AttributeTypes: map[string]tftypes.Type{"id": tftypes.String}},
			map[string]tftypes.Value{"id": tftypes.NewValue(tftypes.String, nil)},
		),
	}
}

// testGID returns a syntactically valid GID for the supplied type code. A GID is
// the base64url encoding of "<code>_<uuid>", which is the shape trn.Normalize
// uses to tell GIDs apart from bare resource paths.
func testGID(code string) string {
	return base64.RawURLEncoding.EncodeToString(
		[]byte(code + "_1f8e5a92-7c41-4d6b-9b3a-2e5d8c7f1a04"),
	)
}

func TestImportByNormalizedID(t *testing.T) {
	groupGID := testGID("group")
	workspaceGID := testGID("workspace")

	tests := []struct {
		name     string
		trnType  trn.Type
		importID string
		want     string
	}{
		{
			name:     "bare single-segment group path becomes a TRN",
			trnType:  trn.TypeGroup,
			importID: "my-group",
			want:     "trn:group:my-group",
		},
		{
			name:     "bare nested group path becomes a TRN",
			trnType:  trn.TypeGroup,
			importID: "my-group/sub-group",
			want:     "trn:group:my-group/sub-group",
		},
		{
			name:     "group GID passes through unchanged",
			trnType:  trn.TypeGroup,
			importID: groupGID,
			want:     groupGID,
		},
		{
			name:     "group TRN passes through unchanged",
			trnType:  trn.TypeGroup,
			importID: "trn:group:my-group/sub-group",
			want:     "trn:group:my-group/sub-group",
		},
		{
			name:     "workspace path becomes a TRN",
			trnType:  trn.TypeWorkspace,
			importID: "my-group/my-workspace",
			want:     "trn:workspace:my-group/my-workspace",
		},
		{
			name:     "workspace GID passes through unchanged",
			trnType:  trn.TypeWorkspace,
			importID: workspaceGID,
			want:     workspaceGID,
		},
		{
			name:     "managed identity path becomes a TRN",
			trnType:  trn.TypeManagedIdentity,
			importID: "my-group/my-identity",
			want:     "trn:managed_identity:my-group/my-identity",
		},
		{
			name:     "terraform module path with system becomes a TRN",
			trnType:  trn.TypeTerraformModule,
			importID: "my-group/my-module/aws",
			want:     "trn:terraform_module:my-group/my-module/aws",
		},
		{
			name:     "variable path with category and key becomes a TRN",
			trnType:  trn.TypeVariable,
			importID: "my-group/my-workspace/terraform/my-key",
			want:     "trn:variable:my-group/my-workspace/terraform/my-key",
		},
		{
			name:     "service account path becomes a TRN",
			trnType:  trn.TypeServiceAccount,
			importID: "my-group/my-service-account",
			want:     "trn:service_account:my-group/my-service-account",
		},
		{
			name:     "a TRN of a different type is not rewritten",
			trnType:  trn.TypeWorkspace,
			importID: "trn:group:my-group",
			want:     "trn:group:my-group",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			req := resource.ImportStateRequest{ID: test.importID}
			resp := &resource.ImportStateResponse{State: newTestImportState()}

			importByNormalizedID(ctx, test.trnType, req, resp)

			if resp.Diagnostics.HasError() {
				t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
			}

			var got types.String
			if diags := resp.State.GetAttribute(ctx, path.Root("id"), &got); diags.HasError() {
				t.Fatalf("failed to read id back from state: %v", diags)
			}

			if got.ValueString() != test.want {
				t.Errorf("id = %q, want %q", got.ValueString(), test.want)
			}
		})
	}
}
