package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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

const (
	unitGroupGID = "R18wMWExYTY4NC03NDE2LTc3ZDUtODMwMC02YWJkMmIyNjg2Nzk"
	unitGroupTRN = "trn:group:some/group"
	unitOtherGID = "R18wMWExYTY4NC03NDE2LTc3ZDUtODMwMC0wMDAwMDAwMDAwMDA"
)

func TestKeepOrRefresh(t *testing.T) {
	cases := []struct {
		name   string
		stored types.String
		apiTRN string
		want   string
	}{
		{"null takes the API Global ID", types.StringNull(), unitGroupTRN, unitGroupGID},
		{"same Global ID is kept", types.StringValue(unitGroupGID), unitGroupTRN, unitGroupGID},
		{"same TRN is kept", types.StringValue(unitGroupTRN), unitGroupTRN, unitGroupTRN},
		{"different Global ID is drift", types.StringValue(unitOtherGID), unitGroupTRN, unitGroupGID},
		{"different TRN is drift", types.StringValue("trn:group:other"), unitGroupTRN, unitGroupGID},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := keepOrRefresh(tc.stored, unitGroupGID, tc.apiTRN).ValueString(); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// Without a reachable API, a TRN cannot be resolved; the stored value must be kept rather
// than reported as drift.
func TestKeepOrRefreshResolveWithoutAPI(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name   string
		stored types.String
		want   string
	}{
		{"null takes the API Global ID", types.StringNull(), unitGroupGID},
		{"same Global ID is kept", types.StringValue(unitGroupGID), unitGroupGID},
		{"different Global ID is drift", types.StringValue(unitOtherGID), unitGroupGID},
		{"unresolvable TRN is kept", types.StringValue(unitGroupTRN), unitGroupTRN},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := keepOrRefreshResolve(ctx, nil, tc.stored, unitGroupGID, trn.TypeGroup).ValueString()
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestResolveObjectGIDRejectsWithoutLookup(t *testing.T) {
	ctx := context.Background()
	if _, err := resolveObjectGID(ctx, nil, unitGroupGID, trn.TypeGroup); err == nil {
		t.Fatal("nil client: expected an error")
	}
	// A configured client with no sub-clients: these must fail before any API call.
	c := &client.GRPCClient{}
	if _, err := resolveObjectGID(ctx, c, "trn:workspace:some/group/ws", trn.TypeGroup); err == nil {
		t.Fatal("TRN of another type: expected an error")
	}
	if _, err := resolveObjectGID(ctx, c, unitGroupGID, trn.TypeTeam); err == nil {
		t.Fatal("unsupported type: expected an error")
	}
}

// With no client, requireReplaceIfDifferentObject cannot prove two different strings name
// the same object, so it must behave like a plain RequiresReplace.
func TestRequireReplaceIfDifferentObjectWithoutAPI(t *testing.T) {
	ctx := context.Background()
	s := schema.Schema{Attributes: map[string]schema.Attribute{"group_id": schema.StringAttribute{Optional: true}}}
	objType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{"group_id": tftypes.String}}
	obj := func(v tftypes.Value) tftypes.Value {
		return tftypes.NewValue(objType, map[string]tftypes.Value{"group_id": v})
	}
	str := func(v string) tftypes.Value { return tftypes.NewValue(tftypes.String, v) }
	null := tftypes.NewValue(objType, nil)

	cases := []struct {
		name        string
		prior, plan tftypes.Value
		wantReplace bool
		wantWarning bool
	}{
		{"create", null, obj(str(unitGroupGID)), false, false},
		{"destroy", obj(str(unitGroupGID)), null, false, false},
		{"unchanged", obj(str(unitGroupGID)), obj(str(unitGroupGID)), false, false},
		{"changed, cannot resolve", obj(str(unitGroupGID)), obj(str(unitGroupTRN)), true, true},
		{"two different Global IDs", obj(str(unitGroupGID)), obj(str(unitGroupGID + "x")), true, false},
		{"planned unknown", obj(str(unitGroupGID)), obj(tftypes.NewValue(tftypes.String, tftypes.UnknownValue)), true, false},
		{"null to a value", obj(tftypes.NewValue(tftypes.String, nil)), obj(str(unitGroupGID)), true, false},
		{"a value to null", obj(str(unitGroupGID)), obj(tftypes.NewValue(tftypes.String, nil)), true, false},
		{"null unchanged", obj(tftypes.NewValue(tftypes.String, nil)), obj(tftypes.NewValue(tftypes.String, nil)), false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := resource.ModifyPlanRequest{
				State: tfsdk.State{Schema: s, Raw: tc.prior},
				Plan:  tfsdk.Plan{Schema: s, Raw: tc.plan},
			}
			resp := &resource.ModifyPlanResponse{Plan: req.Plan}
			requireReplaceIfDifferentObject(ctx, nil, req, resp, path.Root("group_id"), trn.TypeGroup)
			if resp.Diagnostics.HasError() {
				t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
			}
			if got := len(resp.RequiresReplace) > 0; got != tc.wantReplace {
				t.Fatalf("RequiresReplace = %v, want %v", got, tc.wantReplace)
			}
			if got := resp.Diagnostics.WarningsCount() > 0; got != tc.wantWarning {
				t.Fatalf("warning = %v, want %v", got, tc.wantWarning)
			}
		})
	}
}

func TestKeepPlanned(t *testing.T) {
	cases := []struct {
		name    string
		planned types.String
		apiGID  string
		want    types.String
	}{
		{"known TRN is kept", types.StringValue(unitGroupTRN), unitGroupGID, types.StringValue(unitGroupTRN)},
		{"known GID is kept", types.StringValue(unitGroupGID), unitOtherGID, types.StringValue(unitGroupGID)},
		{"unknown takes the API ID", types.StringUnknown(), unitGroupGID, types.StringValue(unitGroupGID)},
		{"null takes the API ID", types.StringNull(), unitGroupGID, types.StringValue(unitGroupGID)},
		{"no reference stays null", types.StringNull(), "", types.StringNull()},
		{"unknown with no reference is null", types.StringUnknown(), "", types.StringNull()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := keepPlanned(tc.planned, tc.apiGID); !got.Equal(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

// A workspace VCS provider link cannot be created here: it needs a completed VCS OAuth
// flow. Its two references, vcs_provider_id and workspace_id, go through the same helpers
// as the references TestTRNReferenceMatrix covers end to end. This test creates a real VCS
// provider and workspace, then calls the link's ModifyPlan and the Read-side helper
// directly against the API, covering the only lookups those references add.
func TestTRNReferenceLinkHelpers(t *testing.T) {
	var vcsGID, wsGID, wsGID2 string
	vcsTRN := "trn:vcs_provider:" + trnRefGroupPath + "/trn_vcs"
	wsTRN := "trn:workspace:" + trnRefGroupPath + "/trn_ws"

	cfg := trnRefBase("ws", "ws2") + `
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

// testFakeVCSURLEnvVar names a GitLab stand-in that the Tharsis API can reach and that
// completes the OAuth code exchange and answers user and project lookups (see
// testdata/fake_vcs/server.py). Without it the end-to-end link test is skipped.
const testFakeVCSURLEnvVar = "THARSIS_TEST_FAKE_VCS_URL"

// TestTRNReferenceLinkEndToEnd runs the workspace VCS provider link's two references
// through the same steps as TestTRNReferenceMatrix, against a VCS provider whose OAuth flow
// is completed against a fake GitLab: create with TRNs, switch both to Global IDs and back
// without replacement and without any write call, then move each to another object.
func TestTRNReferenceLinkEndToEnd(t *testing.T) {
	fakeURL := os.Getenv(testFakeVCSURLEnvVar)
	if fakeURL == "" {
		t.Skipf("set %s to a fake GitLab reachable from the Tharsis API", testFakeVCSURLEnvVar)
	}
	wsTRN := "trn:workspace:" + trnRefGroupPath + "/trn_ws"
	ws2TRN := "trn:workspace:" + trnRefGroupPath + "/trn_ws2"
	vcsTRN := "trn:vcs_provider:" + trnRefGroupPath + "/trn_vcs"
	vcs2TRN := "trn:vcs_provider:" + trnRefGroupPath + "/trn_vcs2"

	providers := ""
	for _, n := range []string{"trn_vcs", "trn_vcs2"} {
		providers += fmt.Sprintf(`
resource "tharsis_vcs_provider" "%[1]s" {
	name                 = "%[1]s"
	description          = "VCS provider for the TRN reference tests"
	url                  = "%[2]s"
	group_path           = tharsis_group.root-group.full_path
	type                 = "gitlab"
	auto_create_webhooks = false
	oauth_client_id      = "trn-oauth-client-id"
	oauth_client_secret  = "trn-oauth-client-secret" # nosec G101 -- test value
}
`, n, fakeURL)
	}
	base := trnRefBase("ws", "ws2") + providers
	link := func(wsRef, vcsRef string) string {
		return base + fmt.Sprintf(`
resource "tharsis_workspace_vcs_provider_link" "trn_target" {
	workspace_id          = %s
	vcs_provider_id       = %s
	repository_path       = "trn/reference"
	module_directory      = "."
	branch                = "main"
	tag_regex             = "^v.*"
	glob_patterns         = ["**/*.tf"]
	auto_speculative_plan = false
	webhook_disabled      = true
	depends_on = [tharsis_workspace.trn_ws, tharsis_workspace.trn_ws2,
		tharsis_vcs_provider.trn_vcs, tharsis_vcs_provider.trn_vcs2]
}`, wsRef, vcsRef)
	}
	const addr = "tharsis_workspace_vcs_provider_link.trn_target"
	var id string
	calls := newWriteCallLog(t)

	sdkresource.Test(t, sdkresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []sdkresource.TestStep{
			{
				// The providers exist; complete their OAuth flows before linking.
				Config: base,
				Check: sdkresource.ComposeAggregateTestCheckFunc(
					testCompleteFakeOAuth("tharsis_vcs_provider.trn_vcs"),
					testCompleteFakeOAuth("tharsis_vcs_provider.trn_vcs2"),
				),
			},
			{
				Config: link(quoted(wsTRN), quoted(vcsTRN)),
				Check: sdkresource.ComposeAggregateTestCheckFunc(
					sdkresource.TestCheckResourceAttr(addr, "workspace_id", wsTRN),
					sdkresource.TestCheckResourceAttr(addr, "vcs_provider_id", vcsTRN),
					testCaptureAttr(addr, "id", &id),
					calls.mark(),
				),
			},
			{
				Config: link("tharsis_workspace.trn_ws.id", "tharsis_vcs_provider.trn_vcs.id"),
				Check: sdkresource.ComposeAggregateTestCheckFunc(
					testCheckAttrUnchanged(addr, "id", &id),
					sdkresource.TestCheckResourceAttrPair(addr, "workspace_id", "tharsis_workspace.trn_ws", "id"),
					sdkresource.TestCheckResourceAttrPair(addr, "vcs_provider_id", "tharsis_vcs_provider.trn_vcs", "id"),
					calls.checkNoneSinceMark(),
				),
			},
			{
				Config: link(quoted(wsTRN), quoted(vcsTRN)),
				Check: sdkresource.ComposeAggregateTestCheckFunc(
					testCheckAttrUnchanged(addr, "id", &id),
					sdkresource.TestCheckResourceAttr(addr, "workspace_id", wsTRN),
					sdkresource.TestCheckResourceAttr(addr, "vcs_provider_id", vcsTRN),
					calls.checkNoneSinceMark(),
				),
			},
			{
				Config: link(quoted(wsTRN), quoted(vcs2TRN)),
				Check: sdkresource.ComposeAggregateTestCheckFunc(
					testCheckAttrChanged(addr, "id", &id),
					sdkresource.TestCheckResourceAttr(addr, "vcs_provider_id", vcs2TRN),
					testCaptureAttr(addr, "id", &id),
				),
			},
			{
				Config: link(quoted(ws2TRN), quoted(vcs2TRN)),
				Check: sdkresource.ComposeAggregateTestCheckFunc(
					testCheckAttrChanged(addr, "id", &id),
					sdkresource.TestCheckResourceAttr(addr, "workspace_id", ws2TRN),
				),
			},
		},
	})
}

// testCompleteFakeOAuth finishes a VCS provider's OAuth flow the way a browser would: it
// calls the Tharsis callback with the state from the provider's authorization URL. The
// fake GitLab accepts any code.
func testCompleteFakeOAuth(name string) sdkresource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("resource not found: %s", name)
		}
		authURL, err := url.Parse(rs.Primary.Attributes["oauth_authorization_url"])
		if err != nil {
			return fmt.Errorf("parsing the OAuth authorization URL: %w", err)
		}
		callback := os.Getenv("THARSIS_ENDPOINT") + "/v1/vcs/auth/callback?" + url.Values{
			"code":  {"fake-code"},
			"state": {authURL.Query().Get("state")},
		}.Encode()
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, callback, nil)
		if err != nil {
			return err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("calling the OAuth callback: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("OAuth callback for %s returned %s", name, resp.Status)
		}
		return nil
	}
}

// TestTRNReferenceLinkAfterImport imports a workspace VCS provider link, which stores both
// references as Global IDs, then switches the configuration to their TRNs: an in-place
// update with no replacement and no write call. Like TestTRNReferenceAfterImport, the link
// is created under one address and imported into a second one.
func TestTRNReferenceLinkAfterImport(t *testing.T) {
	fakeURL := os.Getenv(testFakeVCSURLEnvVar)
	if fakeURL == "" {
		t.Skipf("set %s to a fake GitLab reachable from the Tharsis API", testFakeVCSURLEnvVar)
	}
	base := trnRefBase("ws") + fmt.Sprintf(`
resource "tharsis_vcs_provider" "trn_vcs" {
	name                 = "trn_vcs"
	description          = "VCS provider for the TRN reference tests"
	url                  = "%s"
	group_path           = tharsis_group.root-group.full_path
	type                 = "gitlab"
	auto_create_webhooks = false
	oauth_client_id      = "trn-oauth-client-id"
	oauth_client_secret  = "trn-oauth-client-secret" # nosec G101 -- test value
}
`, fakeURL)
	block := func(name, wsRef, vcsRef string) string {
		return fmt.Sprintf(`
resource "tharsis_workspace_vcs_provider_link" "%s" {
	workspace_id          = %s
	vcs_provider_id       = %s
	repository_path       = "trn/reference"
	module_directory      = "."
	branch                = "main"
	tag_regex             = "^v.*"
	glob_patterns         = ["**/*.tf"]
	auto_speculative_plan = false
	webhook_disabled      = true
	depends_on            = [tharsis_workspace.trn_ws, tharsis_vcs_provider.trn_vcs]
}`, name, wsRef, vcsRef)
	}
	const wsGID, vcsGID = "tharsis_workspace.trn_ws.id", "tharsis_vcs_provider.trn_vcs.id"
	wsTRN := quoted("trn:workspace:" + trnRefGroupPath + "/trn_ws")
	vcsTRN := quoted("trn:vcs_provider:" + trnRefGroupPath + "/trn_vcs")
	own := block("trn_target", wsGID, vcsGID)
	const imported = "tharsis_workspace_vcs_provider_link.trn_import"
	var linkID, importedID string
	calls := newWriteCallLog(t)

	sdkresource.Test(t, sdkresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []sdkresource.TestStep{
			{Config: base, Check: testCompleteFakeOAuth("tharsis_vcs_provider.trn_vcs")},
			{
				Config: base + own,
				Check:  testCaptureAttr("tharsis_workspace_vcs_provider_link.trn_target", "id", &linkID),
			},
			{
				Config:             base + own + block("trn_import", wsGID, vcsGID),
				ResourceName:       imported,
				ImportState:        true,
				ImportStatePersist: true,
				ImportStateIdFunc:  func(*terraform.State) (string, error) { return linkID, nil },
			},
			{
				Config: base + own + block("trn_import", wsGID, vcsGID),
				Check: sdkresource.ComposeAggregateTestCheckFunc(
					sdkresource.TestCheckResourceAttrPair(imported, "workspace_id", "tharsis_workspace.trn_ws", "id"),
					sdkresource.TestCheckResourceAttrPair(imported, "vcs_provider_id", "tharsis_vcs_provider.trn_vcs", "id"),
					testCaptureAttr(imported, "id", &importedID),
					calls.mark(),
				),
			},
			{
				Config: base + own + block("trn_import", wsTRN, vcsTRN),
				Check: sdkresource.ComposeAggregateTestCheckFunc(
					testCheckAttrUnchanged(imported, "id", &importedID),
					sdkresource.TestCheckResourceAttr(imported, "workspace_id", "trn:workspace:"+trnRefGroupPath+"/trn_ws"),
					sdkresource.TestCheckResourceAttr(imported, "vcs_provider_id", "trn:vcs_provider:"+trnRefGroupPath+"/trn_vcs"),
					calls.checkNoneSinceMark(),
				),
			},
		},
	})
}
