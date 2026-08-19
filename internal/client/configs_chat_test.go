package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetSetChatConfigRoundTrip(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ENABLE_CONTEXT_COMPACTION":true,"CONTEXT_COMPACTION_TOKEN_THRESHOLD":8000,"CONTEXT_COMPACTION_TOKEN_CAP":12000,"CONTEXT_COMPACTION_RETENTION_PERCENTAGE":50.5,"CONTEXT_COMPACTION_PROMPT_TEMPLATE":"compact prompt"}`))
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
	if got.ContextCompactionTokenCap != 12000 {
		t.Fatalf("expected ContextCompactionTokenCap=12000, got %v", got.ContextCompactionTokenCap)
	}
	if got.ContextCompactionRetentionPercentage != 50.5 {
		t.Fatalf("expected ContextCompactionRetentionPercentage=50.5, got %v", got.ContextCompactionRetentionPercentage)
	}
	if got.ContextCompactionPromptTemplate != "compact prompt" {
		t.Fatalf("expected ContextCompactionPromptTemplate=compact prompt, got %v", got.ContextCompactionPromptTemplate)
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
