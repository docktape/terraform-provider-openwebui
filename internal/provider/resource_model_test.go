package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// --- mergeStringAnyMaps ---

func TestMergeStringAnyMaps_BothNil(t *testing.T) {
	result := mergeStringAnyMaps(nil, nil)
	if result != nil {
		t.Fatalf("expected nil, got %+v", result)
	}
}

func TestMergeStringAnyMaps_PrimaryNil(t *testing.T) {
	secondary := map[string]any{"a": "1"}
	result := mergeStringAnyMaps(nil, secondary)
	if result["a"] != "1" {
		t.Fatalf("expected a=1, got %v", result["a"])
	}
}

func TestMergeStringAnyMaps_SecondaryNil(t *testing.T) {
	primary := map[string]any{"a": "1"}
	result := mergeStringAnyMaps(primary, nil)
	if result["a"] != "1" {
		t.Fatalf("expected a=1, got %v", result["a"])
	}
}

// Primary must win when both maps share a key — explicit Terraform attributes
// should never be overwritten by the "additional" JSON blob.
func TestMergeStringAnyMaps_PrimaryWinsOnConflict(t *testing.T) {
	primary := map[string]any{"key": "primary-value", "only-primary": "yes"}
	secondary := map[string]any{"key": "secondary-value", "only-secondary": "yes"}
	result := mergeStringAnyMaps(primary, secondary)

	if result["key"] != "primary-value" {
		t.Fatalf("expected primary-value to win, got %v", result["key"])
	}
	if result["only-primary"] != "yes" {
		t.Fatalf("expected only-primary key present, got %v", result["only-primary"])
	}
	if result["only-secondary"] != "yes" {
		t.Fatalf("expected only-secondary key present, got %v", result["only-secondary"])
	}
}

// --- flattenModelMeta ---

// When the API returns description=null, the key must not appear in
// meta_additional_json — previously it would leak in as "description":null,
// causing the provider to report a plan/apply state mismatch.
func TestFlattenModelMeta_NullDescriptionNotLeakedToAdditional(t *testing.T) {
	ctx := context.Background()
	data := map[string]any{
		"description":       nil,
		"profile_image_url": nil,
		"extra_field":       "should-remain",
	}

	_, additionalJSON, diags := flattenModelMeta(ctx, data)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if additionalJSON.IsNull() {
		// No additional at all is fine only if extra_field is also absent, which
		// it shouldn't be — so fail with a clear message.
		t.Fatal("expected non-null additional JSON (extra_field should be present)")
	}

	raw := additionalJSON.ValueString()
	if strings.Contains(raw, "description") {
		t.Errorf("null description leaked into meta_additional_json: %s", raw)
	}
	if strings.Contains(raw, "profile_image_url") {
		t.Errorf("null profile_image_url leaked into meta_additional_json: %s", raw)
	}
	if !strings.Contains(raw, "extra_field") {
		t.Errorf("expected extra_field in meta_additional_json, got: %s", raw)
	}
}

// Sanity check: a non-null description should route to the structured state
// field only — not appear in additional at all.
func TestFlattenModelMeta_NonNullDescriptionNotLeakedToAdditional(t *testing.T) {
	ctx := context.Background()
	data := map[string]any{
		"description": "my model",
	}

	state, additionalJSON, diags := flattenModelMeta(ctx, data)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if state.Description.ValueString() != "my model" {
		t.Fatalf("expected Description=my model, got %q", state.Description.ValueString())
	}

	// No extra keys in the input — additional must be null, not an object
	// containing description. The && in the old guard was a bug: it would
	// silently pass even if a regression put description back in additional.
	if !additionalJSON.IsNull() {
		t.Errorf("expected null additional JSON when no extra keys, got: %s", additionalJSON.ValueString())
	}
}

