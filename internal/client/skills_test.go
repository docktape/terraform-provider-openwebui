package client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newSkillTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	c, err := NewClient(server.URL, "test-token", false)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func TestCreateSkillSendsLiteralWireKeys(t *testing.T) {
	var sent map[string]any
	var gotMethod, gotPath string
	c := newSkillTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &sent)
		_, _ = w.Write([]byte(`{"id":"s1","user_id":"u1","name":"Skill One","description":"d","meta":{"tags":["x"]},"is_active":true,"access_grants":[{"id":"a1","principal_type":"group","principal_id":"g1","permission":"read"}],"updated_at":1,"created_at":2}`))
	})

	description := "d"
	form := SkillForm{
		ID:          "s1",
		Name:        "Skill One",
		Description: &description,
		Content:     "instructions",
		Meta:        SkillMeta{Tags: []string{"x"}},
		IsActive:    true,
		AccessControl: map[string]any{
			"read":  map[string]any{"group_ids": []string{"g1"}, "user_ids": []string{}},
			"write": map[string]any{"group_ids": []string{}, "user_ids": []string{}},
		},
	}
	created, err := c.CreateSkill(context.Background(), form)
	if err != nil {
		t.Fatalf("CreateSkill: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/api/v1/skills/create" {
		t.Fatalf("unexpected request %s %s", gotMethod, gotPath)
	}
	if created.ID != "s1" || created.Name != "Skill One" || created.Description == nil || *created.Description != "d" || !created.IsActive {
		t.Fatalf("unexpected response: %+v", created)
	}
	if len(created.Meta.Tags) != 1 || created.Meta.Tags[0] != "x" {
		t.Fatalf("unexpected tags: %+v", created.Meta.Tags)
	}
	if created.AccessControl == nil {
		t.Fatalf("expected AccessControl populated from access_grants")
	}

	for key, want := range map[string]any{"id": "s1", "name": "Skill One", "description": "d", "content": "instructions", "is_active": true} {
		if sent[key] != want {
			t.Fatalf("expected %s=%v on the wire, got %v (body %+v)", key, want, sent[key], sent)
		}
	}
	meta, ok := sent["meta"].(map[string]any)
	if !ok {
		t.Fatalf("expected meta object, got %+v", sent["meta"])
	}
	tags, ok := meta["tags"].([]any)
	if !ok || len(tags) != 1 || tags[0] != "x" {
		t.Fatalf("expected meta.tags=[x], got %+v", meta["tags"])
	}
	grants, ok := sent["access_grants"].([]any)
	if !ok || len(grants) != 1 {
		t.Fatalf("expected one access grant, got %+v", sent["access_grants"])
	}
	grant, ok := grants[0].(map[string]any)
	if !ok || grant["principal_type"] != "group" || grant["principal_id"] != "g1" || grant["permission"] != "read" {
		t.Fatalf("expected grant {principal_type:group principal_id:g1 permission:read}, got %+v", grants[0])
	}
	if len(grant) != 3 {
		t.Fatalf("unexpected extra keys in grant: %+v", grant)
	}
	if _, present := sent["access_control"]; present {
		t.Fatalf("access_control must not be sent: %+v", sent)
	}
}

func TestSkillFormSendsNullDescriptionAndEmptyTagsList(t *testing.T) {
	var sent map[string]any
	c := newSkillTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &sent)
		_, _ = w.Write([]byte(`{"id":"s1","user_id":"u1","name":"N","description":null,"content":"c","meta":{"tags":[]},"is_active":false,"access_grants":[],"updated_at":1,"created_at":2}`))
	})

	// A nil description must be sent as an explicit null (so an update clears it)
	// and nil tags must be sent as [] rather than null; is_active=false must not
	// be dropped by omitempty.
	_, err := c.UpdateSkill(context.Background(), "s1", SkillForm{ID: "s1", Name: "N", Content: "c"})
	if err != nil {
		t.Fatalf("UpdateSkill: %v", err)
	}
	if v, present := sent["description"]; !present || v != nil {
		t.Fatalf("expected description:null on the wire, got present=%v value=%v", present, v)
	}
	if v, present := sent["is_active"]; !present || v != false {
		t.Fatalf("expected is_active:false on the wire, got present=%v value=%v", present, v)
	}
	meta, ok := sent["meta"].(map[string]any)
	if !ok {
		t.Fatalf("meta must be an object (null is rejected with 422), got %+v", sent["meta"])
	}
	tags, ok := meta["tags"].([]any)
	if !ok || len(tags) != 0 {
		t.Fatalf("expected meta.tags=[] (not null), got %+v", meta["tags"])
	}
	grants, ok := sent["access_grants"].([]any)
	if !ok || len(grants) != 0 {
		t.Fatalf("expected access_grants=[] so existing grants are cleared, got %+v", sent["access_grants"])
	}
}

