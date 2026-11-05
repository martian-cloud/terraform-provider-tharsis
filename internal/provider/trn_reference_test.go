package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/client"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
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
