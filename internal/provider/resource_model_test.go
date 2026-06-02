package provider

import (
	"context"
	"strings"
	"testing"
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

// Sanity check: a non-null description should not appear in additional either
// — it belongs in the structured state field only.
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

	// additional should be null (no extra keys) or at least not contain description
	if !additionalJSON.IsNull() && strings.Contains(additionalJSON.ValueString(), "description") {
		t.Errorf("description leaked into meta_additional_json: %s", additionalJSON.ValueString())
	}
}
