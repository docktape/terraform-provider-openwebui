package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// boolPtr and boolValueOrNull are shared across resources that model a
// nullable server-side bool as a Go *bool (compat-optional fields, fields the
// backend treats as Optional[bool] = None, etc.) - currently used by both
// resource_config_admin.go and resource_config_ldap.go. Per CLAUDE.md's
// "Shared cross-cutting helpers live at package level rather than
// per-resource" convention, they live here rather than in either resource
// file so that neither file can be deleted without silently breaking the
// other.
func boolPtr(value types.Bool) *bool {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	result := value.ValueBool()
	return &result
}

func boolValueOrNull(value *bool) types.Bool {
	if value == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*value)
}

// stringPtr, int64Ptr, stringValueOrNull and int64ValueOrNull are the
// string/int64 counterparts of boolPtr/boolValueOrNull: a null or unknown
// Terraform value maps to a nil pointer, and a nil pointer to a null value.
func stringPtr(value types.String) *string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	result := value.ValueString()
	return &result
}

func int64Ptr(value types.Int64) *int64 {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	result := value.ValueInt64()
	return &result
}

func stringValueOrNull(value *string) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(*value)
}

func int64ValueOrNull(value *int64) types.Int64 {
	if value == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*value)
}

// knownObject reports whether a nested object attribute holds a concrete
// (non-null, non-unknown) value.
func knownObject(v types.Object) bool { return !v.IsNull() && !v.IsUnknown() }
