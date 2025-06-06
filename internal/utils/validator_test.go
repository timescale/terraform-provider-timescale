package multiplyvalidator

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestEqualToMultipleOfValidator(t *testing.T) {
	t.Parallel()

	type testCase struct {
		val                            types.Int64
		multiple                       int64
		attributesToSumPathExpressions path.Expressions
		requestConfigRaw               map[string]tftypes.Value
		expectError                    bool
	}
	tests := map[string]testCase{
		"unknown Int64": {
			val: types.Int64Unknown(),
		},
		"null Int64": {
			val: types.Int64Null(),
		},
		"valid integer as Int64 more than sum of attributes": {
			val:      types.Int64Value(11),
			multiple: 2,
			attributesToSumPathExpressions: path.Expressions{
				path.MatchRoot("one"),
			},
			requestConfigRaw: map[string]tftypes.Value{
				"one": tftypes.NewValue(tftypes.Number, 5),
			},
			expectError: true,
		},
		"valid integer as Int64 less than sum of attributes": {
			val:      types.Int64Value(9),
			multiple: 2,
			attributesToSumPathExpressions: path.Expressions{
				path.MatchRoot("one"),
			},
			requestConfigRaw: map[string]tftypes.Value{
				"one": tftypes.NewValue(tftypes.Number, 5),
			},
			expectError: true,
		},
		"valid integer as Int64 equal to sum of attributes": {
			val:      types.Int64Value(10),
			multiple: 2,
			attributesToSumPathExpressions: path.Expressions{
				path.MatchRoot("one"),
			},
			requestConfigRaw: map[string]tftypes.Value{
				"one": tftypes.NewValue(tftypes.Number, 5),
			},
		},
		"valid integer as Int64 equal to sum of 2 attributes": {
			val:      types.Int64Value(30),
			multiple: 2,
			attributesToSumPathExpressions: path.Expressions{
				path.MatchRoot("one"),
				path.MatchRoot("two"),
			},
			requestConfigRaw: map[string]tftypes.Value{
				"one": tftypes.NewValue(tftypes.Number, 5),
				"two": tftypes.NewValue(tftypes.Number, 3),
			},
		},
		// "valid integer as Int64 equal to sum of attributes, when one summed attribute is null": {
		// 	val: types.Int64Value(8),
		// 	attributesToSumPathExpressions: path.Expressions{
		// 		path.MatchRoot("one"),
		// 		path.MatchRoot("two"),
		// 	},
		// 	requestConfigRaw: map[string]tftypes.Value{
		// 		"one": tftypes.NewValue(tftypes.Number, nil),
		// 		"two": tftypes.NewValue(tftypes.Number, 8),
		// 	},
		// },
		// "valid integer as Int64 does not return error when all attributes are null": {
		// 	val: types.Int64Null(),
		// 	attributesToSumPathExpressions: path.Expressions{
		// 		path.MatchRoot("one"),
		// 		path.MatchRoot("two"),
		// 	},
		// 	requestConfigRaw: map[string]tftypes.Value{
		// 		"one": tftypes.NewValue(tftypes.Number, nil),
		// 		"two": tftypes.NewValue(tftypes.Number, nil),
		// 	},
		// },
		// "valid integer as Int64 returns error when all attributes to sum are null": {
		// 	val: types.Int64Value(1),
		// 	attributesToSumPathExpressions: path.Expressions{
		// 		path.MatchRoot("one"),
		// 		path.MatchRoot("two"),
		// 	},
		// 	requestConfigRaw: map[string]tftypes.Value{
		// 		"one": tftypes.NewValue(tftypes.Number, nil),
		// 		"two": tftypes.NewValue(tftypes.Number, nil),
		// 	},
		// 	expectError: true,
		// },
		// "valid integer as Int64 equal to sum of attributes, when one summed attribute is unknown": {
		// 	val: types.Int64Value(8),
		// 	attributesToSumPathExpressions: path.Expressions{
		// 		path.MatchRoot("one"),
		// 		path.MatchRoot("two"),
		// 	},
		// 	requestConfigRaw: map[string]tftypes.Value{
		// 		"one": tftypes.NewValue(tftypes.Number, tftypes.UnknownValue),
		// 		"two": tftypes.NewValue(tftypes.Number, 8),
		// 	},
		// },
		// "valid integer as Int64 does not return error when all attributes are unknown": {
		// 	val: types.Int64Unknown(),
		// 	attributesToSumPathExpressions: path.Expressions{
		// 		path.MatchRoot("one"),
		// 		path.MatchRoot("two"),
		// 	},
		// 	requestConfigRaw: map[string]tftypes.Value{
		// 		"one": tftypes.NewValue(tftypes.Number, tftypes.UnknownValue),
		// 		"two": tftypes.NewValue(tftypes.Number, tftypes.UnknownValue),
		// 	},
		// },
		// "valid integer as Int64 does not return error when all attributes to sum are unknown": {
		// 	val: types.Int64Value(1),
		// 	attributesToSumPathExpressions: path.Expressions{
		// 		path.MatchRoot("one"),
		// 		path.MatchRoot("two"),
		// 	},
		// 	requestConfigRaw: map[string]tftypes.Value{
		// 		"one": tftypes.NewValue(tftypes.Number, tftypes.UnknownValue),
		// 		"two": tftypes.NewValue(tftypes.Number, tftypes.UnknownValue),
		// 	},
		// },
		// "error when attribute to sum is not Number": {
		// 	val: types.Int64Value(9),
		// 	attributesToSumPathExpressions: path.Expressions{
		// 		path.MatchRoot("one"),
		// 		path.MatchRoot("two"),
		// 	},
		// 	requestConfigRaw: map[string]tftypes.Value{
		// 		"one": tftypes.NewValue(tftypes.Bool, true),
		// 		"two": tftypes.NewValue(tftypes.Number, 9),
		// 	},
		// 	expectError: true,
		// },
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			request := validator.Int64Request{
				Path:           path.Root("test"),
				PathExpression: path.MatchRoot("test"),
				ConfigValue:    test.val,
				Config: tfsdk.Config{
					Raw: tftypes.NewValue(tftypes.Object{}, test.requestConfigRaw),
					Schema: schema.Schema{
						Attributes: map[string]schema.Attribute{
							"test": schema.Int64Attribute{},
							"one":  schema.Int64Attribute{},
							"two":  schema.Int64Attribute{},
						},
					},
				},
			}

			response := validator.Int64Response{}

			EqualToMultipleOf(test.multiple, test.attributesToSumPathExpressions...).ValidateInt64(t.Context(), request, &response)

			if !response.Diagnostics.HasError() && test.expectError {
				t.Fatal("expected error, got no error")
			}

			if response.Diagnostics.HasError() && !test.expectError {
				t.Fatalf("got unexpected error: %s", response.Diagnostics)
			}
		})
	}
}