func TestUpdateSkillReturnsContent(t *testing.T) {
	var gotMethod, gotPath string
	c := newSkillTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_, _ = w.Write([]byte(`{"id":"s1","user_id":"u1","name":"N","description":"d","content":"new body","meta":{"tags":null},"is_active":true,"access_grants":[],"updated_at":5,"created_at":2}`))
	})
	out, err := c.UpdateSkill(context.Background(), "s1", SkillForm{ID: "s1", Name: "N", Content: "new body"})
	if err != nil {
		t.Fatalf("UpdateSkill: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/api/v1/skills/id/s1/update" {
		t.Fatalf("unexpected request %s %s", gotMethod, gotPath)
	}
	if out.Content != "new body" || out.UpdatedAt != 5 {
		t.Fatalf("unexpected response: %+v", out)
	}
	if out.Meta.Tags != nil {
		t.Fatalf("a null meta.tags from the server must decode to nil, got %+v", out.Meta.Tags)
	}
}

func TestGetSkillParsesAccessResponse(t *testing.T) {
	var gotMethod, gotPath string
	c := newSkillTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_, _ = w.Write([]byte(`{"id":"s1","user_id":"u1","name":"N","description":null,"meta":{"tags":["a","b"]},"is_active":false,"access_grants":[{"id":"a1","principal_type":"group","principal_id":"g7","permission":"write"}],"updated_at":1,"created_at":2,"user":null,"write_access":true,"content":"body"}`))
	})
	out, err := c.GetSkill(context.Background(), "s1")
	if err != nil {
		t.Fatalf("GetSkill: %v", err)
	}
	if gotMethod != http.MethodGet || gotPath != "/api/v1/skills/id/s1" {
		t.Fatalf("unexpected request %s %s", gotMethod, gotPath)
	}
	if out.Description != nil {
		t.Fatalf("expected nil description for JSON null, got %q", *out.Description)
	}
	if out.IsActive {
		t.Fatalf("expected is_active=false")
	}
	if len(out.Meta.Tags) != 2 || out.Meta.Tags[0] != "a" || out.Meta.Tags[1] != "b" {
		t.Fatalf("unexpected tags: %+v", out.Meta.Tags)
	}
	if out.WriteAccess == nil || !*out.WriteAccess {
		t.Fatalf("expected write_access=true")
	}
	if out.Content == nil || *out.Content != "body" {
		t.Fatalf("expected content to be captured when the server includes it, got %v", out.Content)
	}
	write, ok := out.AccessControl["write"].(map[string]any)
	if !ok {
		t.Fatalf("expected write section, got %+v", out.AccessControl)
	}
	if ids, _ := write["group_ids"].([]string); len(ids) != 1 || ids[0] != "g7" {
		t.Fatalf("expected write.group_ids=[g7], got %+v", write["group_ids"])
	}
}

func TestGetSkillWithoutContentLeavesItNil(t *testing.T) {
	c := newSkillTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":"s1","user_id":"u1","name":"N","description":"","meta":{"tags":[]},"is_active":true,"access_grants":[],"updated_at":1,"created_at":2,"user":null,"write_access":false}`))
	})
	out, err := c.GetSkill(context.Background(), "s1")
	if err != nil {
		t.Fatalf("GetSkill: %v", err)
	}
	if out.Content != nil {
		t.Fatalf("expected nil content when the server omits it, got %q", *out.Content)
	}
	if out.Description == nil || *out.Description != "" {
		t.Fatalf("expected empty-string description to be preserved, got %v", out.Description)
	}
}

