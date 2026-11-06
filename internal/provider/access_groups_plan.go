package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Resources exposing read_groups/write_groups send every write group as a read
// group too (see buildAccessControl): Open WebUI v0.11.0 does not treat write
// access as read access, so a write-only group could not see the resource.
// Open WebUI therefore returns read_groups as a superset of write_groups, and
// the helpers below keep the planned values consistent with that.

var (
	accessReadGroupsPath  = path.Root("read_groups")
	accessWriteGroupsPath = path.Root("write_groups")
)

// accessReadGroupsDescription returns the shared read_groups description.
// resourceLabel names the resource in the text, e.g. "tool".
func accessReadGroupsDescription(resourceLabel string) string {
	return fmt.Sprintf("Names of groups granted read access to the %s, spelled exactly as stored in Open WebUI (state holds the stored group "+
		"name, so an ID or a different letter case shows up as a difference). Open WebUI stores every write group as a read group too, so "+
		"this list always includes the `write_groups` entries. When `read_groups` is not set, its planned value keeps the current read "+
		"groups and is extended with any `write_groups` entry missing from it. When it is set, it must include every write group: to "+
		"remove a group from `read_groups`, remove it from `write_groups` too (or set `write_groups = []`). Set both lists to `[]` for "+
		"no group grants.", resourceLabel)
}

// accessWriteGroupsDescription returns the shared write_groups description.
func accessWriteGroupsDescription(resourceLabel string) string {
	return fmt.Sprintf("Names of groups granted write access to the %s, spelled exactly as stored in Open WebUI. Open WebUI stores these "+
		"groups as read groups too: when `read_groups` is set it must list every group here, and when it is not set it is extended with "+
		"them automatically. Leaving `write_groups` unset keeps the current write groups; set it to `[]` to revoke write access.", resourceLabel)
}

// accessGroupKey normalises a group reference the same way the group-name
// resolver matches names: surrounding spaces are ignored and case is folded.
func accessGroupKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

// knownStringElements returns the list's elements when the list is known,
// non-null and every element is a known, non-null string.
func knownStringElements(list types.List) ([]string, bool) {
	if list.IsNull() || list.IsUnknown() {
		return nil, false
	}
	out := make([]string, 0, len(list.Elements()))
	for _, e := range list.Elements() {
		sv, ok := e.(types.String)
		if !ok || sv.IsNull() || sv.IsUnknown() {
			return nil, false
		}
		out = append(out, sv.ValueString())
	}
	return out, true
}

// missingWriteGroups returns, in write order and without duplicates, the
// non-blank write groups that are not in read (compared via accessGroupKey).
func missingWriteGroups(read, write []string) []string {
	have := make(map[string]struct{}, len(read)+len(write))
	for _, r := range read {
		have[accessGroupKey(r)] = struct{}{}
	}
	var missing []string
	for _, w := range write {
		key := accessGroupKey(w)
		if key == "" {
			continue
		}
		if _, ok := have[key]; ok {
			continue
		}
		have[key] = struct{}{}
		missing = append(missing, w)
	}
	return missing
}

func quoteGroupNames(names []string) string {
	quoted := make([]string, len(names))
	for i, n := range names {
		quoted[i] = fmt.Sprintf("%q", n)
	}
	return strings.Join(quoted, ", ")
}

// validateAccessGroupsConfig rejects a configuration where read_groups and
// write_groups are both fully known and a write group is not a read group.
func validateAccessGroupsConfig(ctx context.Context, config tfsdk.Config, diags *diag.Diagnostics) {
	var read, write types.List
	diags.Append(config.GetAttribute(ctx, accessReadGroupsPath, &read)...)
	diags.Append(config.GetAttribute(ctx, accessWriteGroupsPath, &write)...)
	if diags.HasError() {
		return
	}

	readNames, ok := knownStringElements(read)
	if !ok {
		return
	}
	writeNames, ok := knownStringElements(write)
	if !ok {
		return
	}

	if missing := missingWriteGroups(readNames, writeNames); len(missing) > 0 {
		diags.AddAttributeError(
			accessWriteGroupsPath,
			"write_groups must also be listed in read_groups",
			fmt.Sprintf("Open WebUI stores every write group as a read group too, so a write group missing from read_groups would come back "+
				"in read_groups after apply. Add these write_groups entries to read_groups: %s.", quoteGroupNames(missing)),
		)
	}
}

// modifyAccessGroupsPlan keeps the planned write_groups a subset of the planned
// read_groups. It runs after the attribute plan modifiers, so values carried
// over from state by UseStateForUnknown are already in the plan.
//
// When read_groups is not configured, the missing write groups are appended to
// the planned read_groups (allowed because the attribute is Computed and null
// in config). When read_groups is configured, its planned value must equal the
// configuration, so the conflict (a write group kept from state that the
// configured read_groups no longer lists) is reported as a plan error.
func modifyAccessGroupsPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var configRead, configWrite, plannedRead, plannedWrite types.List
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, accessReadGroupsPath, &configRead)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, accessWriteGroupsPath, &configWrite)...)
	resp.Diagnostics.Append(resp.Plan.GetAttribute(ctx, accessReadGroupsPath, &plannedRead)...)
	resp.Diagnostics.Append(resp.Plan.GetAttribute(ctx, accessWriteGroupsPath, &plannedWrite)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readNames, ok := knownStringElements(plannedRead)
	if !ok {
		return
	}
	writeNames, ok := knownStringElements(plannedWrite)
	if !ok {
		return
	}

	missing := missingWriteGroups(readNames, writeNames)
	if len(missing) == 0 {
		return
	}

	if configRead.IsNull() {
		elems := make([]attr.Value, 0, len(readNames)+len(missing))
		for _, n := range append(readNames, missing...) {
			elems = append(elems, types.StringValue(n))
		}
		extended, d := types.ListValue(types.StringType, elems)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, accessReadGroupsPath, extended)...)
		return
	}

	source := "The write group(s) %s are"
	if configWrite.IsNull() {
		source = "The write group(s) %s, kept from the current state because write_groups is not set, are"
	}
	resp.Diagnostics.AddAttributeError(
		accessWriteGroupsPath,
		"write_groups must also be listed in read_groups",
		fmt.Sprintf(source+" not in the configured read_groups. Open WebUI stores every write group as a read group too. "+
			"Set write_groups explicitly (for example `write_groups = []` to revoke write access) or add these groups to read_groups.",
			quoteGroupNames(missing)),
	)
}
