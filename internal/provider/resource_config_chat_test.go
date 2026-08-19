package provider

import (
	"testing"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

func TestChatConfigFormModelRoundTrip(t *testing.T) {
	form := client.ChatConfigForm{
		EnableContextCompaction:              true,
		ContextCompactionTokenThreshold:      8000,
		ContextCompactionTokenCap:            12000,
		ContextCompactionRetentionPercentage: 50.5,
		ContextCompactionPromptTemplate:      "compact prompt",
	}
	model := chatConfigModelFromForm(form)
	if !model.EnableContextCompaction.ValueBool() {
		t.Fatal("expected enable_context_compaction=true")
	}
	if model.ContextCompactionTokenThreshold.ValueInt64() != 8000 {
		t.Fatalf("expected 8000, got %d", model.ContextCompactionTokenThreshold.ValueInt64())
	}
	if model.ContextCompactionRetentionPercentage.ValueFloat64() != 50.5 {
		t.Fatalf("expected 50.5, got %f", model.ContextCompactionRetentionPercentage.ValueFloat64())
	}

	roundTripped := chatConfigFormFromModel(model)
	if !roundTripped.EnableContextCompaction {
		t.Fatal("round-trip lost EnableContextCompaction")
	}
	if roundTripped.ContextCompactionTokenThreshold != 8000 {
		t.Fatalf("round-trip lost ContextCompactionTokenThreshold: %d", roundTripped.ContextCompactionTokenThreshold)
	}
	if roundTripped.ContextCompactionRetentionPercentage != 50.5 {
		t.Fatalf("round-trip lost ContextCompactionRetentionPercentage: %f", roundTripped.ContextCompactionRetentionPercentage)
	}
}
