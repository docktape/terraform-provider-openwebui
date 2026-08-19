package provider

import (
	"context"
	"testing"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestEvaluationArenaConfigModelRoundTrip(t *testing.T) {
	ctx := context.Background()

	// Create a client form with test data
	profileImageURL := "/favicon.png"
	description := "first arena model"
	form := client.EvaluationArenaConfigForm{
		EnableEvaluationArenaModels: true,
		EvaluationArenaModels: []client.ArenaModel{
			{
				ID:   "arena-1",
				Name: "Arena One",
				Meta: &client.ArenaModelMeta{
					ProfileImageURL: &profileImageURL,
					Description:     &description,
					ModelIDs:        []string{"model-1", "model-2"},
				},
			},
			{
				ID:   "arena-2",
				Name: "Arena Two",
				Meta: &client.ArenaModelMeta{
					ModelIDs: []string{"model-3"},
				},
			},
		},
	}

	// Convert to Terraform model via flattenArenaModels
	modelList, listDiags := flattenArenaModels(ctx, form.EvaluationArenaModels)
	if listDiags.HasError() {
		t.Fatalf("flattenArenaModels failed: %v", listDiags)
	}

	model := evaluationArenaConfigModel{
		ID:                          types.StringValue("evaluation_arena"),
		EnableEvaluationArenaModels: types.BoolValue(form.EnableEvaluationArenaModels),
		EvaluationArenaModels:       modelList,
	}

	if !model.EnableEvaluationArenaModels.ValueBool() {
		t.Fatal("expected enable_evaluation_arena_models=true")
	}
	if len(model.EvaluationArenaModels) != 2 {
		t.Fatalf("expected 2 arena models, got %d", len(model.EvaluationArenaModels))
	}
	if model.EvaluationArenaModels[0].ID.ValueString() != "arena-1" {
		t.Fatalf("expected id=arena-1, got %s", model.EvaluationArenaModels[0].ID.ValueString())
	}
	if model.EvaluationArenaModels[0].Name.ValueString() != "Arena One" {
		t.Fatalf("expected name=Arena One, got %s", model.EvaluationArenaModels[0].Name.ValueString())
	}
	if model.EvaluationArenaModels[0].ProfileImageURL.ValueString() != "/favicon.png" {
		t.Fatalf("expected profile_image_url=/favicon.png, got %s", model.EvaluationArenaModels[0].ProfileImageURL.ValueString())
	}
	if model.EvaluationArenaModels[1].ProfileImageURL.IsNull() != true {
		t.Fatalf("expected second entry profile_image_url to be null, got %v", model.EvaluationArenaModels[1].ProfileImageURL)
	}

	// Convert back to client structs via expandArenaModels
	var diags diag.Diagnostics
	roundTripModels := expandArenaModels(ctx, model.EvaluationArenaModels, &diags)
	if diags.HasError() {
		t.Fatalf("expandArenaModels failed: %v", diags)
	}

	if len(roundTripModels) != 2 {
		t.Fatalf("expected 2 models, got %d", len(roundTripModels))
	}
	if roundTripModels[0].ID != "arena-1" {
		t.Fatalf("expected arena-1, got %s", roundTripModels[0].ID)
	}
	if roundTripModels[0].Meta == nil || len(roundTripModels[0].Meta.ModelIDs) != 2 || roundTripModels[0].Meta.ModelIDs[0] != "model-1" {
		t.Fatalf("expected round-tripped model_ids=[model-1 model-2], got %+v", roundTripModels[0].Meta)
	}
	if roundTripModels[1].ID != "arena-2" {
		t.Fatalf("expected arena-2, got %s", roundTripModels[1].ID)
	}
}
