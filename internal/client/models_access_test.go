package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newModelTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	c, err := NewClient(server.URL, "test-token")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func TestCreateModelSendsAccessGrants(t *testing.T) {
	var sent map[string]any
	c := newModelTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &sent)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"m1","user_id":"u1","name":"M","meta":{},"params":{},"is_active":true,"created_at":1,"updated_at":2,"access_grants":[{"id":"a1","principal_type":"group","principal_id":"g1","permission":"read"}]}`))
	})

	form := ModelForm{
		ID:   "m1",
		Name: "M",
		Meta: map[string]any{}, Params: map[string]any{},
		AccessControl: map[string]any{
			"read":  map[string]any{"group_ids": []string{"g1"}, "user_ids": []string{}},
			"write": map[string]any{"group_ids": []string{}, "user_ids": []string{}},
		},
	}
	out, err := c.CreateModel(context.Background(), form)
	if err != nil {
		t.Fatalf("CreateModel: %v", err)
	}

	if _, ok := sent["access_control"]; ok {
		t.Fatalf("access_control must not be sent: %+v", sent)
	}
	grants, ok := sent["access_grants"].([]any)
	if !ok || len(grants) != 1 {
		t.Fatalf("expected 1 access_grant, got %+v", sent["access_grants"])
	}

	read := out.AccessControl["read"].(map[string]any)
	ids := read["group_ids"].([]string)
	if len(ids) != 1 || ids[0] != "g1" {
		t.Fatalf("expected read.group_ids=[g1], got %+v", out.AccessControl)
	}
}

func TestGetModelParsesAccessGrants(t *testing.T) {
	c := newModelTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":"m1","user_id":"u1","name":"M","meta":{},"params":{},"is_active":true,"created_at":1,"updated_at":2,"access_grants":[{"id":"a1","principal_type":"group","principal_id":"g9","permission":"write"}]}`))
	})
	out, err := c.GetModel(context.Background(), "m1")
	if err != nil {
		t.Fatalf("GetModel: %v", err)
	}
	write := out.AccessControl["write"].(map[string]any)
	ids := write["group_ids"].([]string)
	if len(ids) != 1 || ids[0] != "g9" {
		t.Fatalf("expected write.group_ids=[g9], got %+v", out.AccessControl)
	}
}
