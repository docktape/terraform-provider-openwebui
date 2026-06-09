package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestFilterPermissionKeys_KnownKey(t *testing.T) {
	var diags diag.Diagnostics
	result := filterPermissionKeys(
		"workspace",
		map[string]bool{"models": true, "tools": false},
		path.Root("permissions").AtName("workspace"),
		&diags,
	)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if result["models"] != true {
		t.Fatalf("expected models=true, got %v", result["models"])
	}
	if result["tools"] != false {
		t.Fatalf("expected tools=false, got %v", result["tools"])
	}
}

func TestFilterPermissionKeys_UnknownKey(t *testing.T) {
	var diags diag.Diagnostics
	result := filterPermissionKeys(
		"workspace",
		map[string]bool{"bad_key": true},
		path.Root("permissions").AtName("workspace"),
		&diags,
	)
	if !diags.HasError() {
		t.Fatal("expected error diagnostic for unsupported key")
	}
	if len(result) != 0 {
		t.Fatalf("expected empty result for invalid key, got %v", result)
	}
}

func TestFilterPermissionKeys_ChatKey(t *testing.T) {
	var diags diag.Diagnostics
	result := filterPermissionKeys(
		"chat",
		map[string]bool{"file_upload": true, "delete": false},
		path.Root("permissions").AtName("chat"),
		&diags,
	)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if result["file_upload"] != true || result["delete"] != false {
		t.Fatalf("unexpected result: %v", result)
	}
}

