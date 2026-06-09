package provider

import (
	"context"
	"testing"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

func TestPromptResponseToModel_NewFields(t *testing.T) {
	ctx := context.Background()
	active := false
	resp := &client.PromptModel{
		ID:       "p1",
		Command:  "/greet",
		Name:     "Greet",
		Content:  "Hello!",
		IsActive: &active,
		Tags:     []string{"util", "test"},
	}

	state, diags := promptResponseToModel(ctx, nil, resp)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}

	if state.IsActive.IsNull() || state.IsActive.ValueBool() != false {
		t.Fatalf("expected is_active=false, got %v", state.IsActive)
	}

	var tags []string
	if err := state.Tags.ElementsAs(ctx, &tags, false); err != nil {
		t.Fatalf("ElementsAs tags: %v", err)
	}
	if len(tags) != 2 || tags[0] != "util" || tags[1] != "test" {
		t.Fatalf("unexpected tags: %v", tags)
	}

	if !state.DataJSON.IsNull() {
		t.Fatalf("expected data_json to be null when Data is nil, got %v", state.DataJSON)
	}
	if !state.MetaJSON.IsNull() {
		t.Fatalf("expected meta_json to be null when Meta is nil, got %v", state.MetaJSON)
	}
}

func TestPromptResponseToModel_NilIsActive(t *testing.T) {
	ctx := context.Background()
	resp := &client.PromptModel{
		ID:       "p2",
		Command:  "/test",
		Name:     "Test",
		Content:  "content",
		IsActive: nil,
		Tags:     nil,
	}

	state, diags := promptResponseToModel(ctx, nil, resp)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}

	// nil IsActive should default to true
	if state.IsActive.IsNull() || !state.IsActive.ValueBool() {
		t.Fatalf("expected is_active=true (default) when nil, got %v", state.IsActive)
	}

	// nil Tags should produce null list
	if !state.Tags.IsNull() {
		t.Fatalf("expected nil tags to produce null list, got %v", state.Tags)
	}
}
