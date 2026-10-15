package client

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestGetLDAPServerConfig(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/auths/admin/config/ldap/server" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"label": "Corp LDAP", "host": "ldap.example.com", "port": 636,
			"attribute_for_mail": "mail", "attribute_for_username": "uid",
			"app_dn": "cn=admin,dc=example,dc=com", "app_dn_password": "s3cret",
			"search_base": "ou=people,dc=example,dc=com", "search_filters": "(objectClass=person)",
			"use_tls": true, "certificate_path": null, "validate_cert": false, "ciphers": "ALL",
			"enable_group_management": true, "enable_group_creation": false, "attribute_for_groups": "memberOf"
		}`))
	})
	client, server := newConfigTestClient(handler)
	defer server.Close()

	got, err := client.GetLDAPServerConfig(context.Background())
	if err != nil {
		t.Fatalf("GetLDAPServerConfig failed: %v", err)
	}
	if got.Label != "Corp LDAP" || got.Host != "ldap.example.com" || got.AppDNPassword != "s3cret" ||
		got.SearchBase != "ou=people,dc=example,dc=com" || got.AppDN != "cn=admin,dc=example,dc=com" {
		t.Errorf("unexpected scalar fields: %+v", got)
	}
	if got.Port == nil || *got.Port != 636 {
		t.Errorf("expected port 636, got %v", got.Port)
	}
	if got.CertificatePath != nil {
		t.Errorf("expected nil certificate_path, got %v", *got.CertificatePath)
	}
	if got.ValidateCert == nil || *got.ValidateCert || got.UseTLS == nil || !*got.UseTLS {
		t.Errorf("unexpected bool fields: %+v", got)
	}
	if got.EnableGroupManagement == nil || !*got.EnableGroupManagement || got.AttributeForGroups == nil || *got.AttributeForGroups != "memberOf" {
		t.Errorf("unexpected group fields: %+v", got)
	}
}

// TestGetLDAPServerConfigOlderServer covers a v0.9.6-shaped response that has
// no group keys and a null port.
func TestGetLDAPServerConfigOlderServer(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"label":"l","host":"h","port":null,"app_dn":"","app_dn_password":"","search_base":"b"}`))
	})
	client, server := newConfigTestClient(handler)
	defer server.Close()

	got, err := client.GetLDAPServerConfig(context.Background())
	if err != nil {
		t.Fatalf("GetLDAPServerConfig failed: %v", err)
	}
	if got.Port != nil || got.EnableGroupManagement != nil || got.AttributeForGroups != nil {
		t.Errorf("expected nil optional fields, got %+v", got)
	}
}

// TestSetLDAPServerConfig pins the literal lowercase wire keys: optional
// fields that are set are present; non-nullable optional fields left nil are
// omitted, while the nullable ones (certificate_path, ciphers) go out as an
// explicit null.
func TestSetLDAPServerConfig(t *testing.T) {
	var body map[string]any
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/auths/admin/config/ldap/server" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"label":"L","host":"H","port":389,"app_dn":"dn","app_dn_password":"pw","search_base":"sb","enable_group_management":true,"attribute_for_groups":"memberOf"}`))
	})
	client, server := newConfigTestClient(handler)
	defer server.Close()

	port := int64(389)
	tru := true
	groups := "memberOf"
	resp, err := client.SetLDAPServerConfig(context.Background(), LDAPServerConfigForm{
		Label: "L", Host: "H", Port: &port, AppDN: "dn", AppDNPassword: "pw", SearchBase: "sb",
		EnableGroupManagement: &tru, AttributeForGroups: &groups,
	})
	if err != nil {
		t.Fatalf("SetLDAPServerConfig failed: %v", err)
	}

	want := map[string]any{
		"label": "L", "host": "H", "port": float64(389), "app_dn": "dn",
		"app_dn_password": "pw", "search_base": "sb",
		"enable_group_management": true, "attribute_for_groups": "memberOf",
	}
	for k, v := range want {
		if body[k] != v {
			t.Errorf("wire key %q: expected %v, got %v", k, v, body[k])
		}
	}
	for _, k := range []string{"attribute_for_mail", "attribute_for_username", "search_filters", "use_tls", "validate_cert", "enable_group_creation"} {
		if _, present := body[k]; present {
			t.Errorf("expected unset key %q to be omitted, body: %v", k, body)
		}
	}
	for _, k := range []string{"certificate_path", "ciphers"} {
		if v, present := body[k]; !present || v != nil {
			t.Errorf("expected nullable key %q sent as null, got %v (present=%v)", k, v, present)
		}
	}
	if resp.Port == nil || *resp.Port != 389 || resp.EnableGroupManagement == nil || !*resp.EnableGroupManagement {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestSetLDAPServerConfigExplicitFalseSent(t *testing.T) {
	var body map[string]any
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"label":"L","host":"H","app_dn":"","app_dn_password":"","search_base":"sb"}`))
	})
	client, server := newConfigTestClient(handler)
	defer server.Close()

	f := false
	if _, err := client.SetLDAPServerConfig(context.Background(), LDAPServerConfigForm{
		Label: "L", Host: "H", SearchBase: "sb", UseTLS: &f, ValidateCert: &f,
	}); err != nil {
		t.Fatal(err)
	}
	if v, ok := body["use_tls"]; !ok || v != false {
		t.Errorf("expected use_tls=false on the wire, got %v (present=%v)", v, ok)
	}
	if v, ok := body["validate_cert"]; !ok || v != false {
		t.Errorf("expected validate_cert=false on the wire, got %v (present=%v)", v, ok)
	}
	if v, ok := body["app_dn"]; !ok || v != "" {
		t.Errorf("expected required app_dn always sent, got %v", v)
	}
	if v, ok := body["port"]; !ok || v != nil {
		t.Errorf("expected nil port sent as null, got %v (present=%v)", v, ok)
	}
}
