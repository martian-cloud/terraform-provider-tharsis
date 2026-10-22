package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
)

// trnPrefix is the leading marker of a Tharsis Resource Name. Resources that
// parse composite import identifiers use it to detect a TRN where an ID is
// required.
const trnPrefix = "trn:"

// importByNormalizedID seeds the id attribute from an import request, accepting a
// resource path, a GID, or a TRN and normalizing it to a canonical identifier.
//
// The Tharsis API resolves identifiers through its service catalog, which accepts
// only TRNs and GIDs. A bare resource path such as "my-group/my-identity" is
// rejected, so the path must be converted to a TRN before it reaches the API.
// trn.Type.Normalize performs that conversion and passes TRNs and GIDs through
// unchanged, which lets users import by whichever form they have on hand.
//
// Only the id attribute is written here. Terraform always calls Read immediately
// after ImportState, and Read replaces id with the authoritative GID returned by
// the API while populating the remaining attributes, so storing a TRN at this
// point is transient and the final state converges on the GID.
//
// Resources whose identity is not expressible as a single path, such as
// tharsis_assigned_managed_identity, parse their own composite identifiers
// instead of using this helper.
func importByNormalizedID(ctx context.Context, trnType trn.Type,
	req resource.ImportStateRequest, resp *resource.ImportStateResponse,
) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"),
		trnType.Normalize(req.ID))...)
}
