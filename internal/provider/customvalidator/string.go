package customvalidator

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-validators/helpers/validatordiag"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// stringValueWithOtherPathsValidator validates that when the given String value matches that the given paths are set.
type stringValueWithOtherPathsValidator struct {
	paths path.Expressions
	value types.String
}

func (v stringValueWithOtherPathsValidator) Description(ctx context.Context) string {
	return v.MarkdownDescription(ctx)
}

func (v stringValueWithOtherPathsValidator) MarkdownDescription(_ context.Context) string {
	return fmt.Sprintf("when value %q is set these paths must also be set: %s", v.value, v.paths.String())
}

func (v stringValueWithOtherPathsValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue
	if !value.Equal(v.value) {
		return
	}

	expressions := req.PathExpression.MergeExpressions(v.paths...)

	for _, expression := range expressions {
		matchedPaths, diags := req.Config.PathMatches(ctx, expression)

		resp.Diagnostics.Append(diags...)

		// Collect all errors
		if diags.HasError() {
			continue
		}

		for _, mp := range matchedPaths {
			// If the user specifies the same attribute this validator is applied to,
			// also as part of the input, skip it
			if mp.Equal(req.Path) {
				continue
			}

			var mpVal attr.Value
			diags := req.Config.GetAttribute(ctx, mp, &mpVal)
			resp.Diagnostics.Append(diags...)

			// Collect all errors
			if diags.HasError() {
				continue
			}

			// Delay validation until all involved attribute have a known value
			if mpVal.IsUnknown() {
				return
			}

			if mpVal.IsNull() {
				resp.Diagnostics.Append(validatordiag.InvalidAttributeCombinationDiagnostic(
					req.Path,
					fmt.Sprintf("Attribute %q must be specified when %q is %s", mp, req.Path, v.value),
				))
			}
		}
	}

}

// StringValueWithPaths checks that the String held in the attribute is the given `value` and all attributes set in
// `paths` set.
func StringValueWithPaths(value string, paths ...path.Expression) validator.String {
	return stringValueWithOtherPathsValidator{
		paths: paths,
		value: types.StringValue(value),
	}
}

// stringValueConflictsWithOtherPathsValidator validates that when the given String value matches that the given paths
// are not set.
type stringValueConflictsWithOtherPathsValidator struct {
	paths path.Expressions
	value types.String
}

func (v stringValueConflictsWithOtherPathsValidator) Description(ctx context.Context) string {
	return v.MarkdownDescription(ctx)
}

func (v stringValueConflictsWithOtherPathsValidator) MarkdownDescription(_ context.Context) string {
	return fmt.Sprintf("when value %q is set these paths must not be set: %s", v.value, v.paths.String())
}

func (v stringValueConflictsWithOtherPathsValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue
	if !value.Equal(v.value) {
		return
	}

	expressions := req.PathExpression.MergeExpressions(v.paths...)

	for _, expression := range expressions {
		matchedPaths, diags := req.Config.PathMatches(ctx, expression)

		resp.Diagnostics.Append(diags...)

		// Collect all errors
		if diags.HasError() {
			continue
		}

		for _, mp := range matchedPaths {
			// If the user specifies the same attribute this validator is applied to,
			// also as part of the input, skip it
			if mp.Equal(req.Path) {
				continue
			}

			var mpVal attr.Value
			diags := req.Config.GetAttribute(ctx, mp, &mpVal)
			resp.Diagnostics.Append(diags...)

			// Collect all errors
			if diags.HasError() {
				continue
			}

			// Delay validation until all involved attribute have a known value
			if mpVal.IsUnknown() {
				return
			}

			if !mpVal.IsNull() {
				resp.Diagnostics.Append(validatordiag.InvalidAttributeCombinationDiagnostic(
					req.Path,
					fmt.Sprintf("Attribute %q must not be specified when %q is %s", mp, req.Path, v.value),
				))
			}
		}
	}

}

// StringValueConflictsWithPaths checks that the String held in the attribute is the given `value` and all attributes set
// in `paths` set.
func StringValueConflictsWithPaths(value string, paths ...path.Expression) validator.String {
	return stringValueConflictsWithOtherPathsValidator{
		paths: paths,
		value: types.StringValue(value),
	}
}
