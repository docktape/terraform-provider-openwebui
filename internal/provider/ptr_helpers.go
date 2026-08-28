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
