package provider

import (
	"testing"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

func TestTasksConfigFormModelRoundTrip(t *testing.T) {
	form := client.TaskConfigForm{
		TaskModel:                            strPtr("gpt-4o"),
		EnableTitleGeneration:                true,
		TitleGenerationPromptTemplate:        "t",
		AutocompleteGenerationInputMaxLength: 42,
	}
	model := tasksConfigModelFromForm(form)
	if model.TaskModel.ValueString() != "gpt-4o" {
		t.Fatalf("expected task_model=gpt-4o, got %s", model.TaskModel.ValueString())
	}
	if model.AutocompleteGenerationInputMaxLength.ValueInt64() != 42 {
		t.Fatalf("expected 42, got %d", model.AutocompleteGenerationInputMaxLength.ValueInt64())
	}

	roundTripped := tasksConfigFormFromModel(model)
	if roundTripped.TaskModel == nil || *roundTripped.TaskModel != "gpt-4o" {
		t.Fatalf("round-trip lost TaskModel: %v", roundTripped.TaskModel)
	}
}

func strPtr(s string) *string { return &s }
