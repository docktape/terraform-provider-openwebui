package provider

import (
	"testing"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

func TestChatConfigFormModelRoundTrip(t *testing.T) {
	tokenCap := int64(12000)
	form := client.ChatConfigForm{
		EnableContextCompaction:              true,
		ContextCompactionTokenThreshold:      8000,
		ContextCompactionTokenCap:            &tokenCap,
		ContextCompactionRetentionPercentage: 40,
		ContextCompactionPromptTemplate:      "compact prompt",
	}
	model := chatConfigModelFromForm(form)
	if !model.EnableContextCompaction.ValueBool() {
		t.Fatal("expected enable_context_compaction=true")
	}
	if model.ContextCompactionTokenThreshold.ValueInt64() != 8000 {
		t.Fatalf("expected 8000, got %d", model.ContextCompactionTokenThreshold.ValueInt64())
	}
	if model.ContextCompactionRetentionPercentage.ValueInt64() != 40 {
		t.Fatalf("expected 40, got %d", model.ContextCompactionRetentionPercentage.ValueInt64())
	}

	roundTripped := chatConfigFormFromModel(model)
	if !roundTripped.EnableContextCompaction {
		t.Fatal("round-trip lost EnableContextCompaction")
	}
	if roundTripped.ContextCompactionTokenThreshold != 8000 {
		t.Fatalf("round-trip lost ContextCompactionTokenThreshold: %d", roundTripped.ContextCompactionTokenThreshold)
	}
	if roundTripped.ContextCompactionRetentionPercentage != 40 {
		t.Fatalf("round-trip lost ContextCompactionRetentionPercentage: %d", roundTripped.ContextCompactionRetentionPercentage)
	}
}
