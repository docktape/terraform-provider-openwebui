package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetSetTasksConfigRoundTrip(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"TASK_MODEL":"gpt-4o","TASK_MODEL_EXTERNAL":null,"ENABLE_TITLE_GENERATION":true,"TITLE_GENERATION_PROMPT_TEMPLATE":"t","IMAGE_PROMPT_GENERATION_PROMPT_TEMPLATE":"i","ENABLE_AUTOCOMPLETE_GENERATION":true,"AUTOCOMPLETE_GENERATION_INPUT_MAX_LENGTH":100,"AUTOCOMPLETE_GENERATION_PROMPT_TEMPLATE":"a","TAGS_GENERATION_PROMPT_TEMPLATE":"tg","FOLLOW_UP_GENERATION_PROMPT_TEMPLATE":"f","ENABLE_FOLLOW_UP_GENERATION":true,"ENABLE_TAGS_GENERATION":true,"ENABLE_SEARCH_QUERY_GENERATION":true,"ENABLE_RETRIEVAL_QUERY_GENERATION":true,"QUERY_GENERATION_PROMPT_TEMPLATE":"q","TOOLS_FUNCTION_CALLING_PROMPT_TEMPLATE":"tf","ENABLE_VOICE_MODE_PROMPT":true,"VOICE_MODE_PROMPT_TEMPLATE":"v"}`))
	}))
	t.Cleanup(server.Close)

	c, err := NewClient(server.URL, "test-token", false)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	got, err := c.GetTasksConfig(context.Background())
	if err != nil {
		t.Fatalf("GetTasksConfig: %v", err)
	}
	if got.TaskModel == nil || *got.TaskModel != "gpt-4o" {
		t.Fatalf("expected TaskModel=gpt-4o, got %v", got.TaskModel)
	}
	if !got.EnableTitleGeneration {
		t.Fatal("expected EnableTitleGeneration=true")
	}

	updated, err := c.SetTasksConfig(context.Background(), *got)
	if err != nil {
		t.Fatalf("SetTasksConfig: %v", err)
	}
	if updated.TaskModel == nil || *updated.TaskModel != "gpt-4o" {
		t.Fatalf("expected round-tripped TaskModel=gpt-4o, got %v", updated.TaskModel)
	}
}
