package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newPromptTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	c, err := NewClient(server.URL, "test-token", false)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func TestCreatePromptSendsNameAndGrants(t *testing.T) {
	var sent map[string]any
	var gotPath string
	c := newPromptTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &sent)
		_, _ = w.Write([]byte(`{"id":"p1","command":"/greet","user_id":"u1","name":"Greet","content":"hi","created_at":1,"updated_at":2,"access_grants":[{"id":"a1","principal_type":"group","principal_id":"g1","permission":"read"}]}`))
	})

	form := PromptForm{
		Command: "/greet", Name: "Greet", Content: "hi",
		AccessControl: map[string]any{
			"read":  map[string]any{"group_ids": []string{"g1"}, "user_ids": []string{}},
			"write": map[string]any{"group_ids": []string{}, "user_ids": []string{}},
		},
	}
	out, err := c.CreatePrompt(context.Background(), form)
	if err != nil {
		t.Fatalf("CreatePrompt: %v", err)
	}
	if gotPath != "/api/v1/prompts/create" {
		t.Fatalf("expected /api/v1/prompts/create, got %q", gotPath)
	}
	if _, ok := sent["access_control"]; ok {
		t.Fatalf("access_control must not be sent: %+v", sent)
	}
	if sent["name"] != "Greet" {
		t.Fatalf("expected name=Greet, got %+v", sent["name"])
	}
	if _, ok := sent["title"]; ok {
		t.Fatalf("title must not be sent: %+v", sent)
	}
	if _, ok := sent["access_grants"].([]any); !ok {
		t.Fatalf("expected access_grants list, got %+v", sent["access_grants"])
	}
	if out.ID != "p1" {
		t.Fatalf("expected id=p1, got %q", out.ID)
	}
	read, ok := out.AccessControl["read"].(map[string]any)
	if !ok {
		t.Fatalf("expected read to be map[string]any, got %T", out.AccessControl["read"])
	}
	ids, ok := read["group_ids"].([]string)
	if !ok {
		t.Fatalf("expected group_ids to be []string, got %T", read["group_ids"])
	}
	if len(ids) != 1 || ids[0] != "g1" {
		t.Fatalf("expected read.group_ids=[g1], got %+v", out.AccessControl)
	}
}

