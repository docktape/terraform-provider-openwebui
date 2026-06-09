package provider

import (
	"context"
	"sort"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceSchema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ---------------------------------------------------------------------------
// nil-vs-empty slice behaviour that caused the "inconsistent result" bug
// ---------------------------------------------------------------------------

// TestFetchUsernamesForIDs_NilSliceProducesNullList documents why the old
// `var names []string` initialisation was wrong: passing a nil slice to
// types.ListValueFrom produces a *null* list, not an empty list.  When the
// Terraform plan expected an empty list (users = []), Terraform reported
// "provider produced inconsistent result after apply".
func TestFetchUsernamesForIDs_NilSliceProducesNullList(t *testing.T) {
	ctx := context.Background()
	var nilSlice []string
	list, diags := types.ListValueFrom(ctx, types.StringType, nilSlice)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if !list.IsNull() {
		t.Error("expected nil []string to produce a null list (documents old buggy behaviour)")
	}
}

// TestFetchUsernamesForIDs_EmptySliceProducesEmptyList confirms the fix:
// make([]string, 0) yields an empty (non-null) list, which is consistent
// with a plan that contains users = [].
func TestFetchUsernamesForIDs_EmptySliceProducesEmptyList(t *testing.T) {
	ctx := context.Background()
	emptySlice := make([]string, 0)
	list, diags := types.ListValueFrom(ctx, types.StringType, emptySlice)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if list.IsNull() {
		t.Error("empty []string must not produce a null list")
	}
	if list.IsUnknown() {
		t.Error("empty []string must not produce an unknown list")
	}
	if len(list.Elements()) != 0 {
		t.Errorf("expected 0 elements, got %d", len(list.Elements()))
	}
}

// ---------------------------------------------------------------------------
// uniqueStrings
// ---------------------------------------------------------------------------

func TestUniqueStrings_Nil(t *testing.T) {
	got := uniqueStrings(nil)
	if got != nil {
		t.Fatalf("expected nil for nil input, got %v", got)
	}
}

func TestUniqueStrings_Empty(t *testing.T) {
	got := uniqueStrings([]string{})
	// empty (non-nil) slice in → empty slice out, deduplication is a no-op
	if len(got) != 0 {
		t.Fatalf("expected empty slice, got %v", got)
	}
}

func TestUniqueStrings_NoDuplicates(t *testing.T) {
	in := []string{"a", "b", "c"}
	got := uniqueStrings(in)
	if len(got) != 3 {
		t.Fatalf("expected 3 elements, got %v", got)
	}
}

func TestUniqueStrings_WithDuplicates(t *testing.T) {
	in := []string{"x", "y", "x", "z", "y", "x"}
	got := uniqueStrings(in)
	if len(got) != 3 {
		t.Fatalf("expected 3 unique elements, got %v", got)
	}
	// order must be preserved for first occurrence
	if got[0] != "x" || got[1] != "y" || got[2] != "z" {
		t.Fatalf("unexpected order: %v", got)
	}
}

// ---------------------------------------------------------------------------
// diffStringSets
// ---------------------------------------------------------------------------

func sortedStrings(s []string) []string {
	cp := append([]string(nil), s...)
	sort.Strings(cp)
	return cp
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestDiffStringSets_NoChange(t *testing.T) {
	add, remove := diffStringSets([]string{"a", "b"}, []string{"a", "b"})
	if len(add) != 0 {
		t.Errorf("expected nothing to add, got %v", add)
	}
	if len(remove) != 0 {
		t.Errorf("expected nothing to remove, got %v", remove)
	}
}

func TestDiffStringSets_AddNew(t *testing.T) {
	add, remove := diffStringSets([]string{"a"}, []string{"a", "b", "c"})
	if !equalStringSlices(sortedStrings(add), []string{"b", "c"}) {
		t.Errorf("expected [b c] to add, got %v", add)
	}
	if len(remove) != 0 {
		t.Errorf("expected nothing to remove, got %v", remove)
	}
}

func TestDiffStringSets_RemoveExisting(t *testing.T) {
	add, remove := diffStringSets([]string{"a", "b", "c"}, []string{"a"})
	if len(add) != 0 {
		t.Errorf("expected nothing to add, got %v", add)
	}
	if !equalStringSlices(sortedStrings(remove), []string{"b", "c"}) {
		t.Errorf("expected [b c] to remove, got %v", remove)
	}
}

func TestDiffStringSets_AddAndRemove(t *testing.T) {
	add, remove := diffStringSets([]string{"a", "b"}, []string{"b", "c"})
	if !equalStringSlices(sortedStrings(add), []string{"c"}) {
		t.Errorf("expected [c] to add, got %v", add)
	}
	if !equalStringSlices(sortedStrings(remove), []string{"a"}) {
		t.Errorf("expected [a] to remove, got %v", remove)
	}
}

func TestDiffStringSets_EmptyToEmpty(t *testing.T) {
	add, remove := diffStringSets([]string{}, []string{})
	if len(add) != 0 || len(remove) != 0 {
		t.Errorf("both empty: add=%v remove=%v", add, remove)
	}
}

func TestDiffStringSets_NilToNil(t *testing.T) {
	add, remove := diffStringSets(nil, nil)
	if len(add) != 0 || len(remove) != 0 {
		t.Errorf("both nil: add=%v remove=%v", add, remove)
	}
}

func TestPermissionsRoundTrip_AccessGrantsSettings(t *testing.T) {
	ctx := context.Background()
	agMap, _ := types.MapValueFrom(ctx, types.BoolType, map[string]bool{"allow_users": true})
	stMap, _ := types.MapValueFrom(ctx, types.BoolType, map[string]bool{"interface": false})
	original := groupPermissionsModel{
		Workspace:    types.MapNull(types.BoolType),
		Sharing:      types.MapNull(types.BoolType),
		Chat:         types.MapNull(types.BoolType),
		Features:     types.MapNull(types.BoolType),
		AccessGrants: agMap,
		Settings:     stMap,
	}
	obj, diags := permissionsModelToObject(ctx, original)
	if diags.HasError() {
		t.Fatalf("modelToObject: %s", diags)
	}
	var roundDiags diag.Diagnostics
	result := objectToPermissionsModel(ctx, obj, &roundDiags)
	if roundDiags.HasError() {
		t.Fatalf("objectToModel: %s", roundDiags)
	}
	if !result.AccessGrants.Equal(original.AccessGrants) {
		t.Fatalf("access_grants mismatch")
	}
	if !result.Settings.Equal(original.Settings) {
		t.Fatalf("settings mismatch")
	}
}

// ---------------------------------------------------------------------------
// Group data source schema completeness
// ---------------------------------------------------------------------------

func TestGroupDataSourceSchema_HasAllPermissionSubMaps(t *testing.T) {
	ctx := context.Background()
	ds := NewGroupDataSource().(*groupDataSource)
	var resp datasource.SchemaResponse
	ds.Schema(ctx, datasource.SchemaRequest{}, &resp)

	permsRaw, ok := resp.Schema.Attributes["permissions"]
	if !ok {
		t.Fatal("permissions attribute missing from group data source schema")
	}
	perms, ok := permsRaw.(datasourceSchema.SingleNestedAttribute)
	if !ok {
		t.Fatal("permissions is not a SingleNestedAttribute in group data source schema")
	}
	required := []string{"workspace", "sharing", "chat", "features", "access_grants", "settings"}
	for _, key := range required {
		if _, exists := perms.Attributes[key]; !exists {
			t.Errorf("permissions.%s is missing from the group data source schema", key)
		}
	}
}
