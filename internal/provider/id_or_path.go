package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/client"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
)

// idOrPath is a target a resource names with an ID attribute (a Global ID or a TRN) or, for
// older configurations, its deprecated path counterpart, such as workspace_id and
// workspace_path.
type idOrPath struct {
	ID   types.String
	Path types.String
}

// reference returns the target as the resource's Create picks it: ID when set, else Path as a
// TRN of type t. ok is false when the target is not known yet (ID unknown until apply) or
// neither attribute is set, so there is nothing to compare.
func (r idOrPath) reference(t trn.Type) (ref string, ok bool) {
	switch {
	case isKnownValue(r.ID) && r.ID.ValueString() != "":
		return r.ID.ValueString(), true
	case r.ID.IsUnknown():
		return "", false
	case isKnownValue(r.Path) && r.Path.ValueString() != "":
		return t.Build(r.Path.ValueString()), true
	default:
		return "", false
	}
}

// requireReplaceIfDifferentIDOrPathObject is called from ModifyPlan when an ID attribute or
// its path counterpart changed (the attributes in changed). Switching between the two, or
// between a Global ID and a TRN, of the same object is only a change of form, so it plans no
// replacement. Otherwise, including when either target is not known or cannot be resolved, it
// replaces every attribute in changed, as RequiresReplace would. It returns whether it
// planned the replacement.
func requireReplaceIfDifferentIDOrPathObject(ctx context.Context, c *client.GRPCClient,
	resp *resource.ModifyPlanResponse, prior, planned idOrPath, t trn.Type, changed ...path.Path,
) (replaced bool) {
	priorRef, priorOK := prior.reference(t)
	plannedRef, plannedOK := planned.reference(t)
	if priorOK && plannedOK {
		same, err := sameObject(ctx, c, priorRef, plannedRef, t)
		if same {
			return false
		}
		if err != nil {
			resp.Diagnostics.AddWarning("Could not compare reference values",
				fmt.Sprintf("Planning a replacement because %q and %q could not both be resolved, so it "+
					"is not known whether they name the same object: %s", priorRef, plannedRef, err))
		}
	}
	resp.RequiresReplace = append(resp.RequiresReplace, changed...)
	return true
}