func TestGetSkillNotFound(t *testing.T) {
	// Open WebUI signals a missing skill with HTTP 404 (NOT_FOUND detail), unlike
	// groups which answer 401. 401 on this route means "no read access".
	c := newSkillTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"detail":"We could not find what you're looking for :/"}`))
	})
	_, err := c.GetSkill(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGetSkillUnauthorizedIsNotNotFound(t *testing.T) {
	c := newSkillTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"detail":"Access prohibited"}`))
	})
	_, err := c.GetSkill(context.Background(), "s1")
	if err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("expected a hard error for 401, got %v", err)
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusUnauthorized {
		t.Fatalf("expected APIError 401, got %v", err)
	}
}

func TestDeleteSkill(t *testing.T) {
	var gotMethod, gotPath string
	c := newSkillTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_, _ = w.Write([]byte(`true`))
	})
	if err := c.DeleteSkill(context.Background(), "s1"); err != nil {
		t.Fatalf("DeleteSkill: %v", err)
	}
	if gotMethod != http.MethodDelete || gotPath != "/api/v1/skills/id/s1/delete" {
		t.Fatalf("unexpected request %s %s", gotMethod, gotPath)
	}
}

func TestDeleteSkillFalseBodyIsError(t *testing.T) {
	for body, wantErr := range map[string]bool{"true": false, "false": true} {
		c := newSkillTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(body))
		})
		err := c.DeleteSkill(context.Background(), "s1")
		if wantErr && (err == nil || errors.Is(err, ErrNotFound)) {
			t.Fatalf("body %s: expected a delete-failed error, got %v", body, err)
		}
		if !wantErr && err != nil {
			t.Fatalf("body %s: unexpected error %v", body, err)
		}
	}

	// 404 is still ErrNotFound so the resource can treat it as already deleted.
	c := newSkillTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	if err := c.DeleteSkill(context.Background(), "s1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestListSkills(t *testing.T) {
	var gotPath string
	c := newSkillTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`[{"id":"s1","user_id":"u1","name":"N","description":null,"meta":{"tags":null},"is_active":true,"access_grants":[],"updated_at":1,"created_at":2,"user":{"id":"u1","name":"admin","role":"admin","email":"a@a.a"}}]`))
	})
	out, err := c.ListSkills(context.Background())
	if err != nil {
		t.Fatalf("ListSkills: %v", err)
	}
	if gotPath != "/api/v1/skills/" {
		t.Fatalf("unexpected path %s", gotPath)
	}
	if len(out) != 1 || out[0].ID != "s1" || out[0].User == nil || out[0].User.Email != "a@a.a" {
		t.Fatalf("unexpected response: %+v", out)
	}
}

func TestListSkillAccessIsPaginatedObject(t *testing.T) {
	var gotPath, gotPage string
	c := newSkillTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotPage = r.URL.Path, r.URL.Query().Get("page")
		_, _ = w.Write([]byte(`{"items":[{"id":"s1","user_id":"u1","name":"N","description":null,"meta":{"tags":[]},"is_active":true,"access_grants":[],"updated_at":1,"created_at":2,"write_access":true}],"total":31}`))
	})
	out, err := c.ListSkillAccess(context.Background(), 2)
	if err != nil {
		t.Fatalf("ListSkillAccess: %v", err)
	}
	if gotPath != "/api/v1/skills/list" || gotPage != "2" {
		t.Fatalf("unexpected request %s page=%q", gotPath, gotPage)
	}
	if out.Total != 31 || len(out.Items) != 1 || out.Items[0].ID != "s1" {
		t.Fatalf("unexpected response: %+v", out)
	}
}

func TestExportSkillsReturnsContent(t *testing.T) {
	var gotPath string
	c := newSkillTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`[{"id":"s1","user_id":"u1","name":"N","description":null,"content":"body","meta":{"tags":["t"]},"is_active":true,"access_grants":[],"updated_at":1,"created_at":2}]`))
	})
	out, err := c.ExportSkills(context.Background())
	if err != nil {
		t.Fatalf("ExportSkills: %v", err)
	}
	if gotPath != "/api/v1/skills/export" {
		t.Fatalf("unexpected path %s", gotPath)
	}
	if len(out) != 1 || out[0].Content != "body" || len(out[0].Meta.Tags) != 1 {
		t.Fatalf("unexpected response: %+v", out)
	}
}
