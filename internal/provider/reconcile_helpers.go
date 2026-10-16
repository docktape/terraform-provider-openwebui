package provider

import (
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
)

// reconcileUnconfiguredAttributes is shared by config resources that send
// only the attributes set in the configuration (every attribute
// Optional+Computed with UseStateForUnknown). For each top-level attribute of
// the flat model T (a struct of attr.Value fields with tfsdk tags) that is
// null in config while its planned value is known (including a known null),
// i.e. copied from the prior state, Terraform requires the applied state to
// equal that planned value.
// The server value can legitimately differ (changed outside Terraform after
// the last refresh, with -refresh=false, or by Open WebUI itself), and the
// apply did not change the attribute, so the planned value is kept and an attribute
// warning is added; the next refresh shows the server value. cause completes
// the warning's parenthesis with the resource-specific reasons.
//
// Values are deliberately never printed: several attributes are secrets.
func reconcileUnconfiguredAttributes[T any](config, plan, fresh T, cause string) (T, diag.Diagnostics) {
	return reconcileUnconfiguredAttributesAt(path.Empty(), config, plan, fresh, cause)
}

// reconcileUnconfiguredAttributesAt is reconcileUnconfiguredAttributes for
// the attributes of a nested object at base (path.Empty() for the top level),
// for example the inner attributes of a configured single nested attribute.
func reconcileUnconfiguredAttributesAt[T any](base path.Path, config, plan, fresh T, cause string) (T, diag.Diagnostics) {
	var diags diag.Diagnostics
	configValue := reflect.ValueOf(config)
	planValue := reflect.ValueOf(plan)
	freshValue := reflect.ValueOf(&fresh).Elem()
	modelType := configValue.Type()
	for i := 0; i < modelType.NumField(); i++ {
		name := modelType.Field(i).Tag.Get("tfsdk")
		if name == "id" {
			continue
		}
		configured, _ := configValue.Field(i).Interface().(attr.Value)
		planned, _ := planValue.Field(i).Interface().(attr.Value)
		got, _ := freshValue.Field(i).Interface().(attr.Value)
		if configured == nil || planned == nil || got == nil {
			continue
		}
		if !configured.IsNull() || planned.IsUnknown() || planned.Equal(got) {
			continue
		}
		freshValue.Field(i).Set(planValue.Field(i))
		attrPath := base.AtName(name)
		diags.AddAttributeWarning(attrPath, "Open WebUI value differs from state",
			fmt.Sprintf("%s is not set in the configuration, so this apply left it unchanged, but Open WebUI reports a different value than the state (%s). The state keeps the previous value for now; the next refresh shows the server value.", attrPath, cause))
	}
	return fresh, diags
}
