package client

import (
	"context"
	"errors"
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
	users, err := c.SearchUsers(context.Background(), "alice")
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

func TestGetUserNotFoundMapsFrom400(t *testing.T) {
	c := newUsersTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"detail":"We could not find what you're looking for :/"}`))
	})
	_, err := c.GetUser(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGetUserBadRequestStaysError(t *testing.T) {
	c := newUsersTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"detail":"Bad request"}`))
	})
	_, err := c.GetUser(context.Background(), "bad")
	if err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("expected a non-ErrNotFound error, got %v", err)
	}
}

func TestGetUserNotFoundDetailOnOtherStatusStaysError(t *testing.T) {
	c := newUsersTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"detail":"We could not find what you're looking for :/"}`))
	})
	_, err := c.GetUser(context.Background(), "odd")
	if err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("expected a non-ErrNotFound error, got %v", err)
	}
}
