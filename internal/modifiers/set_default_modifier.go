// Package modifiers contains modifiers necessary for asserting
// default values when none are supplied to the provider.
package modifiers

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ planmodifier.Set = setDefaultModifier{}

// setDefaultModifier is a plan modifier that sets a default value for a
// types.SetType attribute when it is not configured. The attribute must be
// marked as Optional and Computed. When setting the state during the resource
// Create, Read, or Update methods, this default value must also be included or
// the Terraform CLI will generate an error.
type setDefaultModifier struct {
	Elements []attr.Value
}

func SetDefault(elements []attr.Value) planmodifier.Set {
	return setDefaultModifier{
		Elements: elements,
	}
}

// Description returns a plain text description of the validator's behavior, suitable for a practitioner to understand its impact.
func (m setDefaultModifier) Description(_ context.Context) string {
	return fmt.Sprintf("If value is not configured, defaults to %s", m.Elements)
}

// MarkdownDescription returns a markdown formatted description of the validator's behavior, suitable for a practitioner to understand its impact.
func (m setDefaultModifier) MarkdownDescription(_ context.Context) string {
	return fmt.Sprintf("If value is not configured, defaults to `%s`", m.Elements)
}

// PlanModifySet runs the logic of the plan modifier.
// Access to the configuration, plan, and state is available in `req`, while
// `resp` contains fields for updating the planned value, triggering resource
// replacement, and returning diagnostics.
func (m setDefaultModifier) PlanModifySet(ctx context.Context, req planmodifier.SetRequest, resp *planmodifier.SetResponse) {
	// If the value is unknown or known, do not set default value.
	if !req.PlanValue.IsNull() {
		return
	}

	resp.PlanValue, resp.Diagnostics = types.SetValue(req.PlanValue.ElementType(ctx), m.Elements)
}
