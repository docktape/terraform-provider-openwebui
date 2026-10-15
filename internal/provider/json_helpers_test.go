package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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

func TestEquivalentJSON(t *testing.T) {
	cases := []struct {
		name string
		a, b string
		want bool
	}{
		{"identical", `{"a":1}`, `{"a":1}`, true},
		{"key order and whitespace", `{ "b": 1,  "a": 2 }`, `{"a":2,"b":1}`, true},
		{"trailing zero decimal", `{"speed": 1.50}`, `{"speed":1.5}`, true},
		{"exponent vs decimal", `{"n": 1e2}`, `{"n": 100.0}`, true},
		{"integer vs float form", `{"n": 100}`, `{"n": 1E+2}`, true},
		{"negative exponent", `{"n": 15e-1}`, `{"n": 1.5}`, true},
		{"negative zero", `{"n": -0.0}`, `{"n": 0}`, true},
		{"different numbers", `{"speed": 1.5}`, `{"speed": 1.51}`, false},
		{"number vs string", `{"n": 1}`, `{"n": "1"}`, false},
		{"nested equal", `{"a": {"b": [1.0, {"c": 2.50}]}}`, `{"a":{"b":[1,{"c":2.5}]}}`, true},
		{"nested unequal", `{"a": {"b": [1.0, {"c": 2.50}]}}`, `{"a":{"b":[1,{"c":2.6}]}}`, false},
		{"array order matters", `[1, 2]`, `[2, 1]`, false},
		{"array length", `[1, 2]`, `[1, 2, 3]`, false},
		{"extra key", `{"a":1}`, `{"a":1,"b":null}`, false},
		{"bool and null", `{"a":true,"b":null}`, `{"b":null,"a":true}`, true},
		{"huge ints beyond float64 precision stay distinct", `{"n": 9007199254740993}`, `{"n": 9007199254740992}`, false},
		{"huge ints equal by value", `{"n": 9007199254740993}`, `{"n": 9007199254740993.0}`, true},
		{"absurd exponent compared as text", `{"n": 1e999999999}`, `{"n": 1e999999999}`, true},
		{"absurd exponent never expanded", `{"n": 1e999999999}`, `{"n": 10e999999998}`, false},
		{"absurd negative exponent", `{"n": 1e-999999999}`, `{"n": 0}`, false},
		{"invalid JSON compared verbatim", `{`, `{`, true},
		{"invalid vs valid", `{`, `{}`, false},
		{"trailing data is invalid", `{} {}`, `{}`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := equivalentJSON(tc.a, tc.b); got != tc.want {
				t.Errorf("equivalentJSON(%s, %s) = %v, want %v", tc.a, tc.b, got, tc.want)
			}
			if got := equivalentJSON(tc.b, tc.a); got != tc.want {
				t.Errorf("equivalentJSON(%s, %s) = %v, want %v (reversed)", tc.b, tc.a, got, tc.want)
			}
		})
	}
}

// canonicalJSON is what goes on the wire: numbers stay verbatim.
func TestCanonicalJSONKeepsNumberLiterals(t *testing.T) {
	got, err := canonicalJSON([]byte(`{ "speed": 1.50, "big": 9007199254740993 }`))
	if err != nil {
		t.Fatal(err)
	}
	if got != `{"big":9007199254740993,"speed":1.50}` {
		t.Errorf("unexpected canonical form %s", got)
	}
}

// TestJSONShapeValidatorNumbers checks the plan-time rule that every JSON
// number literal must come back from Open WebUI's Python json round trip with
// the same value: integers are exact at any (reasonable) size, fractions and
// exponents must equal the float64 Python stores.
func TestJSONShapeValidatorNumbers(t *testing.T) {
	ctx := context.Background()
	hugeInt := strings.Repeat("9", 400)
	tooLongInt := strings.Repeat("1", pythonMaxIntDigits+1)
	cases := []struct {
		text    string
		wantErr bool
	}{
		{`{"a": 1}`, false},
		{`{"a": 1.5, "b": 0.25, "c": -0.0, "d": 1e2, "e": 1.50}`, false},
		{`{"a": 0.1, "b": 2.5e-3, "c": 1E+2}`, false},
		{`{"big": 9007199254740993}`, false},
		{`{"big": ` + hugeInt + `}`, false},
		{`{"neg": -` + hugeInt + `}`, false},
		{`{"a": 5e-324}`, false},
		{`{"a": 1.7976931348623157e308}`, false},
		{`{"nested": [{"x": [1, 2.5]}]}`, false},
		{`{"big": 9007199254740993.0}`, true},
		{`{"big": 9.007199254740993e15}`, true},
		{`{"a": 0.10000000000000000001}`, true},
		{`{"a": 1e400}`, true},
		{`{"a": -1e400}`, true},
		{`{"a": 1e-400}`, true},
		{`{"a": 1e1001}`, true},
		{`{"a": 0e2000}`, true},
		{`{"nested": {"deep": [1, 1e400]}}`, true},
		{`{"a": ` + tooLongInt + `}`, true},
	}
	for _, tc := range cases {
		resp := &validator.StringResponse{}
		jsonObjectValidator().ValidateString(ctx, validator.StringRequest{Path: path.Root("x"), ConfigValue: types.StringValue(tc.text)}, resp)
		if got := resp.Diagnostics.HasError(); got != tc.wantErr {
			t.Errorf("%.80s: error=%v, want %v (%v)", tc.text, got, tc.wantErr, resp.Diagnostics)
		}
	}

	resp := &validator.StringResponse{}
	jsonArrayOfObjectsValidator().ValidateString(ctx, validator.StringRequest{Path: path.Root("x"), ConfigValue: types.StringValue(`[{"a": 9007199254740993.0}]`)}, resp)
	if !resp.Diagnostics.HasError() {
		t.Errorf("array-of-objects validator accepted an unrepresentable fraction")
	}
}

// TestJSONShapeValidatorShapes keeps the top-level shape checks.
func TestJSONShapeValidatorShapes(t *testing.T) {
	ctx := context.Background()
	check := func(v validator.String, text string, wantErr bool) {
		t.Helper()
		resp := &validator.StringResponse{}
		v.ValidateString(ctx, validator.StringRequest{Path: path.Root("x"), ConfigValue: types.StringValue(text)}, resp)
		if got := resp.Diagnostics.HasError(); got != wantErr {
			t.Errorf("%q: error=%v, want %v", text, got, wantErr)
		}
	}
	for text, wantErr := range map[string]bool{`{}`: false, `{"a":1}`: false, `null`: true, `[]`: true, `"x"`: true, `{} {}`: true, `{`: true, `1`: true} {
		check(jsonObjectValidator(), text, wantErr)
	}
	for text, wantErr := range map[string]bool{`[]`: false, `[{}]`: false, `[{"a":1},{}]`: false, `[null]`: true, `[1]`: true, `{}`: true, `null`: true} {
		check(jsonArrayOfObjectsValidator(), text, wantErr)
	}
	for _, v := range []types.String{types.StringNull(), types.StringUnknown()} {
		resp := &validator.StringResponse{}
		jsonObjectValidator().ValidateString(ctx, validator.StringRequest{Path: path.Root("x"), ConfigValue: v}, resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("validator rejected %s", v)
		}
	}
}
