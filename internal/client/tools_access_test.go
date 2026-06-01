package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newToolTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	c, err := NewClient(server.URL, "test-token", false)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func TestCreateToolSendsAccessGrants(t *testing.T) {
	var sent map[string]any
	c := newToolTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &sent)
		_, _ = w.Write([]byte(`{"id":"t1","user_id":"u1","name":"T","meta":{},"created_at":1,"updated_at":2,"access_grants":[{"id":"a1","principal_type":"group","principal_id":"g1","permission":"write"}]}`))
	})
	form := ToolForm{
		ID: "t1", Name: "T", Content: "x",
		AccessControl: map[string]any{
			"read":  map[string]any{"group_ids": []string{"g1"}, "user_ids": []string{}},
			"write": map[string]any{"group_ids": []string{"g1"}, "user_ids": []string{}},
		},
	}
	out, err := c.CreateTool(context.Background(), form)
	if err != nil {
		t.Fatalf("CreateTool: %v", err)
	}
	if _, ok := sent["access_control"]; ok {
		t.Fatalf("access_control must not be sent: %+v", sent)
	}
	if _, ok := sent["access_grants"].([]any); !ok {
		t.Fatalf("expected access_grants list, got %+v", sent["access_grants"])
	}
	if out.AccessControl == nil {
		t.Fatalf("expected AccessControl populated from grants")
	}
}

func TestGetToolParsesAccessGrants(t *testing.T) {
	c := newToolTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":"t1","user_id":"u1","name":"T","meta":{},"created_at":1,"updated_at":2,"write_access":true,"access_grants":[{"id":"a1","principal_type":"group","principal_id":"g7","permission":"read"}]}`))
	})
	out, err := c.GetTool(context.Background(), "t1")
	if err != nil {
		t.Fatalf("GetTool: %v", err)
	}
	read := out.AccessControl["read"].(map[string]any)
	ids := read["group_ids"].([]string)
	if len(ids) != 1 || ids[0] != "g7" {
		t.Fatalf("expected read.group_ids=[g7], got %+v", out.AccessControl)
	}
}
