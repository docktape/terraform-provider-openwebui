package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListKnowledgeFilesSendsLimitParam(t *testing.T) {
	var gotLimit, gotPage string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotLimit = r.URL.Query().Get("limit")
		gotPage = r.URL.Query().Get("page")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[{"id":"f1","user_id":"u","filename":"a.txt","meta":{},"created_at":1,"updated_at":2}],"total":1}`))
	}))
	t.Cleanup(server.Close)

	c, err := NewClient(server.URL, "test-token", false)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	resp, err := c.ListKnowledgeFiles(context.Background(), "kb1", "", "", "", "", 1, KnowledgeFilesLookupLimit)
	if err != nil {
		t.Fatalf("ListKnowledgeFiles: %v", err)
	}
	if gotLimit != "10000" {
		t.Fatalf("expected limit=10000, got %q", gotLimit)
	}
	if gotPage != "1" {
		t.Fatalf("expected page=1, got %q", gotPage)
	}
	if len(resp.Items) != 1 || resp.Items[0].ID != "f1" {
		t.Fatalf("unexpected items: %+v", resp.Items)
	}
}

func TestListKnowledgeFilesOmitsLimitWhenZero(t *testing.T) {
	var sawLimit bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawLimit = r.URL.Query().Has("limit")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[],"total":0}`))
	}))
	t.Cleanup(server.Close)

	c, err := NewClient(server.URL, "test-token", false)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if _, err := c.ListKnowledgeFiles(context.Background(), "kb1", "", "", "", "", 1, 0); err != nil {
		t.Fatalf("ListKnowledgeFiles: %v", err)
	}
	if sawLimit {
		t.Fatal("expected no limit param when limit=0")
	}
}
