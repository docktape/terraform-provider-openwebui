package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListFilesPaginatesAndAggregates(t *testing.T) {
	var gotContents []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/files/" {
			t.Fatalf("expected /files/, got %q", r.URL.Path)
		}
		gotContents = append(gotContents, r.URL.Query().Get("content"))
		page := r.URL.Query().Get("page")
		w.Header().Set("Content-Type", "application/json")
		switch page {
		case "", "1":
			items := make([]string, 0, filesListPageSize)
			for i := 0; i < filesListPageSize; i++ {
				items = append(items, fmt.Sprintf(`{"id":"f%d","user_id":"u","filename":"f%d.txt","meta":{},"created_at":1,"updated_at":2}`, i, i))
			}
			_, _ = fmt.Fprintf(w, `{"items":[%s],"total":%d}`, join(items), filesListPageSize+1)
		case "2":
			_, _ = fmt.Fprintf(w, `{"items":[{"id":"flast","user_id":"u","filename":"last.txt","meta":{},"created_at":1,"updated_at":2}],"total":%d}`, filesListPageSize+1)
		default:
			t.Fatalf("unexpected page %q", page)
		}
	}))
	t.Cleanup(server.Close)

	c, err := NewClient(server.URL, "test-token", false)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	files, err := c.ListFiles(context.Background(), false)
	if err != nil {
		t.Fatalf("ListFiles: %v", err)
	}
	if len(files) != filesListPageSize+1 {
		t.Fatalf("expected %d files, got %d", filesListPageSize+1, len(files))
	}
	for i, got := range gotContents {
		if got != "false" {
			t.Fatalf("page %d: expected content=false, got %q", i+1, got)
		}
	}
	if files[len(files)-1].ID != "flast" {
		t.Fatalf("expected last file id=flast, got %q", files[len(files)-1].ID)
	}
}

func TestListFilesPassesContentParam(t *testing.T) {
	var gotContent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContent = r.URL.Query().Get("content")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[],"total":0}`))
	}))
	t.Cleanup(server.Close)

	c, err := NewClient(server.URL, "test-token", false)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if _, err := c.ListFiles(context.Background(), true); err != nil {
		t.Fatalf("ListFiles: %v", err)
	}
	if gotContent != "true" {
		t.Fatalf("expected content=true, got %q", gotContent)
	}
}
