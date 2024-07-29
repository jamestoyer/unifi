package customplanmodifier

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// portOverridePortProfile is a plan modifier that sets a value to `null` if a port profile is set on a
// port override.
type portOverridePortProfile struct{}

// Description returns a human-readable description of the plan modifier.
func (m portOverridePortProfile) Description(ctx context.Context) string {
	return "The value will be set to null when the port_profile_id attribute is set."
}

// MarkdownDescription returns a markdown description of the plan modifier.
func (m portOverridePortProfile) MarkdownDescription(_ context.Context) string {
	return "The value will be set to null when the `port_profile_id` attribute is set."
}

// PlanModifyString implements the plan modification logic.
func (m portOverridePortProfile) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	expressions := req.PathExpression.MergeExpressions(path.MatchRelative().AtParent().AtName("port_profile_id"))

	for _, expression := range expressions {
		matchedPaths, diags := req.Config.PathMatches(ctx, expression)

		resp.Diagnostics.Append(diags...)

		// Collect all errors
		if diags.HasError() {
			continue
		}

		for _, mp := range matchedPaths {
			var portProfileID types.String
			diags := req.Config.GetAttribute(ctx, mp, &portProfileID)
			resp.Diagnostics.Append(diags...)

			// Collect all errors
			if diags.HasError() {
				continue
			}

			if portProfileID.ValueString() != "" {
				resp.PlanValue = types.StringPointerValue(nil)
				return
			}
		}
	}
}

// PortOverridePortProfileIDString returns a string plan modifier that sets the value to `null` if a port profile is set
// on the override.
func PortOverridePortProfileIDString() planmodifier.String {
	return portOverridePortProfile{}
}
