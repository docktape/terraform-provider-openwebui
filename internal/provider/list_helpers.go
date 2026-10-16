package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// expandStringList converts a Terraform list attribute into a Go slice of strings.
func expandStringList(ctx context.Context, value types.List, attribute path.Path, diags *diag.Diagnostics) []string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}

	var result []string
	if err := value.ElementsAs(ctx, &result, false); err != nil {
		diags.AddAttributeError(
			attribute,
			"Invalid string list value",
			fmt.Sprintf("Unable to decode attribute %s into a list of strings: %v", attribute.String(), err),
		)
		return nil
	}

	return result
}

// flattenStringSlice converts a slice of strings into a Terraform List value.
// An empty or nil slice produces an empty list (not null) so that round-trips
// through the API remain consistent when a config explicitly sets list = [].
func flattenStringSlice(ctx context.Context, values []string) (types.List, diag.Diagnostics) {
	if len(values) == 0 {
		return types.ListValueMust(types.StringType, []attr.Value{}), nil
	}

	list, diags := types.ListValueFrom(ctx, types.StringType, values)
	return list, diags
}

// flattenNullableStringSlice is flattenStringSlice for server fields where
// null and [] mean different things: a nil slice (JSON null or an absent key)
// produces a null list, an empty non-nil slice an empty list.
func flattenNullableStringSlice(ctx context.Context, values []string) (types.List, diag.Diagnostics) {
	if values == nil {
		return types.ListNull(types.StringType), nil
	}
	return flattenStringSlice(ctx, values)
}

// expandStringListPtr converts a known list attribute into a pointer to a
// non-nil slice (so an empty list is sent as []), and a null or unknown one
// into nil (so the key is omitted).
func expandStringListPtr(ctx context.Context, value types.List, attribute path.Path, diags *diag.Diagnostics) *[]string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	result := expandStringList(ctx, value, attribute, diags)
	if result == nil {
		result = []string{}
	}
	return &result
}