// Symmetric test for profile_image_url — the fix applied to both fields and
// both need regression coverage.
func TestFlattenModelMeta_NonNullProfileImageURLNotLeakedToAdditional(t *testing.T) {
	ctx := context.Background()
	data := map[string]any{
		"profile_image_url": "https://example.com/img.png",
	}

	state, additionalJSON, diags := flattenModelMeta(ctx, data)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if state.ProfileImageURL.ValueString() != "https://example.com/img.png" {
		t.Fatalf("expected ProfileImageURL=https://example.com/img.png, got %q", state.ProfileImageURL.ValueString())
	}

	if !additionalJSON.IsNull() {
		t.Errorf("expected null additional JSON when no extra keys, got: %s", additionalJSON.ValueString())
	}
}

func TestFlattenModelMeta_Hidden(t *testing.T) {
	ctx := context.Background()
	data := map[string]any{
		"hidden": true,
	}
	state, _, diags := flattenModelMeta(ctx, data)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if state.Hidden.IsNull() {
		t.Fatal("expected non-null hidden")
	}
	if !state.Hidden.ValueBool() {
		t.Fatalf("expected hidden=true, got false")
	}
}

func TestFlattenModelMeta_HiddenNotLeakedToAdditional(t *testing.T) {
	ctx := context.Background()
	data := map[string]any{
		"hidden": true,
	}
	_, additionalJSON, diags := flattenModelMeta(ctx, data)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if !additionalJSON.IsNull() {
		t.Errorf("hidden must not appear in meta_additional_json, got: %s", additionalJSON.ValueString())
	}
}

func TestExpandModelMeta_Hidden(t *testing.T) {
	ctx := context.Background()
	plan := &modelResourceModel{
		Hidden:             types.BoolValue(true),
		SuggestionPrompts:  types.ListNull(types.StringType),
		Tags:               types.ListNull(types.StringType),
		ToolIDs:            types.ListNull(types.StringType),
		DefaultFeatureIDs:  types.ListNull(types.StringType),
		ProfileImageURL:    types.StringNull(),
		Description:        types.StringNull(),
		MetaAdditionalJSON: types.StringNull(),
		IsActive:           types.BoolNull(),
	}
	var diags diag.Diagnostics
	result := expandModelMeta(ctx, plan, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if result["hidden"] != true {
		t.Fatalf("expected meta[hidden]=true, got %v", result["hidden"])
	}
}

func TestFlattenModelMeta_HiddenFalse(t *testing.T) {
	ctx := context.Background()
	data := map[string]any{
		"hidden": false,
	}
	state, _, diags := flattenModelMeta(ctx, data)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if state.Hidden.IsNull() {
		t.Fatal("expected non-null hidden")
	}
	if state.Hidden.ValueBool() {
		t.Fatalf("expected hidden=false, got true")
	}
}

func TestFlattenModelMeta_HiddenAbsent(t *testing.T) {
	ctx := context.Background()
	data := map[string]any{
		"description": "no hidden key",
	}
	state, _, diags := flattenModelMeta(ctx, data)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if !state.Hidden.IsNull() {
		t.Fatalf("expected null hidden when key absent, got %v", state.Hidden)
	}
}

func TestFlattenModelMeta_HiddenNull(t *testing.T) {
	ctx := context.Background()
	data := map[string]any{
		"hidden": nil,
	}
	state, _, diags := flattenModelMeta(ctx, data)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if !state.Hidden.IsNull() {
		t.Fatalf("expected null hidden when API sends null, got %v", state.Hidden)
	}
}

func TestExpandModelMeta_HiddenFalse(t *testing.T) {
	ctx := context.Background()
	plan := &modelResourceModel{
		Hidden:             types.BoolValue(false),
		SuggestionPrompts:  types.ListNull(types.StringType),
		Tags:               types.ListNull(types.StringType),
		ToolIDs:            types.ListNull(types.StringType),
		DefaultFeatureIDs:  types.ListNull(types.StringType),
		ProfileImageURL:    types.StringNull(),
		Description:        types.StringNull(),
		MetaAdditionalJSON: types.StringNull(),
		IsActive:           types.BoolNull(),
	}
	var diags diag.Diagnostics
	result := expandModelMeta(ctx, plan, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	val, ok := result["hidden"]
	if !ok {
		t.Fatal("expected hidden key in result")
	}
	if val != false {
		t.Fatalf("expected hidden=false, got %v", val)
	}
}
