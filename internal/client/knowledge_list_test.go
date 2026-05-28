package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListKnowledgePaginatesAndAggregates(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		page := r.URL.Query().Get("page")
		w.Header().Set("Content-Type", "application/json")
		switch page {
		case "", "1":
			// First page: a full page (knowledgeListPageSize) so the client keeps going.
			items := make([]string, 0, knowledgeListPageSize)
			for i := 0; i < knowledgeListPageSize; i++ {
				items = append(items, fmt.Sprintf(`{"id":"k%d","user_id":"u","name":"K%d","description":"","created_at":1,"updated_at":2,"access_grants":[]}`, i, i))
			}
			_, _ = fmt.Fprintf(w, `{"items":[%s],"total":%d}`, join(items), knowledgeListPageSize+1)
		case "2":
			_, _ = w.Write([]byte(`{"items":[{"id":"klast","user_id":"u","name":"Last","description":"","created_at":1,"updated_at":2,"access_grants":[{"id":"a","principal_type":"group","principal_id":"g1","permission":"read"}]}],"total":` + fmt.Sprintf("%d", knowledgeListPageSize+1) + `}`))
		default:
			t.Fatalf("unexpected page %q", page)
		}
	}))
	t.Cleanup(server.Close)

	c, err := NewClient(server.URL, "test-token")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	items, err := c.ListKnowledge(context.Background())
	if err != nil {
		t.Fatalf("ListKnowledge: %v", err)
	}
	if len(items) != knowledgeListPageSize+1 {
		t.Fatalf("expected %d items, got %d", knowledgeListPageSize+1, len(items))
	}
	for _, p := range paths {
		if p != "/knowledge/" {
			t.Fatalf("expected /knowledge/, got %q", p)
		}
	}
	last := items[len(items)-1]
	read := last.AccessControl["read"].(map[string]any)
	ids := read["group_ids"].([]string)
	if len(ids) != 1 || ids[0] != "g1" {
		t.Fatalf("expected last item read.group_ids=[g1], got %+v", last.AccessControl)
	}
}

// join concatenates JSON object strings with commas.
func join(items []string) string {
	out := ""
	for i, s := range items {
		if i > 0 {
			out += ","
		}
		out += s
	}
	return out
}