func TestGetPromptUsesIDRoute(t *testing.T) {
	var gotPath string
	c := newPromptTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"id":"p1","command":"/greet","user_id":"u1","name":"Greet","content":"hi","created_at":10,"updated_at":20,"access_grants":[]}`))
	})
	out, err := c.GetPrompt(context.Background(), "p1")
	if err != nil {
		t.Fatalf("GetPrompt: %v", err)
	}
	if gotPath != "/api/v1/prompts/id/p1" {
		t.Fatalf("expected /api/v1/prompts/id/p1, got %q", gotPath)
	}
	if out.Name != "Greet" || out.UpdatedAt != 20 {
		t.Fatalf("unexpected prompt: %+v", out)
	}
}

func TestUpdatePromptUsesIDRoute(t *testing.T) {
	var gotPath, gotMethod string
	c := newPromptTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		_, _ = w.Write([]byte(`{"id":"p1","command":"/greet","user_id":"u1","name":"Greet2","content":"yo","created_at":10,"updated_at":30,"access_grants":[]}`))
	})
	out, err := c.UpdatePrompt(context.Background(), "p1", PromptForm{Command: "/greet", Name: "Greet2", Content: "yo"})
	if err != nil {
		t.Fatalf("UpdatePrompt: %v", err)
	}
	if gotPath != "/api/v1/prompts/id/p1/update" || gotMethod != http.MethodPost {
		t.Fatalf("expected POST /api/v1/prompts/id/p1/update, got %s %q", gotMethod, gotPath)
	}
	if out.Name != "Greet2" {
		t.Fatalf("expected name=Greet2, got %q", out.Name)
	}
}

func TestDeletePromptUsesIDRoute(t *testing.T) {
	var gotPath, gotMethod string
	c := newPromptTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		_, _ = w.Write([]byte(`true`))
	})
	if err := c.DeletePrompt(context.Background(), "p1"); err != nil {
		t.Fatalf("DeletePrompt: %v", err)
	}
	if gotPath != "/api/v1/prompts/id/p1/delete" || gotMethod != http.MethodDelete {
		t.Fatalf("expected DELETE /api/v1/prompts/id/p1/delete, got %s %q", gotMethod, gotPath)
	}
}

func TestListPromptsHitsRoot(t *testing.T) {
	var gotPath string
	c := newPromptTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`[{"id":"p1","command":"/greet","user_id":"u1","name":"Greet","content":"hi","created_at":1,"updated_at":2,"access_grants":[]}]`))
	})
	prompts, err := c.ListPrompts(context.Background())
	if err != nil {
		t.Fatalf("ListPrompts: %v", err)
	}
	if gotPath != "/api/v1/prompts/" {
		t.Fatalf("expected /api/v1/prompts/, got %q", gotPath)
	}
	if len(prompts) != 1 || prompts[0].Command != "/greet" {
		t.Fatalf("unexpected prompts: %+v", prompts)
	}
}

func TestPromptFormMarshal_WithNewFields(t *testing.T) {
	form := PromptForm{
		Command: "/greet",
		Name:    "Greet",
		Content: "Hello!",
		Tags:    []string{"util", "greeting"},
	}
	data, err := json.Marshal(form)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// Neither PromptForm at v0.11.0 nor at v0.12.0 declares is_active; it can
	// only be changed through POST /prompts/id/{id}/toggle.
	if _, ok := out["is_active"]; ok {
		t.Fatalf("PromptForm must not send is_active, got %v", out["is_active"])
	}
	tags, ok := out["tags"].([]any)
	if !ok || len(tags) != 2 {
		t.Fatalf("expected 2 tags, got %v", out["tags"])
	}
}

func TestPromptWritesSendNoIsActive(t *testing.T) {
	var bodies []map[string]any
	c := newPromptTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var sent map[string]any
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &sent)
		bodies = append(bodies, sent)
		_, _ = w.Write([]byte(`{"id":"p1","command":"/greet","user_id":"u1","name":"Greet","content":"hi","is_active":true,"created_at":1,"updated_at":2,"access_grants":[]}`))
	})
	form := PromptForm{Command: "/greet", Name: "Greet", Content: "hi"}
	if _, err := c.CreatePrompt(context.Background(), form); err != nil {
		t.Fatalf("CreatePrompt: %v", err)
	}
	if _, err := c.UpdatePrompt(context.Background(), "p1", form); err != nil {
		t.Fatalf("UpdatePrompt: %v", err)
	}
	if len(bodies) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(bodies))
	}
	for i, b := range bodies {
		if _, ok := b["is_active"]; ok {
			t.Fatalf("request %d sent is_active: %v", i, b)
		}
	}
}

func TestTogglePromptPath(t *testing.T) {
	var gotPath, gotMethod string
	c := newPromptTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		_, _ = w.Write([]byte(`{"id":"p1","command":"/greet","user_id":"u1","name":"Greet","content":"hi","is_active":false,"created_at":10,"updated_at":40,"access_grants":[]}`))
	})
	out, err := c.TogglePrompt(context.Background(), "p1")
	if err != nil {
		t.Fatalf("TogglePrompt: %v", err)
	}
	if gotPath != "/api/v1/prompts/id/p1/toggle" || gotMethod != http.MethodPost {
		t.Fatalf("expected POST /api/v1/prompts/id/p1/toggle, got %s %q", gotMethod, gotPath)
	}
	if out.ID != "p1" || out.IsActive == nil || *out.IsActive {
		t.Fatalf("expected decoded prompt p1 with is_active=false, got %+v", out)
	}
}
