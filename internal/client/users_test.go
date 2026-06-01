package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newUsersTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	c, err := NewClient(server.URL, "test-token", false)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func TestSearchUsersOmitsLimit(t *testing.T) {
	var gotQuery, gotLimit string
	c := newUsersTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query().Get("query")
		gotLimit = r.URL.Query().Get("limit")
		_, _ = w.Write([]byte(`{"users":[{"id":"u1","name":"Alice","email":"alice@example.com","role":"user","last_active_at":1,"updated_at":2,"created_at":3}],"total":1}`))
	})
	users, err := c.SearchUsers(context.Background(), "alice", 50)
	if err != nil {
		t.Fatalf("SearchUsers: %v", err)
	}
	if gotQuery != "alice" {
		t.Fatalf("expected query=alice, got %q", gotQuery)
	}
	if gotLimit != "" {
		t.Fatalf("expected no limit param, got %q", gotLimit)
	}
	if len(users) != 1 || users[0].ID != "u1" {
		t.Fatalf("unexpected users: %+v", users)
	}
}
