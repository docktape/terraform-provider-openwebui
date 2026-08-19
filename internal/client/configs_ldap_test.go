package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newLDAPConfigTestClient(handler http.HandlerFunc) (*Client, *httptest.Server) {
	server := httptest.NewServer(handler)
	client, err := NewClient(server.URL, "test-token", false)
	if err != nil {
		panic(err)
	}
	return client, server
}

func TestGetLDAPConfig(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/auths/admin/config/ldap" {
			t.Errorf("Expected path /api/v1/auths/admin/config/ldap, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ENABLE_LDAP": true,
		})
	})

	client, server := newLDAPConfigTestClient(handler)
	defer server.Close()

	resp, err := client.GetLDAPConfig(context.Background())
	if err != nil {
		t.Fatalf("GetLDAPConfig failed: %v", err)
	}

	if resp == nil {
		t.Fatalf("GetLDAPConfig returned nil")
	}

	if resp.EnableLDAP == nil || *resp.EnableLDAP != true {
		t.Errorf("Expected EnableLDAP true, got %v", resp.EnableLDAP)
	}
}

func TestSetLDAPConfig(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/auths/admin/config/ldap" {
			t.Errorf("Expected path /api/v1/auths/admin/config/ldap, got %s", r.URL.Path)
		}

		var req LDAPConfigForm
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ENABLE_LDAP": req.EnableLDAP,
		})
	})

	client, server := newLDAPConfigTestClient(handler)
	defer server.Close()

	enableTrue := true
	form := LDAPConfigForm{
		EnableLDAP: &enableTrue,
	}

	resp, err := client.SetLDAPConfig(context.Background(), form)
	if err != nil {
		t.Fatalf("SetLDAPConfig failed: %v", err)
	}

	if resp == nil {
		t.Fatalf("SetLDAPConfig returned nil")
	}

	if resp.EnableLDAP == nil || *resp.EnableLDAP != true {
		t.Errorf("Expected EnableLDAP true, got %v", resp.EnableLDAP)
	}
}
