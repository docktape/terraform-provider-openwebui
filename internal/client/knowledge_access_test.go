package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newKnowledgeTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	c, err := NewClient(server.URL, "test-token", false)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func TestCreateKnowledgeSendsAccessGrants(t *testing.T) {
	var sent map[string]any
	c := newKnowledgeTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &sent)
		_, _ = w.Write([]byte(`{"id":"k1","user_id":"u1","name":"K","description":"d","created_at":1,"updated_at":2,"access_grants":[{"id":"a1","principal_type":"group","principal_id":"g1","permission":"read"}]}`))
	})
	form := KnowledgeForm{
		Name: "K", Description: "d",
		AccessControl: map[string]any{
			"read":  map[string]any{"group_ids": []string{"g1"}, "user_ids": []string{}},
			"write": map[string]any{"group_ids": []string{}, "user_ids": []string{}},
		},
	}
	out, err := c.CreateKnowledge(context.Background(), form)
	if err != nil {
		t.Fatalf("CreateKnowledge: %v", err)
	}
	if _, ok := sent["access_control"]; ok {
		t.Fatalf("access_control must not be sent: %+v", sent)
	}
	if _, ok := sent["access_grants"].([]any); !ok {
		t.Fatalf("expected access_grants list, got %+v", sent["access_grants"])
	}
	read := out.AccessControl["read"].(map[string]any)
	ids := read["group_ids"].([]string)
	if len(ids) != 1 || ids[0] != "g1" {
		t.Fatalf("expected read.group_ids=[g1], got %+v", out.AccessControl)
	}
}

func TestGetKnowledgeParsesAccessGrants(t *testing.T) {
	c := newKnowledgeTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":"k1","user_id":"u1","name":"K","description":"d","created_at":1,"updated_at":2,"files":[],"access_grants":[{"id":"a1","principal_type":"group","principal_id":"g3","permission":"write"}]}`))
	})
	out, err := c.GetKnowledge(context.Background(), "k1")
	if err != nil {
		t.Fatalf("GetKnowledge: %v", err)
	}
	write := out.AccessControl["write"].(map[string]any)
	ids := write["group_ids"].([]string)
	if len(ids) != 1 || ids[0] != "g3" {
		t.Fatalf("expected write.group_ids=[g3], got %+v", out.AccessControl)
	}
}
