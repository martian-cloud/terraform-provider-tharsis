package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

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

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// The providers exist; complete their OAuth flows before linking.
				Config: base,
				Check: resource.ComposeAggregateTestCheckFunc(
					testCompleteFakeOAuth("tharsis_vcs_provider.trn_vcs"),
					testCompleteFakeOAuth("tharsis_vcs_provider.trn_vcs2"),
				),
			},
			{
				Config: link(quoted(wsTRN), quoted(vcsTRN)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "workspace_id", wsTRN),
					resource.TestCheckResourceAttr(addr, "vcs_provider_id", vcsTRN),
					testCaptureAttr(addr, "id", &id),
					calls.mark(),
				),
			},
			{
				Config: link("tharsis_workspace.trn_ws.id", "tharsis_vcs_provider.trn_vcs.id"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckAttrUnchanged(addr, "id", &id),
					resource.TestCheckResourceAttrPair(addr, "workspace_id", "tharsis_workspace.trn_ws", "id"),
					resource.TestCheckResourceAttrPair(addr, "vcs_provider_id", "tharsis_vcs_provider.trn_vcs", "id"),
					calls.checkNoneSinceMark(),
				),
			},
			{
				Config: link(quoted(wsTRN), quoted(vcsTRN)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckAttrUnchanged(addr, "id", &id),
					resource.TestCheckResourceAttr(addr, "workspace_id", wsTRN),
					resource.TestCheckResourceAttr(addr, "vcs_provider_id", vcsTRN),
					calls.checkNoneSinceMark(),
				),
			},
			{
				Config: link(quoted(wsTRN), quoted(vcs2TRN)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckAttrChanged(addr, "id", &id),
					resource.TestCheckResourceAttr(addr, "vcs_provider_id", vcs2TRN),
					testCaptureAttr(addr, "id", &id),
				),
			},
			{
				Config: link(quoted(ws2TRN), quoted(vcs2TRN)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckAttrChanged(addr, "id", &id),
					resource.TestCheckResourceAttr(addr, "workspace_id", ws2TRN),
				),
			},
		},
	})
}

// testCompleteFakeOAuth finishes a VCS provider's OAuth flow the way a browser would: it
// calls the Tharsis callback with the state from the provider's authorization URL. The
// fake GitLab accepts any code.
func testCompleteFakeOAuth(name string) resource.TestCheckFunc {
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

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
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
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(imported, "workspace_id", "tharsis_workspace.trn_ws", "id"),
					resource.TestCheckResourceAttrPair(imported, "vcs_provider_id", "tharsis_vcs_provider.trn_vcs", "id"),
					testCaptureAttr(imported, "id", &importedID),
					calls.mark(),
				),
			},
			{
				Config: base + own + block("trn_import", wsTRN, vcsTRN),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckAttrUnchanged(imported, "id", &importedID),
					resource.TestCheckResourceAttr(imported, "workspace_id", "trn:workspace:"+trnRefGroupPath+"/trn_ws"),
					resource.TestCheckResourceAttr(imported, "vcs_provider_id", "trn:vcs_provider:"+trnRefGroupPath+"/trn_vcs"),
					calls.checkNoneSinceMark(),
				),
			},
		},
	})
}
