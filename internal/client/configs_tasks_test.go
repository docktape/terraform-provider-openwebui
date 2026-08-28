package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetSetTasksConfigRoundTrip(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.Method {
		case http.MethodGet:
			if r.URL.Path != "/api/v1/tasks/config" {
				t.Errorf("expected GET /api/v1/tasks/config, got %s", r.URL.Path)
			}
		case http.MethodPost:
			if r.URL.Path != "/api/v1/tasks/config/update" {
				t.Errorf("expected POST /api/v1/tasks/config/update, got %s", r.URL.Path)
			}

			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("failed to decode request body: %v", err)
			}
			// AUTOCOMPLETE_GENERATION_PROMPT_TEMPLATE has no Python default at
			// v0.11.0 (it's a required field on the backend's TaskConfigForm), so
			// the wire payload must always include the key - omitting it (e.g. via
			// a JSON `omitempty` tag) causes a 422 on a real server.
			if _, ok := body["AUTOCOMPLETE_GENERATION_PROMPT_TEMPLATE"]; !ok {
				t.Errorf("expected request body to contain key %q, got %v", "AUTOCOMPLETE_GENERATION_PROMPT_TEMPLATE", body)
			}
		default:
			t.Errorf("expected GET or POST, got %s", r.Method)
		}

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
	if got.AutocompleteGenerationPromptTemplate != "a" {
		t.Fatalf("expected AutocompleteGenerationPromptTemplate=a, got %q", got.AutocompleteGenerationPromptTemplate)
	}

	updated, err := c.SetTasksConfig(context.Background(), *got)
	if err != nil {
		t.Fatalf("SetTasksConfig: %v", err)
	}
	if updated.TaskModel == nil || *updated.TaskModel != "gpt-4o" {
		t.Fatalf("expected round-tripped TaskModel=gpt-4o, got %v", updated.TaskModel)
	}
}

// TestTaskConfigFormAutocompleteGenerationPromptTemplateZeroValueSentAsEmptyString
// guards the exact bug this fix addresses: AUTOCOMPLETE_GENERATION_PROMPT_TEMPLATE
// has no Python default at v0.11.0 and is declared as a plain, non-Optional `str`
// on the backend's TaskConfigForm (backend/open_webui/routers/tasks.py), so it
// rejects both a missing key and an explicit `null` with a 422. A Terraform user
// who never sets autocomplete_generation_prompt_template must still produce a
// request that sends the key as a genuine, valid (if empty) string - never
// omitted and never null. This exercises the "field genuinely unset by the user"
// path directly via json.Marshal on a zero-value form, rather than round-tripping
// a value that was already decoded from a populated response (which would not
// catch a regression back to a *string/omitempty representation).
func TestTaskConfigFormAutocompleteGenerationPromptTemplateZeroValueSentAsEmptyString(t *testing.T) {
	form := TaskConfigForm{}

	encoded, err := json.Marshal(form)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if !strings.Contains(string(encoded), `"AUTOCOMPLETE_GENERATION_PROMPT_TEMPLATE":""`) {
		t.Fatalf("expected encoded JSON to contain \"AUTOCOMPLETE_GENERATION_PROMPT_TEMPLATE\":\"\", got %s", encoded)
	}
}
