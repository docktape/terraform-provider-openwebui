package provider

import (
	"testing"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

func TestSubagentsConfigFormModelRoundTrip(t *testing.T) {
	form := client.SubagentsConfigForm{
		EnableSubagents:            true,
		SubagentsBackgroundEnabled: false,
		SubagentsMaxConcurrent:     5,
		SubagentsMaxAsync:          3,
		SubagentsMaxIterations:     10,
		SubagentsMaxOutput:         2048,
		SubagentsSystemPrompt:      "You are a helpful agent.",
	}
	model := subagentsConfigModelFromForm(form)
	if !model.EnableSubagents.ValueBool() {
		t.Fatal("expected enable_subagents=true")
	}
	if model.SubagentsBackgroundEnabled.ValueBool() {
		t.Fatal("expected subagents_background_enabled=false")
	}
	if model.SubagentsMaxConcurrent.ValueInt64() != 5 {
		t.Fatalf("expected 5, got %d", model.SubagentsMaxConcurrent.ValueInt64())
	}
	if model.SubagentsMaxAsync.ValueInt64() != 3 {
		t.Fatalf("expected 3, got %d", model.SubagentsMaxAsync.ValueInt64())
	}
	if model.SubagentsMaxIterations.ValueInt64() != 10 {
		t.Fatalf("expected 10, got %d", model.SubagentsMaxIterations.ValueInt64())
	}
	if model.SubagentsMaxOutput.ValueInt64() != 2048 {
		t.Fatalf("expected 2048, got %d", model.SubagentsMaxOutput.ValueInt64())
	}
	if model.SubagentsSystemPrompt.ValueString() != "You are a helpful agent." {
		t.Fatalf("expected 'You are a helpful agent.', got %s", model.SubagentsSystemPrompt.ValueString())
	}

	roundTripped := subagentsConfigFormFromModel(model)
	if !roundTripped.EnableSubagents {
		t.Fatal("round-trip lost EnableSubagents")
	}
	if roundTripped.SubagentsBackgroundEnabled {
		t.Fatal("round-trip lost SubagentsBackgroundEnabled")
	}
	if roundTripped.SubagentsMaxConcurrent != 5 {
		t.Fatalf("round-trip lost SubagentsMaxConcurrent: %d", roundTripped.SubagentsMaxConcurrent)
	}
	if roundTripped.SubagentsMaxAsync != 3 {
		t.Fatalf("round-trip lost SubagentsMaxAsync: %d", roundTripped.SubagentsMaxAsync)
	}
	if roundTripped.SubagentsMaxIterations != 10 {
		t.Fatalf("round-trip lost SubagentsMaxIterations: %d", roundTripped.SubagentsMaxIterations)
	}
	if roundTripped.SubagentsMaxOutput != 2048 {
		t.Fatalf("round-trip lost SubagentsMaxOutput: %d", roundTripped.SubagentsMaxOutput)
	}
	if roundTripped.SubagentsSystemPrompt != "You are a helpful agent." {
		t.Fatalf("round-trip lost SubagentsSystemPrompt: %s", roundTripped.SubagentsSystemPrompt)
	}
}
