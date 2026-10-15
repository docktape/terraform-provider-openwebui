package client

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestGetLDAPConfig(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/auths/admin/config/ldap" {
			t.Errorf("Expected path /api/v1/auths/admin/config/ldap, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"ENABLE_LDAP": true,
		}); err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	})

	client, server := newConfigTestClient(handler)
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

// TestSetLDAPConfig verifies both the wire-level shape of the POST request
// body and the response decoding. The real Open WebUI backend
// (backend/open_webui/routers/auths.py) uses an asymmetric shape here: the
// POST body is decoded server-side into LdapConfigForm{enable_ldap: bool |
// None} (lowercase), while both the GET and POST responses are
// {"ENABLE_LDAP": ...} (uppercase). Decoding the captured request body into
// the client's own LDAPConfigForm struct (which is uppercase-tagged, matching
// only the response shape) would NOT catch a regression where SetLDAPConfig
// sends the wrong casing - it would just silently decode to a nil field on
// both ends. Decoding into a plain map and asserting on the literal JSON key
// is what actually pins the wire contract down.
func TestSetLDAPConfig(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/auths/admin/config/ldap" {
			t.Errorf("Expected path /api/v1/auths/admin/config/ldap, got %s", r.URL.Path)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		rawValue, ok := body["enable_ldap"]
		if !ok {
			t.Fatalf("Expected request body to contain lowercase key %q, got %v", "enable_ldap", body)
		}
		if _, hasUppercase := body["ENABLE_LDAP"]; hasUppercase {
			t.Errorf("Request body should not contain uppercase key %q (that's the response shape, not the request shape): %v", "ENABLE_LDAP", body)
		}

		value, _ := rawValue.(bool)

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"ENABLE_LDAP": value,
		}); err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	})

	client, server := newConfigTestClient(handler)
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
