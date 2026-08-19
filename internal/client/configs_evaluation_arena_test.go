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
		_, _ = w.Write([]byte(`{"ENABLE_EVALUATION_ARENA_MODELS":true,"EVALUATION_ARENA_MODELS":["model-1","model-2","model-3"]}`))
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
	if len(got.EvaluationArenaModels) != 3 {
		t.Fatalf("expected 3 arena models, got %d", len(got.EvaluationArenaModels))
	}
	if got.EvaluationArenaModels[0] != "model-1" {
		t.Fatalf("expected first model=model-1, got %v", got.EvaluationArenaModels[0])
	}

	updated, err := c.SetEvaluationArenaConfig(context.Background(), *got)
	if err != nil {
		t.Fatalf("SetEvaluationArenaConfig: %v", err)
	}
	if !updated.EnableEvaluationArenaModels {
		t.Fatal("expected round-tripped EnableEvaluationArenaModels=true")
	}
	if len(updated.EvaluationArenaModels) != 3 {
		t.Fatalf("expected round-tripped 3 arena models, got %d", len(updated.EvaluationArenaModels))
	}
}
