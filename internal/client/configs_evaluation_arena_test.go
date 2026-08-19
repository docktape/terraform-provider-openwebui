package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

// TestArenaModelMetaModelIDsJSONEncoding guards the exact bug this fix
// addresses: with `omitempty` on ArenaModelMeta.ModelIDs, encoding/json
// cannot distinguish a nil slice from a non-nil, zero-length slice - both
// are dropped from the marshaled payload, so a backend echoing that
// payload back looks identical whether the config said `model_ids = []` or
// omitted model_ids entirely. Without `omitempty`, nil must marshal to
// `"model_ids":null` and a non-nil empty slice must marshal to
// `"model_ids":[]`, and unmarshaling each must reproduce nil/non-nil-empty
// respectively so flattenArenaModels can tell them apart.
func TestArenaModelMetaModelIDsJSONEncoding(t *testing.T) {
	t.Run("nil slice marshals to null and round-trips to nil", func(t *testing.T) {
		meta := ArenaModelMeta{ModelIDs: nil}

		encoded, err := json.Marshal(meta)
		if err != nil {
			t.Fatalf("json.Marshal: %v", err)
		}
		if !strings.Contains(string(encoded), `"model_ids":null`) {
			t.Fatalf("expected encoded JSON to contain \"model_ids\":null, got %s", encoded)
		}

		var decoded ArenaModelMeta
		if err := json.Unmarshal(encoded, &decoded); err != nil {
			t.Fatalf("json.Unmarshal: %v", err)
		}
		if decoded.ModelIDs != nil {
			t.Fatalf("expected decoded ModelIDs to be nil, got %#v", decoded.ModelIDs)
		}
	})

	t.Run("non-nil empty slice marshals to empty array and round-trips to non-nil empty", func(t *testing.T) {
		meta := ArenaModelMeta{ModelIDs: []string{}}

		encoded, err := json.Marshal(meta)
		if err != nil {
			t.Fatalf("json.Marshal: %v", err)
		}
		if !strings.Contains(string(encoded), `"model_ids":[]`) {
			t.Fatalf("expected encoded JSON to contain \"model_ids\":[], got %s", encoded)
		}

		var decoded ArenaModelMeta
		if err := json.Unmarshal(encoded, &decoded); err != nil {
			t.Fatalf("json.Unmarshal: %v", err)
		}
		if decoded.ModelIDs == nil {
			t.Fatal("expected decoded ModelIDs to be a non-nil empty slice, got nil")
		}
		if len(decoded.ModelIDs) != 0 {
			t.Fatalf("expected decoded ModelIDs to be empty, got %#v", decoded.ModelIDs)
		}
	})
}
