package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/client"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Reference attributes (group_id, workspace_id, ...) accept either a Global ID or a TRN.
// State keeps whichever form the configuration uses: the helpers below only replace it
// with the API's Global ID when it names a different object (drift), and only force a
// replacement when the old and new values name different objects.

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

// keepOrRefresh returns the value to store for a reference attribute after reading the
// object back, when the API response lets the referenced object's TRN be built. If the
// stored value still names the same object it is preserved, so a TRN the caller wrote is
// not rewritten to a Global ID. Otherwise the API's Global ID is taken, so a change made
// outside Terraform shows up as drift. Use keepOrRefreshResolve when no TRN is available.
func keepOrRefresh(stored types.String, apiGID, apiTRN string) types.String {
	if sameReference(stored, apiGID, apiTRN) {
		return stored
	}
	return types.StringValue(apiGID)
}

// keepOrRefreshResolve is keepOrRefresh for a response that carries only the referenced
// object's Global ID. A stored TRN is resolved to its Global ID to compare. If the TRN no
// longer names any object, the API's Global ID is taken (drift); on any other lookup
// error the stored value is kept rather than reporting drift that may not exist.
func keepOrRefreshResolve(ctx context.Context, c *client.GRPCClient, stored types.String,
	apiGID string, t trn.Type,
) types.String {
	if stored.IsNull() || stored.IsUnknown() {
		return types.StringValue(apiGID)
	}
	v := stored.ValueString()
	if !trn.IsTRN(v) {
		return keepOrRefresh(stored, apiGID, "")
	}
	gid, err := resolveObjectGID(ctx, c, v, t)
	switch {
	case err == nil && gid == apiGID:
		return stored
	case err == nil, status.Code(err) == codes.NotFound:
		return types.StringValue(apiGID)
	default:
		return stored
	}
}

// resolveObjectGID resolves a reference value (a TRN or a Global ID) to the Global ID of
// the object it names, which must be of type t. The Get*ByID call for t is used; all of
// them accept either form. A TRN of a different type is an error.
func resolveObjectGID(ctx context.Context, c *client.GRPCClient, ref string, t trn.Type) (string, error) {
	if c == nil {
		return "", fmt.Errorf("provider is not configured")
	}
	if trn.IsTRN(ref) {
		parsed, err := trn.ParseAny(ref)
		if err != nil {
			return "", err
		}
		if parsed.Type() != t {
			return "", fmt.Errorf("%q is a %s TRN, expected a %s", ref, parsed.Type(), t)
		}
	}
	switch t {
	case trn.TypeGroup:
		g, err := c.GroupsClient.GetGroupByID(ctx, &pb.GetGroupByIDRequest{Id: ref})
		if err != nil {
			return "", err
		}
		return g.Metadata.Id, nil
	case trn.TypeWorkspace:
		w, err := c.WorkspacesClient.GetWorkspaceByID(ctx, &pb.GetWorkspaceByIDRequest{Id: ref})
		if err != nil {
			return "", err
		}
		return w.Metadata.Id, nil
	case trn.TypeManagedIdentity:
		m, err := c.ManagedIdentitiesClient.GetManagedIdentityByID(ctx, &pb.GetManagedIdentityByIDRequest{Id: ref})
		if err != nil {
			return "", err
		}
		return m.Metadata.Id, nil
	case trn.TypeVCSProvider:
		v, err := c.VCSProvidersClient.GetVCSProviderByID(ctx, &pb.GetVCSProviderByIDRequest{Id: ref})
		if err != nil {
			return "", err
		}
		return v.Metadata.Id, nil
	case trn.TypeRunner:
		r, err := c.RunnersClient.GetRunnerByID(ctx, &pb.GetRunnerByIDRequest{Id: ref})
		if err != nil {
			return "", err
		}
		return r.Metadata.Id, nil
	case trn.TypeServiceAccount:
		sa, err := c.ServiceAccountsClient.GetServiceAccountByID(ctx, &pb.GetServiceAccountByIDRequest{Id: ref})
		if err != nil {
			return "", err
		}
		return sa.Metadata.Id, nil
	default:
		return "", fmt.Errorf("resolving a %s reference is not supported", t)
	}
}

// requireReplaceIfDifferentObject is called from ModifyPlan for a reference attribute the
// API cannot update in place. It forces a replacement only when the prior and planned
// values name different objects of type t. Two forms of the same object (a Global ID and
// its TRN) plan an in-place update that only changes the stored form. The planned value
// itself is never modified, so the plan always matches the configuration.
//
// The check cannot be a RequiresReplaceIf plan modifier: the framework builds schemas
// from an unconfigured resource, so a schema-level closure would have no API client.
// It mirrors stringplanmodifier.RequiresReplace exactly — no replacement on create or
// destroy or for equal values, a replacement for any other difference, including to or
// from null or unknown — with one exemption: two known values that resolve to the same
// object. Whenever sameness cannot be established (no client, a failed lookup) it
// replaces, so it never replaces less than a plain RequiresReplace would.
func requireReplaceIfDifferentObject(ctx context.Context, c *client.GRPCClient,
	req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse, p path.Path, t trn.Type,
) {
	// Nothing to compare on create, nothing to plan on destroy.
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}
	var prior, planned types.String
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, p, &prior)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, p, &planned)...)
	if resp.Diagnostics.HasError() || planned.Equal(prior) {
		return
	}
	if isKnownValue(prior) && isKnownValue(planned) {
		priorGID, priorErr := resolveObjectGID(ctx, c, prior.ValueString(), t)
		plannedGID, plannedErr := resolveObjectGID(ctx, c, planned.ValueString(), t)
		if priorErr == nil && plannedErr == nil && priorGID == plannedGID {
			return
		}
		// NotFound means the two cannot be the same object, so only other errors leave the
		// answer unknown and are worth a warning.
		var lookupErrs []error
		for _, e := range []error{priorErr, plannedErr} {
			if e != nil && status.Code(e) != codes.NotFound {
				lookupErrs = append(lookupErrs, e)
			}
		}
		if err := errors.Join(lookupErrs...); err != nil {
			resp.Diagnostics.AddAttributeWarning(p, "Could not compare reference values",
				fmt.Sprintf("Planning a replacement because %q and %q could not both be resolved, so "+
					"it is not known whether they name the same object: %s", prior.ValueString(),
					planned.ValueString(), err))
		}
	}
	resp.RequiresReplace = append(resp.RequiresReplace, p)
}

// isKnownValue reports whether v is neither null nor unknown.
func isKnownValue(v types.String) bool {
	return !v.IsNull() && !v.IsUnknown()
}