func TestFilterPermissionResponse_ValidBools(t *testing.T) {
	var diags diag.Diagnostics
	result := filterPermissionResponse("features", map[string]any{
		"web_search":       true,
		"image_generation": false,
	}, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if result["web_search"] != true || result["image_generation"] != false {
		t.Fatalf("unexpected result: %v", result)
	}
}

func TestFilterPermissionResponse_NonBool(t *testing.T) {
	var diags diag.Diagnostics
	filterPermissionResponse("features", map[string]any{
		"web_search": "yes", // wrong type
	}, &diags)
	if !diags.HasError() {
		t.Fatal("expected error for non-bool value")
	}
}

func TestFilterPermissionResponse_UnknownKey(t *testing.T) {
	var diags diag.Diagnostics
	result := filterPermissionResponse("workspace", map[string]any{
		"unknown_key": true,
	}, &diags)
	if diags.HasError() {
		t.Fatal("expected unknown keys to be silently skipped, not an error")
	}
	if len(result) != 0 {
		t.Fatalf("expected empty result for unknown key, got %v", result)
	}
}

func TestFilterPermissionResponse_MixedKnownAndUnknown(t *testing.T) {
	var diags diag.Diagnostics
	result := filterPermissionResponse("workspace", map[string]any{
		"models":     true,
		"future_key": false, // unknown — should be silently dropped
	}, &diags)
	if diags.HasError() {
		t.Fatal("unexpected diagnostics: unknown key should be silently skipped")
	}
	if !result["models"] {
		t.Fatalf("expected models=true, got %v", result)
	}
	if _, ok := result["future_key"]; ok {
		t.Fatal("expected future_key to be dropped, but it was kept")
	}
}

func TestExpandPermissions_Workspace(t *testing.T) {
	ctx := context.Background()
	wsMap, mapDiags := types.MapValueFrom(ctx, types.BoolType, map[string]bool{"models": true, "tools": false})
	if mapDiags.HasError() {
		t.Fatalf("setup: %s", mapDiags)
	}
	model := groupPermissionsModel{
		Workspace: wsMap,
		Sharing:   types.MapNull(types.BoolType),
		Chat:      types.MapNull(types.BoolType),
		Features:  types.MapNull(types.BoolType),
	}
	var diags diag.Diagnostics
	result := expandPermissions(ctx, model, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	ws, ok := result["workspace"].(map[string]any)
	if !ok {
		t.Fatalf("expected workspace map, got %T", result["workspace"])
	}
	if ws["models"] != true {
		t.Fatalf("expected models=true, got %v", ws["models"])
	}
}

func TestExpandPermissions_AllNull(t *testing.T) {
	ctx := context.Background()
	model := groupPermissionsModel{
		Workspace: types.MapNull(types.BoolType),
		Sharing:   types.MapNull(types.BoolType),
		Chat:      types.MapNull(types.BoolType),
		Features:  types.MapNull(types.BoolType),
	}
	var diags diag.Diagnostics
	result := expandPermissions(ctx, model, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if result != nil {
		t.Fatalf("expected nil for all-null model, got %+v", result)
	}
}

func TestFlattenPermissions_Nil(t *testing.T) {
	ctx := context.Background()
	model, diags := flattenPermissions(ctx, nil)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if !model.Workspace.IsNull() || !model.Sharing.IsNull() || !model.Chat.IsNull() || !model.Features.IsNull() {
		t.Fatal("expected all null for nil input")
	}
}

func TestFlattenPermissions_WithWorkspace(t *testing.T) {
	ctx := context.Background()
	perms := map[string]any{
		"workspace": map[string]any{
			"models": true,
			"tools":  false,
		},
	}
	model, diags := flattenPermissions(ctx, perms)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if model.Workspace.IsNull() {
		t.Fatal("expected non-null workspace")
	}
	var bools map[string]bool
	if err := model.Workspace.ElementsAs(ctx, &bools, false); err != nil {
		t.Fatalf("ElementsAs: %v", err)
	}
	if !bools["models"] {
		t.Fatalf("expected models=true, got %v", bools)
	}
	if bools["tools"] {
		t.Fatalf("expected tools=false, got %v", bools)
	}
}

func TestExpandFlattenPermissionsRoundTrip(t *testing.T) {
	ctx := context.Background()
	wsMap, _ := types.MapValueFrom(ctx, types.BoolType, map[string]bool{"models": true})
	chatMap, _ := types.MapValueFrom(ctx, types.BoolType, map[string]bool{"file_upload": true, "delete": false})
	original := groupPermissionsModel{
		Workspace: wsMap,
		Sharing:   types.MapNull(types.BoolType),
		Chat:      chatMap,
		Features:  types.MapNull(types.BoolType),
	}

	var expandDiags diag.Diagnostics
	expanded := expandPermissions(ctx, original, &expandDiags)
	if expandDiags.HasError() {
		t.Fatalf("expand: %s", expandDiags)
	}

	flattened, flatDiags := flattenPermissions(ctx, expanded)
	if flatDiags.HasError() {
		t.Fatalf("flatten: %s", flatDiags)
	}

	var wsBools map[string]bool
	if err := flattened.Workspace.ElementsAs(ctx, &wsBools, false); err != nil {
		t.Fatalf("workspace ElementsAs: %v", err)
	}
	if !wsBools["models"] {
		t.Fatalf("round-trip: expected models=true, got %v", wsBools)
	}

	var chatBools map[string]bool
	if err := flattened.Chat.ElementsAs(ctx, &chatBools, false); err != nil {
		t.Fatalf("chat ElementsAs: %v", err)
	}
	if !chatBools["file_upload"] || chatBools["delete"] {
		t.Fatalf("round-trip: unexpected chat values: %v", chatBools)
	}
}

func TestFilterPermissionKeys_SharingKey(t *testing.T) {
	var diags diag.Diagnostics
	result := filterPermissionKeys(
		"sharing",
		map[string]bool{"public_models": true},
		path.Root("permissions").AtName("sharing"),
		&diags,
	)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if result["public_models"] != true {
		t.Fatalf("expected public_models=true, got %v", result["public_models"])
	}
}

func TestFilterPermissionKeys_FeaturesKey(t *testing.T) {
	var diags diag.Diagnostics
	result := filterPermissionKeys(
		"features",
		map[string]bool{"web_search": true},
		path.Root("permissions").AtName("features"),
		&diags,
	)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if result["web_search"] != true {
		t.Fatalf("expected web_search=true, got %v", result["web_search"])
	}
}

func TestExpandPermissions_InvalidKey(t *testing.T) {
	ctx := context.Background()
	badMap, mapDiags := types.MapValueFrom(ctx, types.BoolType, map[string]bool{"bad_key": true})
	if mapDiags.HasError() {
		t.Fatalf("setup: %s", mapDiags)
	}
	model := groupPermissionsModel{
		Workspace: badMap,
		Sharing:   types.MapNull(types.BoolType),
		Chat:      types.MapNull(types.BoolType),
		Features:  types.MapNull(types.BoolType),
	}
	var diags diag.Diagnostics
	expandPermissions(ctx, model, &diags)
	if !diags.HasError() {
		t.Fatal("expected error diagnostic for unsupported key")
	}
}

func TestFlattenPermissions_EmptyMap(t *testing.T) {
	ctx := context.Background()
	model, diags := flattenPermissions(ctx, map[string]any{})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if !model.Workspace.IsNull() || !model.Sharing.IsNull() || !model.Chat.IsNull() || !model.Features.IsNull() {
		t.Fatal("expected all null for empty map input")
	}
}

func TestObjectToPermissionsModel_NullObject(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	obj := types.ObjectNull(permissionsAttrTypes())
	model := objectToPermissionsModel(ctx, obj, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if !model.Workspace.IsNull() || !model.Sharing.IsNull() || !model.Chat.IsNull() || !model.Features.IsNull() {
		t.Fatal("expected null-filled model for null object")
	}
}

func TestObjectToPermissionsModel_UnknownObject(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	obj := types.ObjectUnknown(permissionsAttrTypes())
	model := objectToPermissionsModel(ctx, obj, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if !model.Workspace.IsNull() || !model.Sharing.IsNull() || !model.Chat.IsNull() || !model.Features.IsNull() {
		t.Fatal("expected null-filled model for unknown object")
	}
}

func TestPermissionsModelToObjectRoundTrip(t *testing.T) {
	ctx := context.Background()
	wsMap, _ := types.MapValueFrom(ctx, types.BoolType, map[string]bool{"models": true, "tools": false})
	chatMap, _ := types.MapValueFrom(ctx, types.BoolType, map[string]bool{"file_upload": true})
	original := groupPermissionsModel{
		Workspace: wsMap,
		Sharing:   types.MapNull(types.BoolType),
		Chat:      chatMap,
		Features:  types.MapNull(types.BoolType),
	}

	obj, objDiags := permissionsModelToObject(ctx, original)
	if objDiags.HasError() {
		t.Fatalf("permissionsModelToObject: %s", objDiags)
	}

	var roundDiags diag.Diagnostics
	result := objectToPermissionsModel(ctx, obj, &roundDiags)
	if roundDiags.HasError() {
		t.Fatalf("objectToPermissionsModel: %s", roundDiags)
	}

	if !result.Workspace.Equal(original.Workspace) {
		t.Fatalf("workspace mismatch: got %v, want %v", result.Workspace, original.Workspace)
	}
	if !result.Sharing.Equal(original.Sharing) {
		t.Fatalf("sharing mismatch: got %v, want %v", result.Sharing, original.Sharing)
	}
	if !result.Chat.Equal(original.Chat) {
		t.Fatalf("chat mismatch: got %v, want %v", result.Chat, original.Chat)
	}
	if !result.Features.Equal(original.Features) {
		t.Fatalf("features mismatch: got %v, want %v", result.Features, original.Features)
	}
}

func TestPermissionsObjectSpecified_NullObject(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	obj := types.ObjectNull(permissionsAttrTypes())
	if permissionsObjectSpecified(ctx, obj, &diags) {
		t.Fatal("expected false for null object")
	}
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
}

func TestPermissionsObjectSpecified_PopulatedObject(t *testing.T) {
	ctx := context.Background()
	wsMap, _ := types.MapValueFrom(ctx, types.BoolType, map[string]bool{"models": true})
	model := groupPermissionsModel{
		Workspace: wsMap,
		Sharing:   types.MapNull(types.BoolType),
		Chat:      types.MapNull(types.BoolType),
		Features:  types.MapNull(types.BoolType),
	}
	obj, objDiags := permissionsModelToObject(ctx, model)
	if objDiags.HasError() {
		t.Fatalf("setup: %s", objDiags)
	}

	var diags diag.Diagnostics
	if !permissionsObjectSpecified(ctx, obj, &diags) {
		t.Fatal("expected true for populated object")
	}
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
}

func TestFilterPermissionKeys_NewWorkspaceKeys(t *testing.T) {
	var diags diag.Diagnostics
	result := filterPermissionKeys(
		"workspace",
		map[string]bool{"skills": true, "models_import": false, "tools_export": true},
		path.Root("permissions").AtName("workspace"),
		&diags,
	)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if result["skills"] != true || result["models_import"] != false || result["tools_export"] != true {
		t.Fatalf("unexpected result: %v", result)
	}
}

func TestFilterPermissionKeys_NewSharingKeys(t *testing.T) {
	var diags diag.Diagnostics
	result := filterPermissionKeys(
		"sharing",
		map[string]bool{"models": true, "skills": false, "public_skills": true, "notes": false},
		path.Root("permissions").AtName("sharing"),
		&diags,
	)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if result["models"] != true || result["skills"] != false || result["public_skills"] != true {
		t.Fatalf("unexpected result: %v", result)
	}
}

func TestFilterPermissionKeys_NewFeaturesKeys(t *testing.T) {
	var diags diag.Diagnostics
	result := filterPermissionKeys(
		"features",
		map[string]bool{"memories": true, "api_keys": false, "channels": true, "folders": false},
		path.Root("permissions").AtName("features"),
		&diags,
	)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if result["memories"] != true || result["api_keys"] != false {
		t.Fatalf("unexpected result: %v", result)
	}
}
