package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestFlattenStringSlice_Nil(t *testing.T) {
	ctx := context.Background()
	result, diags := flattenStringSlice(ctx, nil)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if result.IsNull() {
		t.Fatalf("expected non-null empty list for nil input, got null")
	}
	if len(result.Elements()) != 0 {
		t.Fatalf("expected empty list, got %d elements", len(result.Elements()))
	}
}

func TestFlattenStringSlice_Empty(t *testing.T) {
	ctx := context.Background()
	result, diags := flattenStringSlice(ctx, []string{})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if result.IsNull() {
		t.Fatalf("expected non-null empty list for empty slice, got null")
	}
	if len(result.Elements()) != 0 {
		t.Fatalf("expected empty list, got %d elements", len(result.Elements()))
	}
}

func TestFlattenStringSlice_Values(t *testing.T) {
	ctx := context.Background()
	result, diags := flattenStringSlice(ctx, []string{"alpha", "beta"})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if result.IsNull() {
		t.Fatal("expected non-null list")
	}
	elems := result.Elements()
	if len(elems) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(elems))
	}
	expected := []string{"alpha", "beta"}
	for i, el := range elems {
		sv, ok := el.(types.String)
		if !ok {
			t.Fatalf("element %d is not types.String", i)
		}
		if sv.ValueString() != expected[i] {
			t.Fatalf("element %d: got %q, want %q", i, sv.ValueString(), expected[i])
		}
	}
}

func TestExpandStringList_Null(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	result := expandStringList(ctx, types.ListNull(types.StringType), path.Root("field"), &diags)
	if result != nil {
		t.Fatalf("expected nil for null list, got %+v", result)
	}
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
}

func TestExpandStringList_Unknown(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	result := expandStringList(ctx, types.ListUnknown(types.StringType), path.Root("field"), &diags)
	if result != nil {
		t.Fatalf("expected nil for unknown list, got %+v", result)
	}
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
}

func TestExpandStringList_Values(t *testing.T) {
	ctx := context.Background()
	list, listDiags := types.ListValueFrom(ctx, types.StringType, []string{"x", "y", "z"})
	if listDiags.HasError() {
		t.Fatalf("setup failed: %s", listDiags)
	}
	var diags diag.Diagnostics
	result := expandStringList(ctx, list, path.Root("field"), &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if len(result) != 3 || result[0] != "x" || result[1] != "y" || result[2] != "z" {
		t.Fatalf("expected [x y z], got %+v", result)
	}
}

func TestFlattenExpandRoundTrip(t *testing.T) {
	ctx := context.Background()
	original := []string{"one", "two", "three"}

	list, diags := flattenStringSlice(ctx, original)
	if diags.HasError() {
		t.Fatalf("flattenStringSlice: %s", diags)
	}

	var expandDiags diag.Diagnostics
	result := expandStringList(ctx, list, path.Root("field"), &expandDiags)
	if expandDiags.HasError() {
		t.Fatalf("expandStringList: %s", expandDiags)
	}

	if len(result) != len(original) {
		t.Fatalf("round-trip length mismatch: got %d, want %d", len(result), len(original))
	}
	for i, v := range original {
		if result[i] != v {
			t.Fatalf("round-trip mismatch at index %d: got %q, want %q", i, result[i], v)
		}
	}
}

func TestFlattenNullableStringSlice(t *testing.T) {
	ctx := context.Background()
	null, diags := flattenNullableStringSlice(ctx, nil)
	if diags.HasError() || !null.IsNull() {
		t.Fatalf("nil must flatten to null, got %v (%v)", null, diags)
	}
	empty, diags := flattenNullableStringSlice(ctx, []string{})
	if diags.HasError() || empty.IsNull() || len(empty.Elements()) != 0 {
		t.Fatalf("[] must flatten to an empty list, got %v (%v)", empty, diags)
	}
	values, diags := flattenNullableStringSlice(ctx, []string{"a"})
	if diags.HasError() || len(values.Elements()) != 1 {
		t.Fatalf("populated slice must flatten to a list, got %v (%v)", values, diags)
	}
}

func TestExpandStringListPtr(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	if got := expandStringListPtr(ctx, types.ListNull(types.StringType), path.Root("x"), &diags); got != nil {
		t.Errorf("null must expand to nil, got %v", *got)
	}
	if got := expandStringListPtr(ctx, types.ListUnknown(types.StringType), path.Root("x"), &diags); got != nil {
		t.Errorf("unknown must expand to nil, got %v", *got)
	}
	empty, _ := types.ListValueFrom(ctx, types.StringType, []string{})
	if got := expandStringListPtr(ctx, empty, path.Root("x"), &diags); got == nil || *got == nil || len(*got) != 0 {
		t.Errorf("[] must expand to a non-nil empty slice, got %v", got)
	}
	values, _ := types.ListValueFrom(ctx, types.StringType, []string{"a", "b"})
	if got := expandStringListPtr(ctx, values, path.Root("x"), &diags); got == nil || len(*got) != 2 {
		t.Errorf("populated list: got %v", got)
	}
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
}
