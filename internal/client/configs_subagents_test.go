package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newSubagentsTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	c, err := NewClient(server.URL, "test-token", false)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func TestGetSubagentsConfig(t *testing.T) {
	client := newSubagentsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/configs/subagents" {
			t.Errorf("expected /api/v1/configs/subagents, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{
			"ENABLE_SUBAGENTS": true,
			"SUBAGENTS_BACKGROUND_ENABLED": false,
			"SUBAGENTS_MAX_CONCURRENT": 5,
			"SUBAGENTS_MAX_ASYNC": 3,
			"SUBAGENTS_MAX_ITERATIONS": 10,
			"SUBAGENTS_MAX_OUTPUT": 2048,
			"SUBAGENTS_SYSTEM_PROMPT": "You are a helpful assistant."
		}`)); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	})

	config, err := client.GetSubagentsConfig(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !config.EnableSubagents {
		t.Errorf("expected EnableSubagents=true, got false")
	}
	if config.SubagentsBackgroundEnabled {
		t.Errorf("expected SubagentsBackgroundEnabled=false, got true")
	}
	if config.SubagentsMaxConcurrent != 5 {
		t.Errorf("expected SubagentsMaxConcurrent=5, got %d", config.SubagentsMaxConcurrent)
	}
	if config.SubagentsMaxAsync != 3 {
		t.Errorf("expected SubagentsMaxAsync=3, got %d", config.SubagentsMaxAsync)
	}
	if config.SubagentsMaxIterations != 10 {
		t.Errorf("expected SubagentsMaxIterations=10, got %d", config.SubagentsMaxIterations)
	}
	if config.SubagentsMaxOutput != 2048 {
		t.Errorf("expected SubagentsMaxOutput=2048, got %d", config.SubagentsMaxOutput)
	}
	if config.SubagentsSystemPrompt != "You are a helpful assistant." {
		t.Errorf("expected SubagentsSystemPrompt='You are a helpful assistant.', got %q", config.SubagentsSystemPrompt)
	}
}

func TestSetSubagentsConfig(t *testing.T) {
	client := newSubagentsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/configs/subagents" {
			t.Errorf("expected /api/v1/configs/subagents, got %s", r.URL.Path)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		for _, key := range []string{
			"ENABLE_SUBAGENTS",
			"SUBAGENTS_BACKGROUND_ENABLED",
			"SUBAGENTS_MAX_CONCURRENT",
			"SUBAGENTS_MAX_ASYNC",
			"SUBAGENTS_MAX_ITERATIONS",
			"SUBAGENTS_MAX_OUTPUT",
			"SUBAGENTS_SYSTEM_PROMPT",
		} {
			if _, ok := body[key]; !ok {
				t.Errorf("expected request body to contain key %q, got %v", key, body)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{
			"ENABLE_SUBAGENTS": true,
			"SUBAGENTS_BACKGROUND_ENABLED": true,
			"SUBAGENTS_MAX_CONCURRENT": 8,
			"SUBAGENTS_MAX_ASYNC": 4,
			"SUBAGENTS_MAX_ITERATIONS": 15,
			"SUBAGENTS_MAX_OUTPUT": 4096,
			"SUBAGENTS_SYSTEM_PROMPT": "Updated prompt"
		}`)); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	})

	form := SubagentsConfigForm{
		EnableSubagents:            true,
		SubagentsBackgroundEnabled: true,
		SubagentsMaxConcurrent:     8,
		SubagentsMaxAsync:          4,
		SubagentsMaxIterations:     15,
		SubagentsMaxOutput:         4096,
		SubagentsSystemPrompt:      "Updated prompt",
	}

	config, err := client.SetSubagentsConfig(context.Background(), form)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !config.EnableSubagents {
		t.Errorf("expected EnableSubagents=true, got false")
	}
	if !config.SubagentsBackgroundEnabled {
		t.Errorf("expected SubagentsBackgroundEnabled=true, got false")
	}
	if config.SubagentsMaxConcurrent != 8 {
		t.Errorf("expected SubagentsMaxConcurrent=8, got %d", config.SubagentsMaxConcurrent)
	}
	if config.SubagentsMaxAsync != 4 {
		t.Errorf("expected SubagentsMaxAsync=4, got %d", config.SubagentsMaxAsync)
	}
	if config.SubagentsMaxIterations != 15 {
		t.Errorf("expected SubagentsMaxIterations=15, got %d", config.SubagentsMaxIterations)
	}
	if config.SubagentsMaxOutput != 4096 {
		t.Errorf("expected SubagentsMaxOutput=4096, got %d", config.SubagentsMaxOutput)
	}
	if config.SubagentsSystemPrompt != "Updated prompt" {
		t.Errorf("expected SubagentsSystemPrompt='Updated prompt', got %q", config.SubagentsSystemPrompt)
	}
}
