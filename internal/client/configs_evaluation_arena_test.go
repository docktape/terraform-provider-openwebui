package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetSetEvaluationArenaConfigRoundTrip(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ENABLE_EVALUATION_ARENA_MODELS":true,"EVALUATION_ARENA_MODELS":[{"id":"arena-1","name":"Arena One","meta":{"profile_image_url":"/favicon.png","description":"first arena model","model_ids":["model-1","model-2"]}},{"id":"arena-2","name":"Arena Two","meta":{"model_ids":["model-3"]}}]}`))
	}))
	t.Cleanup(server.Close)

	c, err := NewClient(server.URL, "test-token", false)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	got, err := c.GetEvaluationArenaConfig(context.Background())
	if err != nil {
		t.Fatalf("GetEvaluationArenaConfig: %v", err)
	}
	if !got.EnableEvaluationArenaModels {
		t.Fatal("expected EnableEvaluationArenaModels=true")
	}
	if len(got.EvaluationArenaModels) != 2 {
		t.Fatalf("expected 2 arena models, got %d", len(got.EvaluationArenaModels))
	}
	if got.EvaluationArenaModels[0].ID != "arena-1" {
		t.Fatalf("expected first model id=arena-1, got %v", got.EvaluationArenaModels[0].ID)
	}
	if got.EvaluationArenaModels[0].Name != "Arena One" {
		t.Fatalf("expected first model name=Arena One, got %v", got.EvaluationArenaModels[0].Name)
	}
	if got.EvaluationArenaModels[0].Meta == nil {
		t.Fatal("expected first model meta to be non-nil")
	}
	if got.EvaluationArenaModels[0].Meta.ProfileImageURL == nil || *got.EvaluationArenaModels[0].Meta.ProfileImageURL != "/favicon.png" {
		t.Fatalf("expected first model profile_image_url=/favicon.png, got %v", got.EvaluationArenaModels[0].Meta.ProfileImageURL)
	}
	if len(got.EvaluationArenaModels[0].Meta.ModelIDs) != 2 || got.EvaluationArenaModels[0].Meta.ModelIDs[0] != "model-1" {
		t.Fatalf("expected first model model_ids=[model-1 model-2], got %v", got.EvaluationArenaModels[0].Meta.ModelIDs)
	}
	if got.EvaluationArenaModels[1].ID != "arena-2" {
		t.Fatalf("expected second model id=arena-2, got %v", got.EvaluationArenaModels[1].ID)
	}

	updated, err := c.SetEvaluationArenaConfig(context.Background(), *got)
	if err != nil {
		t.Fatalf("SetEvaluationArenaConfig: %v", err)
	}
	if !updated.EnableEvaluationArenaModels {
		t.Fatal("expected round-tripped EnableEvaluationArenaModels=true")
	}
	if len(updated.EvaluationArenaModels) != 2 {
		t.Fatalf("expected round-tripped 2 arena models, got %d", len(updated.EvaluationArenaModels))
	}
	if updated.EvaluationArenaModels[0].ID != "arena-1" {
		t.Fatalf("expected round-tripped first model id=arena-1, got %v", updated.EvaluationArenaModels[0].ID)
	}
}
