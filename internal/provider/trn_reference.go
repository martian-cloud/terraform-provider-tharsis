package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/client"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
)

// sameReference reports whether a stored reference value (a TRN or a Global ID)
// still points to the same object as the authoritative value returned by the API.
//
//   - stored: the value currently in state (what the caller wrote: a TRN or a GID).
//   - apiGID: the object's Global ID from the API response.
//   - apiTRN: the object's TRN from the API response (its own Metadata.Trn, or the
//     parent path built into a group TRN). May be empty if not available.
//
// A stored Global ID matches when it equals apiGID. A stored TRN matches when it
// equals apiTRN. Anything else means the reference now points elsewhere (drift).
func sameReference(stored types.String, apiGID, apiTRN string) bool {
	if stored.IsNull() || stored.IsUnknown() {
		return false
	}
	v := stored.ValueString()
	if trn.IsTRN(v) {
		return apiTRN != "" && v == apiTRN
	}
	return v == apiGID
}

// keepOrRefresh returns the value to store for a reference attribute on read.
// If the stored value still names the same object, it is preserved (so a TRN the
// caller wrote is not rewritten to a Global ID). Otherwise the API's Global ID is
// taken, so a change made outside Terraform shows up as drift.
func keepOrRefresh(stored types.String, apiGID, apiTRN string) types.String {
	if sameReference(stored, apiGID, apiTRN) {
		return stored
	}
	return types.StringValue(apiGID)
}

// resolveObjectGID resolves a reference value (a TRN or a Global ID) to the
// referenced object's Global ID, using the matching Get*ByID call for its type.
// All of these accept either form. An empty string and errors yield "".
func resolveObjectGID(ctx context.Context, c *client.GRPCClient, ref string) string {
	if ref == "" {
		return ""
	}
	t := trn.TypeGroup // default; overridden below by the parsed type when ref is a TRN
	if trn.IsTRN(ref) {
		parsed, err := trn.ParseAny(ref)
		if err != nil {
			return ""
		}
		t = parsed.Type()
	}
	switch t {
	case trn.TypeGroup:
		if g, err := c.GroupsClient.GetGroupByID(ctx, &pb.GetGroupByIDRequest{Id: ref}); err == nil {
			return g.Metadata.Id
		}
	case trn.TypeWorkspace:
		if w, err := c.WorkspacesClient.GetWorkspaceByID(ctx, &pb.GetWorkspaceByIDRequest{Id: ref}); err == nil {
			return w.Metadata.Id
		}
	case trn.TypeManagedIdentity:
		if m, err := c.ManagedIdentitiesClient.GetManagedIdentityByID(ctx, &pb.GetManagedIdentityByIDRequest{Id: ref}); err == nil {
			return m.Metadata.Id
		}
	case trn.TypeVCSProvider:
		if v, err := c.VCSProvidersClient.GetVCSProviderByID(ctx, &pb.GetVCSProviderByIDRequest{Id: ref}); err == nil {
			return v.Metadata.Id
		}
	}
	return ""
}

// planKeepPriorIfSameObject returns the value a RequiresReplace reference attribute
// should plan to. When the prior (state) value and the planned (config) value name the
// same object in different forms (e.g. one a TRN, one a Global ID), it returns prior, so
// no replacement is planned. Otherwise it returns planned unchanged. A nil client (the
// provider is not configured during some plan calls) returns planned unchanged.
func planKeepPriorIfSameObject(ctx context.Context, c *client.GRPCClient, prior, planned types.String) types.String {
	if c == nil || prior.IsNull() || prior.IsUnknown() || planned.IsNull() || planned.IsUnknown() {
		return planned
	}
	if prior.ValueString() == planned.ValueString() {
		return planned
	}
	pr := resolveObjectGID(ctx, c, prior.ValueString())
	pl := resolveObjectGID(ctx, c, planned.ValueString())
	if pr != "" && pr == pl {
		return prior
	}
	return planned
}
