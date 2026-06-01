package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newGroupTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	c, err := NewClient(server.URL, "test-token", false)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func TestGetGroupUsesExportAndReadsUserIDs(t *testing.T) {
	var gotPath string
	c := newGroupTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"id":"grp1","user_id":"owner","name":"G","description":"d","created_at":1,"updated_at":2,"member_count":2,"user_ids":["u1","u2"]}`))
	})
	out, err := c.GetGroup(context.Background(), "grp1")
	if err != nil {
		t.Fatalf("GetGroup: %v", err)
	}
	if gotPath != "/groups/id/grp1/export" {
		t.Fatalf("expected export path, got %q", gotPath)
	}
	if len(out.UserIDs) != 2 || out.UserIDs[0] != "u1" || out.UserIDs[1] != "u2" {
		t.Fatalf("expected user_ids=[u1 u2], got %+v", out.UserIDs)
	}
}

func TestGetGroupTreatsUnauthorizedNotFoundAsErrNotFound(t *testing.T) {
	c := newGroupTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"detail":"We could not find what you're looking for :/"}`))
	})
	_, err := c.GetGroup(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGetGroupKeepsGenuineUnauthorizedAsError(t *testing.T) {
	c := newGroupTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"detail":"401 Unauthorized"}`))
	})
	_, err := c.GetGroup(context.Background(), "forbidden")
	if err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("expected a non-ErrNotFound error, got %v", err)
	}
}
