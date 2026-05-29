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
	filterPermissionResponse("workspace", map[string]any{
		"unknown_key": true,
	}, &diags)
	if !diags.HasError() {
		t.Fatal("expected error for unknown key from API")
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
}
