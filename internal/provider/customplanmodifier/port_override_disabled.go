package customplanmodifier

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// portOverrideDisabled is a plan modifier that sets a value to `null` if a port profile is set on a
// port override.
type portOverrideDisabled struct {
	stringValue string
}

// Description returns a human-readable description of the plan modifier.
func (m portOverrideDisabled) Description(ctx context.Context) string {
	return "The value will be set to null when the port_profile_id attribute is set."
}

// MarkdownDescription returns a markdown description of the plan modifier.
func (m portOverrideDisabled) MarkdownDescription(_ context.Context) string {
	return "The value will be set to null when the `port_profile_id` attribute is set."
}

// PlanModifyString implements the plan modification logic.
func (m portOverrideDisabled) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	expressions := req.PathExpression.MergeExpressions(path.MatchRelative().AtParent().AtName("disabled"))

	for _, expression := range expressions {
		matchedPaths, diags := req.Config.PathMatches(ctx, expression)

		resp.Diagnostics.Append(diags...)

		// Collect all errors
		if diags.HasError() {
			continue
		}

		for _, mp := range matchedPaths {
			var disabled types.Bool
			diags := req.Config.GetAttribute(ctx, mp, &disabled)
			resp.Diagnostics.Append(diags...)

			// Collect all errors
			if diags.HasError() {
				continue
			}

			if disabled.ValueBool() {
				resp.PlanValue = types.StringValue(m.stringValue)
				return
			}
		}
	}
}

// PortOverrideDisabledString returns a string plan modifier that sets the value to given value if `disabled` is `true.
func PortOverrideDisabledString(value string) planmodifier.String {
	return portOverrideDisabled{stringValue: value}
}
