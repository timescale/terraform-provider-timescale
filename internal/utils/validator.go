package multiplyvalidator

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-plugin-framework-validators/helpers/validatordiag"
)

var _ validator.Int64 = multipleValidator{}

// multipleValidator validates that an integer Attribute's value equals the multiplication of
// provided integer and Attributes retrieved via the given path expressions.
type multipleValidator struct {
	attributeToMultiplyExpressions path.Expressions
	multiple                       int64
}

// Description describes the validation in plain text formatting.
func (mv multipleValidator) Description(_ context.Context) string {
	attributePaths := make([]string, 0)
	for _, p := range mv.attributeToMultiplyExpressions {
		attributePaths = append(attributePaths, p.String())
	}

	return fmt.Sprintf("value must be equal to the multiplication of %s * %d",
		strings.Join(attributePaths, "* "), mv.multiple)
}

// MarkdownDescription describes the validation in Markdown formatting.
func (mv multipleValidator) MarkdownDescription(ctx context.Context) string {
	return mv.Description(ctx)
}

// ValidateInt64 performs the validation.
func (mv multipleValidator) ValidateInt64(ctx context.Context, request validator.Int64Request, response *validator.Int64Response) {
	if request.ConfigValue.IsNull() || request.ConfigValue.IsUnknown() {
		return
	}

	// Ensure input path expressions resolution against the current attribute
	expressions := request.PathExpression.MergeExpressions(mv.attributeToMultiplyExpressions...)

	var multiplication = mv.multiple
	if multiplication == 0 {
		multiplication = 1
	}
	// Multiply the value of all the attributes involved, but only if they are all known.
	for _, expression := range expressions {
		matchedPaths, diags := request.Config.PathMatches(ctx, expression)
		response.Diagnostics.Append(diags...)

		// Collect all errors
		if diags.HasError() {
			continue
		}

		for _, mp := range matchedPaths {
			// If the user specifies the same attribute this validator is applied to,
			// also as part of the input, skip it
			if mp.Equal(request.Path) {
				continue
			}

			// Get the value
			var matchedValue attr.Value
			diags := request.Config.GetAttribute(ctx, mp, &matchedValue)
			response.Diagnostics.Append(diags...)
			if diags.HasError() {
				continue
			}

			if matchedValue.IsUnknown() {
				return
			}

			if matchedValue.IsNull() {
				continue
			}

			// We know there is a value, convert it to the expected type
			var attribToSum types.Int64
			diags = tfsdk.ValueAs(ctx, matchedValue, &attribToSum)
			response.Diagnostics.Append(diags...)
			if diags.HasError() {
				continue
			}

			multiplication *= attribToSum.ValueInt64()
		}
	}

	if request.ConfigValue.ValueInt64() != multiplication {
		response.Diagnostics.Append(validatordiag.InvalidAttributeValueDiagnostic(
			request.Path,
			mv.Description(ctx),
			fmt.Sprintf("%d", request.ConfigValue.ValueInt64()),
		))
	}
}

// EqualToMultipleOf returns an AttributeValidator which ensures that any configured
// attribute value:
//
//   - Is a number, which can be represented by a 64-bit integer.
//   - Is equal to the multiplication of the given attributes retrieved via the given path expression(s).
//
// Null (unconfigured) and unknown (known after apply) values are skipped.
func EqualToMultipleOf(multiple int64, attributeToMultiplyExpressions ...path.Expression) validator.Int64 {
	return multipleValidator{attributeToMultiplyExpressions, multiple}
}
