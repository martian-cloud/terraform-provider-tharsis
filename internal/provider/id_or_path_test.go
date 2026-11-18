package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
)

// TestIDOrPathReference checks which value names the target: ID wins, as in Create, and an
// unknown ID is not comparable even if a path is set.
func TestIDOrPathReference(t *testing.T) {
	const gid, p = "V18wMDAwMDAwMC0wMDAwLTAwMDAtMDAwMC0wMDAwMDAwMDAwMDA", "g/ws"
	for _, tc := range []struct {
		name    string
		r       idOrPath
		wantRef string
		wantOK  bool
	}{
		{"id", idOrPath{types.StringValue(gid), types.StringNull()}, gid, true},
		{"id wins over path", idOrPath{types.StringValue(gid), types.StringValue(p)}, gid, true},
		{"path only", idOrPath{types.StringNull(), types.StringValue(p)}, "trn:workspace:" + p, true},
		{"unknown id", idOrPath{types.StringUnknown(), types.StringValue(p)}, "", false},
		{"empty id falls back to path", idOrPath{types.StringValue(""), types.StringValue(p)}, "trn:workspace:" + p, true},
		{"unknown path", idOrPath{types.StringNull(), types.StringUnknown()}, "", false},
		{"neither", idOrPath{types.StringNull(), types.StringNull()}, "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ref, ok := tc.r.reference(trn.TypeWorkspace)
			if ref != tc.wantRef || ok != tc.wantOK {
				t.Fatalf("reference = (%q, %v), want (%q, %v)", ref, ok, tc.wantRef, tc.wantOK)
			}
		})
	}
}

// With no client, requireReplaceIfDifferentIDOrPathObject cannot prove that two different
// values name the same object, so it replaces the changed attributes, as RequiresReplace would.
func TestRequireReplaceIfDifferentIDOrPathObjectWithoutAPI(t *testing.T) {
	ctx := context.Background()
	id := func(v string) idOrPath { return idOrPath{ID: types.StringValue(v), Path: types.StringNull()} }
	byPath := func(v string) idOrPath { return idOrPath{ID: types.StringNull(), Path: types.StringValue(v)} }
	for _, tc := range []struct {
		name           string
		prior, planned idOrPath
		wantReplace    bool
		wantWarning    bool
	}{
		{"same value", id(unitGroupGID), id(unitGroupGID), false, false},
		{"path and the same TRN", byPath("some/group"), id(unitGroupTRN), false, false},
		{"two different Global IDs", id(unitGroupGID), id(unitGroupGID + "x"), true, false},
		{"Global ID and TRN, cannot resolve", id(unitGroupGID), id(unitGroupTRN), true, true},
		{"planned unknown", id(unitGroupGID), idOrPath{ID: types.StringUnknown(), Path: types.StringNull()}, true, false},
		{"planned neither", id(unitGroupGID), idOrPath{ID: types.StringNull(), Path: types.StringNull()}, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp := &resource.ModifyPlanResponse{}
			replaced := requireReplaceIfDifferentIDOrPathObject(ctx, nil, resp, tc.prior, tc.planned, trn.TypeGroup,
				path.Root("group_id"), path.Root("group_path"))
			if replaced != tc.wantReplace {
				t.Fatalf("replaced = %v, want %v", replaced, tc.wantReplace)
			}
			if (len(resp.RequiresReplace) == 2) != tc.wantReplace {
				t.Fatalf("RequiresReplace = %v, want replace %v", resp.RequiresReplace, tc.wantReplace)
			}
			if got := resp.Diagnostics.WarningsCount() > 0; got != tc.wantWarning {
				t.Fatalf("warning = %v, want %v: %v", got, tc.wantWarning, resp.Diagnostics)
			}
		})
	}
}
