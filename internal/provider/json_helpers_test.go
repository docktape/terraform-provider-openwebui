package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestDecodeOptionalJSON_Null(t *testing.T) {
	var diags diag.Diagnostics
	result := decodeOptionalJSON(types.StringNull(), path.Root("field"), &diags)
	if result != nil {
		t.Fatalf("expected nil for null input, got %+v", result)
	}
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
}

func TestDecodeOptionalJSON_Unknown(t *testing.T) {
	var diags diag.Diagnostics
	result := decodeOptionalJSON(types.StringUnknown(), path.Root("field"), &diags)
	if result != nil {
		t.Fatalf("expected nil for unknown input, got %+v", result)
	}
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
}

func TestDecodeOptionalJSON_EmptyString(t *testing.T) {
	var diags diag.Diagnostics
	result := decodeOptionalJSON(types.StringValue(""), path.Root("field"), &diags)
	if result != nil {
		t.Fatalf("expected nil for empty string, got %+v", result)
	}
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
}

func TestDecodeOptionalJSON_ValidObject(t *testing.T) {
	var diags diag.Diagnostics
	result := decodeOptionalJSON(types.StringValue(`{"key":"val","num":42}`), path.Root("field"), &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if result == nil {
		t.Fatal("expected non-nil map")
	}
	if result["key"] != "val" {
		t.Fatalf("expected key=val, got %v", result["key"])
	}
	if result["num"] != float64(42) {
		t.Fatalf("expected num=42, got %v", result["num"])
	}
}

func TestDecodeOptionalJSON_InvalidJSON(t *testing.T) {
	var diags diag.Diagnostics
	result := decodeOptionalJSON(types.StringValue("not-json"), path.Root("field"), &diags)
	if result != nil {
		t.Fatalf("expected nil for invalid JSON, got %+v", result)
	}
	if !diags.HasError() {
		t.Fatal("expected error diagnostic for invalid JSON")
	}
}

func TestDecodeOptionalJSON_WhitespaceOnly(t *testing.T) {
	var diags diag.Diagnostics
	result := decodeOptionalJSON(types.StringValue("   "), path.Root("field"), &diags)
	if result != nil {
		t.Fatalf("expected nil for whitespace-only string, got %+v", result)
	}
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
}

func TestEncodeOptionalJSON_Nil(t *testing.T) {
	result, err := encodeOptionalJSON(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsNull() {
		t.Fatalf("expected null string for nil map, got %s", result.ValueString())
	}
}

func TestEncodeOptionalJSON_ValidMap(t *testing.T) {
	result, err := encodeOptionalJSON(map[string]any{"k": "v"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsNull() {
		t.Fatal("expected non-null string")
	}
	if result.ValueString() != `{"k":"v"}` {
		t.Fatalf("expected {\"k\":\"v\"}, got %s", result.ValueString())
	}
}

func TestEncodeOptionalJSON_EmptyMap(t *testing.T) {
	result, err := encodeOptionalJSON(map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ValueString() != `{}` {
		t.Fatalf("expected {}, got %s", result.ValueString())
	}
}

func TestEncodeOptionalJSONValue_Nil(t *testing.T) {
	result, err := encodeOptionalJSONValue(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsNull() {
		t.Fatalf("expected null for nil, got %s", result.ValueString())
	}
}

func TestEncodeOptionalJSONValue_String(t *testing.T) {
	result, err := encodeOptionalJSONValue("hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ValueString() != `"hello"` {
		t.Fatalf(`expected "hello", got %s`, result.ValueString())
	}
}

func TestEncodeOptionalJSONValue_Number(t *testing.T) {
	result, err := encodeOptionalJSONValue(42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ValueString() != `42` {
		t.Fatalf("expected 42, got %s", result.ValueString())
	}
}
