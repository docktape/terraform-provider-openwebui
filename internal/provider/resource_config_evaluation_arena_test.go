package provider

import (
	"context"
	"testing"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestEvaluationArenaConfigModelRoundTrip(t *testing.T) {
	ctx := context.Background()

	// Create a client form with test data
	profileImageURL := "/favicon.png"
	description := "first arena model"
	enableArenaModels := true
	form := client.EvaluationArenaConfigForm{
		EnableEvaluationArenaModels: &enableArenaModels,
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
		EnableEvaluationArenaModels: boolValueOrNull(form.EnableEvaluationArenaModels),
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

// TestFlattenArenaModelsPreservesNullModelIDs guards against a regression
// where flattenArenaModels unconditionally called flattenStringSlice, which
// always coerces a nil/empty Go slice into an empty (non-null) Terraform
// list. model_ids is schema.Optional but not Computed, so when a config
// omits it Terraform's plan is null and the provider must echo back exactly
// null after apply, or Terraform fails with "Provider produced inconsistent
// result after apply". The real Open WebUI backend's
// DEFAULT_ARENA_MODEL.meta.model_ids defaults to None (Meta.ModelIDs == nil
// after JSON unmarshal), so this is the common case, not an edge case.
func TestFlattenArenaModelsPreservesNullModelIDs(t *testing.T) {
	ctx := context.Background()

	models := []client.ArenaModel{
		{
			ID:   "arena-1",
			Name: "Arena One",
			Meta: &client.ArenaModelMeta{
				// ModelIDs deliberately left nil, mirroring a backend
				// response where "model_ids" was absent or JSON null.
			},
		},
	}

	result, diags := flattenArenaModels(ctx, models)
	if diags.HasError() {
		t.Fatalf("flattenArenaModels failed: %v", diags)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 arena model, got %d", len(result))
	}
	if !result[0].ModelIDs.IsNull() {
		t.Fatalf("expected model_ids to be null when Meta.ModelIDs is nil, got %v", result[0].ModelIDs)
	}
}

// TestFlattenArenaModelsPreservesEmptyModelIDs is the mirror of
// TestFlattenArenaModelsPreservesNullModelIDs: when a config explicitly sets
// model_ids = [], expandArenaModels produces a non-nil, zero-length Go slice,
// and (with the client's ModelIDs omitempty removed) the backend echoes back
// "model_ids": [] rather than dropping the key. flattenArenaModels must turn
// that non-nil-but-empty slice into a non-null, empty Terraform list - not
// null - or Terraform reports "Provider produced inconsistent result after
// apply" for a plan that expected an empty list.
func TestFlattenArenaModelsPreservesEmptyModelIDs(t *testing.T) {
	ctx := context.Background()

	models := []client.ArenaModel{
		{
			ID:   "arena-1",
			Name: "Arena One",
			Meta: &client.ArenaModelMeta{
				ModelIDs: []string{},
			},
		},
	}

	result, diags := flattenArenaModels(ctx, models)
	if diags.HasError() {
		t.Fatalf("flattenArenaModels failed: %v", diags)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 arena model, got %d", len(result))
	}
	if result[0].ModelIDs.IsNull() {
		t.Fatal("expected model_ids to be a non-null empty list when Meta.ModelIDs is non-nil-but-empty, got null")
	}
	if len(result[0].ModelIDs.Elements()) != 0 {
		t.Fatalf("expected model_ids to be empty, got %v", result[0].ModelIDs.Elements())
	}
}

// TestExpandArenaModelsProducesNonNilEmptyModelIDs guards the write-side
// half of the same fix: a Terraform-level model_ids = [] must expand to a
// non-nil, zero-length Go slice (not nil), so that once the client marshals
// it without omitempty the wire payload carries "model_ids": [] rather than
// omitting the key entirely.
func TestExpandArenaModelsProducesNonNilEmptyModelIDs(t *testing.T) {
	ctx := context.Background()

	emptyList, listDiags := types.ListValue(types.StringType, []attr.Value{})
	if listDiags.HasError() {
		t.Fatalf("types.ListValue failed: %v", listDiags)
	}

	items := []arenaModelModel{
		{
			ID:              types.StringValue("arena-1"),
			Name:            types.StringValue("Arena One"),
			ProfileImageURL: types.StringNull(),
			Description:     types.StringNull(),
			ModelIDs:        emptyList,
		},
	}

	var diags diag.Diagnostics
	result := expandArenaModels(ctx, items, &diags)
	if diags.HasError() {
		t.Fatalf("expandArenaModels failed: %v", diags)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 arena model, got %d", len(result))
	}
	if result[0].Meta == nil {
		t.Fatal("expected Meta to be non-nil")
	}
	if result[0].Meta.ModelIDs == nil {
		t.Fatal("expected ModelIDs to be a non-nil empty slice, got nil")
	}
	if len(result[0].Meta.ModelIDs) != 0 {
		t.Fatalf("expected ModelIDs to be empty, got %v", result[0].Meta.ModelIDs)
	}
}
