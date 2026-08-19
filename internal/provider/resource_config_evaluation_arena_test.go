package provider

import (
	"context"
	"testing"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestEvaluationArenaConfigModelRoundTrip(t *testing.T) {
	ctx := context.Background()

	// Create a client form with test data
	form := client.EvaluationArenaConfigForm{
		EnableEvaluationArenaModels: true,
		EvaluationArenaModels:       []string{"model-1", "model-2", "model-3"},
	}

	// Convert to Terraform model via flattenStringSlice
	modelList, listDiags := flattenStringSlice(ctx, form.EvaluationArenaModels)
	if listDiags.HasError() {
		t.Fatalf("flattenStringSlice failed: %v", listDiags)
	}

	model := evaluationArenaConfigModel{
		ID:                          types.StringValue("evaluation_arena"),
		EnableEvaluationArenaModels: types.BoolValue(form.EnableEvaluationArenaModels),
		EvaluationArenaModels:       modelList,
	}

	if !model.EnableEvaluationArenaModels.ValueBool() {
		t.Fatal("expected enable_evaluation_arena_models=true")
	}

	// Convert back to strings via expandStringList
	roundTripModels := expandStringList(ctx, model.EvaluationArenaModels, path.Root("evaluation_arena_models"), &listDiags)
	if listDiags.HasError() {
		t.Fatalf("expandStringList failed: %v", listDiags)
	}

	if len(roundTripModels) != 3 {
		t.Fatalf("expected 3 models, got %d", len(roundTripModels))
	}
	if roundTripModels[0] != "model-1" {
		t.Fatalf("expected model-1, got %s", roundTripModels[0])
	}
	if roundTripModels[1] != "model-2" {
		t.Fatalf("expected model-2, got %s", roundTripModels[1])
	}
	if roundTripModels[2] != "model-3" {
		t.Fatalf("expected model-3, got %s", roundTripModels[2])
	}
}
