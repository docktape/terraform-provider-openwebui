package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetSetChatConfigRoundTrip(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/chats/config" {
			t.Errorf("expected path /api/v1/chats/config, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")

		switch r.Method {
		case http.MethodGet:
			// no request body to inspect.
		case http.MethodPost:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("failed to decode request body: %v", err)
			}
			for _, key := range []string{
				"ENABLE_CONTEXT_COMPACTION",
				"CONTEXT_COMPACTION_TOKEN_THRESHOLD",
				"CONTEXT_COMPACTION_TOKEN_CAP",
				"CONTEXT_COMPACTION_RETENTION_PERCENTAGE",
				"CONTEXT_COMPACTION_PROMPT_TEMPLATE",
				"CONTEXT_COMPACTION_MODEL",
			} {
				if _, ok := body[key]; !ok {
					t.Errorf("expected request body to contain key %q, got %v", key, body)
				}
			}
		default:
			t.Errorf("expected GET or POST, got %s", r.Method)
		}

		_, _ = w.Write([]byte(`{"ENABLE_CONTEXT_COMPACTION":true,"CONTEXT_COMPACTION_TOKEN_THRESHOLD":8000,"CONTEXT_COMPACTION_TOKEN_CAP":12000,"CONTEXT_COMPACTION_RETENTION_PERCENTAGE":40,"CONTEXT_COMPACTION_PROMPT_TEMPLATE":"compact prompt","CONTEXT_COMPACTION_MODEL":"gpt-4o-mini"}`))
	}))
	t.Cleanup(server.Close)

	c, err := NewClient(server.URL, "test-token", false)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	got, err := c.GetChatConfig(context.Background())
	if err != nil {
		t.Fatalf("GetChatConfig: %v", err)
	}
	if !got.EnableContextCompaction {
		t.Fatal("expected EnableContextCompaction=true")
	}
	if got.ContextCompactionTokenThreshold != 8000 {
		t.Fatalf("expected ContextCompactionTokenThreshold=8000, got %v", got.ContextCompactionTokenThreshold)
	}
	if got.ContextCompactionTokenCap == nil || *got.ContextCompactionTokenCap != 12000 {
		t.Fatalf("expected ContextCompactionTokenCap=12000, got %v", got.ContextCompactionTokenCap)
	}
	if got.ContextCompactionRetentionPercentage != 40 {
		t.Fatalf("expected ContextCompactionRetentionPercentage=40, got %v", got.ContextCompactionRetentionPercentage)
	}
	if got.ContextCompactionPromptTemplate != "compact prompt" {
		t.Fatalf("expected ContextCompactionPromptTemplate=compact prompt, got %v", got.ContextCompactionPromptTemplate)
	}
	if got.ContextCompactionModel != "gpt-4o-mini" {
		t.Fatalf("expected ContextCompactionModel=gpt-4o-mini, got %v", got.ContextCompactionModel)
	}

	updated, err := c.SetChatConfig(context.Background(), *got)
	if err != nil {
		t.Fatalf("SetChatConfig: %v", err)
	}
	if !updated.EnableContextCompaction {
		t.Fatal("expected round-tripped EnableContextCompaction=true")
	}
	if updated.ContextCompactionTokenThreshold != 8000 {
		t.Fatalf("expected round-tripped ContextCompactionTokenThreshold=8000, got %v", updated.ContextCompactionTokenThreshold)
	}
}
