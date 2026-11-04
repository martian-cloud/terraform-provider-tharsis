package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	sdkresource "github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/client"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// A workspace VCS provider link cannot be created here: it needs a completed VCS OAuth
// flow. Its two references, vcs_provider_id and workspace_id, go through the same helpers
// as the references TestTRNReferenceMatrix covers end to end. This test creates a real VCS
// provider and workspace, then calls the link's ModifyPlan and the Read-side helper
// directly against the API, covering the only lookups those references add.
func TestTRNReferenceLinkHelpers(t *testing.T) {
	var vcsGID, wsGID, wsGID2 string
	vcsTRN := "trn:vcs_provider:" + trnRefGroupPath + "/trn_vcs"
	wsTRN := "trn:workspace:" + trnRefGroupPath + "/trn_ws"

	cfg := trnRefBase("ws") + `
resource "tharsis_vcs_provider" "trn_vcs" {
	name                 = "trn_vcs"
	description          = "VCS provider for the TRN reference tests"
	url                  = "https://vcs-provider.domain"
	group_path           = tharsis_group.root-group.full_path
	type                 = "gitlab"
	auto_create_webhooks = false
	oauth_client_id      = "trn-oauth-client-id"
	oauth_client_secret  = "trn-oauth-client-secret" # nosec G101 -- test value
}`

	sdkresource.Test(t, sdkresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []sdkresource.TestStep{
			{
				Config: cfg,
				Check: sdkresource.ComposeAggregateTestCheckFunc(
					testCaptureAttr("tharsis_vcs_provider.trn_vcs", "id", &vcsGID),
					testCaptureAttr("tharsis_workspace.trn_ws", "id", &wsGID),
					testCaptureAttr("tharsis_workspace.trn_ws2", "id", &wsGID2),
					func(*terraform.State) error {
						checkLinkHelpers(t, vcsGID, vcsTRN, wsGID, wsTRN, wsGID2)
						return nil
					},
				),
			},
		},
	})
}

func checkLinkHelpers(t *testing.T, vcsGID, vcsTRN, wsGID, wsTRN, wsGID2 string) {
	t.Helper()
	ctx := context.Background()
	c := testTharsisClient(ctx, t)

	// Lookup by TRN and by Global ID, with the expected type.
	for _, ref := range []string{vcsTRN, vcsGID} {
		if got, err := resolveObjectGID(ctx, c, ref, trn.TypeVCSProvider); err != nil || got != vcsGID {
			t.Errorf("resolveObjectGID(%q) = %q, %v; want %q", ref, got, err, vcsGID)
		}
	}
	if _, err := resolveObjectGID(ctx, c, "trn:vcs_provider:"+trnRefGroupPath+"/missing", trn.TypeVCSProvider); status.Code(err) != codes.NotFound {
		t.Errorf("missing VCS provider: got %v, want NotFound", err)
	}

	// Read side: a stored TRN naming the API's object is kept; anything else is drift.
	for _, tc := range []struct {
		stored, apiGID, want string
	}{
		{vcsTRN, vcsGID, vcsTRN},
		{vcsGID, vcsGID, vcsGID},
		{vcsTRN, "VlBfbm90LXRoaXMtb25l", "VlBfbm90LXRoaXMtb25l"},
		{"trn:vcs_provider:" + trnRefGroupPath + "/missing", vcsGID, vcsGID},
	} {
		got := keepOrRefreshResolve(ctx, c, types.StringValue(tc.stored), tc.apiGID, trn.TypeVCSProvider).ValueString()
		if got != tc.want {
			t.Errorf("keepOrRefreshResolve(%q, api %q) = %q, want %q", tc.stored, tc.apiGID, got, tc.want)
		}
	}

	// Plan side, through the link's own ModifyPlan.
	link := &workspaceVCSProviderLinkResource{client: c}
	for _, tc := range []struct {
		name              string
		priorVCS, planVCS string
		priorWS, planWS   string
		wantReplace       []string
	}{
		{"same objects, other forms", vcsGID, vcsTRN, wsGID, wsTRN, nil},
		{"different workspace", vcsGID, vcsGID, wsGID, wsGID2, []string{"workspace_id"}},
		{"VCS provider gone", vcsGID, "trn:vcs_provider:" + trnRefGroupPath + "/missing", wsGID, wsGID, []string{"vcs_provider_id"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req, resp := linkPlanRequest(t, link, tc.priorVCS, tc.planVCS, tc.priorWS, tc.planWS)
			link.ModifyPlan(ctx, req, resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("diagnostics: %v", resp.Diagnostics)
			}
			var got []string
			for _, p := range resp.RequiresReplace {
				got = append(got, p.String())
			}
			if len(got) != len(tc.wantReplace) || (len(got) > 0 && got[0] != tc.wantReplace[0]) {
				t.Fatalf("RequiresReplace = %v, want %v", got, tc.wantReplace)
			}
		})
	}
}

// linkPlanRequest builds a ModifyPlan request for a link whose only set attributes are its
// two references, using the resource's real schema.
func linkPlanRequest(t *testing.T, r *workspaceVCSProviderLinkResource, priorVCS, planVCS, priorWS, planWS string,
) (resource.ModifyPlanRequest, *resource.ModifyPlanResponse) {
	t.Helper()
	ctx := context.Background()
	var sresp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sresp)
	s := sresp.Schema
	objType := s.Type().TerraformType(ctx).(tftypes.Object)

	build := func(vcs, ws string) tftypes.Value {
		vals := map[string]tftypes.Value{}
		for name, at := range objType.AttributeTypes {
			vals[name] = tftypes.NewValue(at, nil)
		}
		vals["vcs_provider_id"] = tftypes.NewValue(tftypes.String, vcs)
		vals["workspace_id"] = tftypes.NewValue(tftypes.String, ws)
		return tftypes.NewValue(objType, vals)
	}
	req := resource.ModifyPlanRequest{
		State: tfsdk.State{Schema: s, Raw: build(priorVCS, priorWS)},
		Plan:  tfsdk.Plan{Schema: s, Raw: build(planVCS, planWS)},
	}
	return req, &resource.ModifyPlanResponse{Plan: req.Plan}
}

// testTharsisClient builds an API client the same way the provider does, from the
// THARSIS_ENDPOINT and THARSIS_STATIC_TOKEN acceptance-test environment.
func testTharsisClient(ctx context.Context, t *testing.T) *client.GRPCClient {
	t.Helper()
	c, err := newTharsisClient(ctx, &providerData{})
	if err != nil {
		t.Fatalf("creating a Tharsis client: %v", err)
	}
	return c
}
