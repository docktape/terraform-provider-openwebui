package client

import (
	"context"
	"net/http"
	"testing"
)

func TestGetVersion(t *testing.T) {
	var gotPath, gotMethod string
	c := newGroupTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		_, _ = w.Write([]byte(`{"version":"0.12.0","deployment_id":"x"}`))
	})

	got, err := c.GetVersion(context.Background())
	if err != nil {
		t.Fatalf("GetVersion: %v", err)
	}
	if got != "0.12.0" {
		t.Errorf("version = %q, want 0.12.0", got)
	}
	if gotPath != "/api/version" || gotMethod != http.MethodGet {
		t.Errorf("request = %s %s, want GET /api/version", gotMethod, gotPath)
	}
}

func TestGetVersionServerError(t *testing.T) {
	c := newGroupTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"detail":"boom"}`, http.StatusInternalServerError)
	})
	if _, err := c.GetVersion(context.Background()); err == nil {
		t.Fatal("expected an error for HTTP 500")
	}
}
