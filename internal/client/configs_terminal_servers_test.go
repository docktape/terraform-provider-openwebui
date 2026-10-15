package client

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

// Mirrors GET /configs/terminal_servers at v0.11.0: stored entries may carry
// extra keys (policy) and null values.
const terminalServersResponse = `{"TERMINAL_SERVER_CONNECTIONS": [
	{"id": "t1", "name": "Dev", "enabled": true, "url": "http://term:8000", "path": "/openapi.json",
	 "key": "secret", "auth_type": "bearer", "config": {"access_grants": [{"principal_type": "user", "principal_id": "*", "permission": "read"}]},
	 "server_type": "orchestrator", "policy_id": "p1", "policy": {"image": "x"}},
	{"id": "", "name": "", "enabled": false, "url": "http://other", "path": null, "key": "", "auth_type": "none",
	 "config": null, "server_type": null, "policy_id": null}
]}`

func TestGetTerminalServersConfig(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/configs/terminal_servers" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(terminalServersResponse))
	})
	c, server := newConfigTestClient(handler)
	defer server.Close()

	got, err := c.GetTerminalServersConfig(context.Background())
	if err != nil {
		t.Fatalf("GetTerminalServersConfig failed: %v", err)
	}
	if len(got.Connections) != 2 {
		t.Fatalf("expected 2 connections, got %d", len(got.Connections))
	}
	a, b := got.Connections[0], got.Connections[1]
	if *a.ID != "t1" || *a.Name != "Dev" || !*a.Enabled || a.URL != "http://term:8000" || *a.Path != "/openapi.json" ||
		*a.Key != "secret" || *a.AuthType != "bearer" || *a.ServerType != "orchestrator" || *a.PolicyID != "p1" {
		t.Errorf("unexpected first connection %+v", a)
	}
	var config map[string]any
	if err := json.Unmarshal(a.Config, &config); err != nil || config["access_grants"] == nil {
		t.Errorf("config not decoded: %s", a.Config)
	}
	if *b.Enabled || b.Path != nil || string(b.Config) != "null" || b.ServerType != nil || b.PolicyID != nil || *b.AuthType != "none" {
		t.Errorf("unexpected second connection %+v", b)
	}
}

func TestGetTerminalServersConfigNullList(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"TERMINAL_SERVER_CONNECTIONS": null}`))
	})
	c, server := newConfigTestClient(handler)
	defer server.Close()
	got, err := c.GetTerminalServersConfig(context.Background())
	if err != nil {
		t.Fatalf("GetTerminalServersConfig failed: %v", err)
	}
	if len(got.Connections) != 0 {
		t.Errorf("expected no connections, got %v", got.Connections)
	}
}

func TestSetTerminalServersConfigEveryWireKey(t *testing.T) {
	var body map[string]any
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/configs/terminal_servers" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		raw, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		_, _ = w.Write([]byte(terminalServersResponse))
	})
	c, server := newConfigTestClient(handler)
	defer server.Close()

	s := func(v string) *string { return &v }
	enabled := true
	form := TerminalServersConfigForm{Connections: []TerminalServerConnection{
		{
			ID: s("t1"), Name: s("Dev"), Enabled: &enabled, URL: "http://term:8000", Path: s("/openapi.json"),
			Key: s("secret"), AuthType: s("bearer"), Config: json.RawMessage(`{"a":1}`),
			ServerType: s("orchestrator"), PolicyID: s("p1"),
		},
		{URL: "http://bare"},
	}}
	got, err := c.SetTerminalServersConfig(context.Background(), form)
	if err != nil {
		t.Fatalf("SetTerminalServersConfig failed: %v", err)
	}
	if len(got.Connections) != 2 {
		t.Errorf("response not decoded: %+v", got)
	}

	if len(body) != 1 {
		t.Fatalf("expected only TERMINAL_SERVER_CONNECTIONS, got %v", body)
	}
	list, ok := body["TERMINAL_SERVER_CONNECTIONS"].([]any)
	if !ok || len(list) != 2 {
		t.Fatalf("unexpected list %v", body["TERMINAL_SERVER_CONNECTIONS"])
	}
	wantKeys := []string{"id", "name", "enabled", "url", "path", "key", "auth_type", "config", "server_type", "policy_id"}
	first, _ := list[0].(map[string]any)
	if len(first) != len(wantKeys) {
		t.Errorf("unexpected key count %d: %v", len(first), first)
	}
	want := map[string]any{
		"id": "t1", "name": "Dev", "enabled": true, "url": "http://term:8000", "path": "/openapi.json",
		"key": "secret", "auth_type": "bearer", "config": map[string]any{"a": 1.0},
		"server_type": "orchestrator", "policy_id": "p1",
	}
	if !reflect.DeepEqual(first, want) {
		t.Errorf("unexpected first entry %v", first)
	}
	// Nil pointers go out as explicit nulls, never omitted.
	second, _ := list[1].(map[string]any)
	for _, k := range wantKeys {
		if _, present := second[k]; !present {
			t.Errorf("key %s missing from zero-value entry", k)
		}
	}
	if second["url"] != "http://bare" || second["path"] != nil || second["config"] != nil {
		t.Errorf("unexpected zero-value entry %v", second)
	}
}

// config is passed through as raw JSON: an integer beyond float64 precision
// and a trailing-zero decimal reach the wire as written, and come back as
// the server wrote them.
func TestTerminalServersConfigKeepsNumberLiterals(t *testing.T) {
	var raw []byte
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ = io.ReadAll(r.Body)
		_, _ = w.Write([]byte(`{"TERMINAL_SERVER_CONNECTIONS": [{"url": "http://t", "config": {"big": 9007199254740993, "speed": 1.5}}]}`))
	})
	c, server := newConfigTestClient(handler)
	defer server.Close()

	got, err := c.SetTerminalServersConfig(context.Background(), TerminalServersConfigForm{Connections: []TerminalServerConnection{
		{URL: "http://t", Config: json.RawMessage(`{"big":9007199254740993,"speed":1.50}`)},
	}})
	if err != nil {
		t.Fatalf("SetTerminalServersConfig failed: %v", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var body map[string]any
	if err := decoder.Decode(&body); err != nil {
		t.Fatal(err)
	}
	list, _ := body["TERMINAL_SERVER_CONNECTIONS"].([]any)
	if len(list) != 1 {
		t.Fatalf("unexpected body %s", raw)
	}
	entry, _ := list[0].(map[string]any)
	config, _ := entry["config"].(map[string]any)
	if config["big"] != json.Number("9007199254740993") || config["speed"] != json.Number("1.50") {
		t.Errorf("number literals changed on the wire: %s", raw)
	}
	if string(got.Connections[0].Config) != `{"big": 9007199254740993, "speed": 1.5}` {
		t.Errorf("response config not kept verbatim: %s", got.Connections[0].Config)
	}
}

// A nil connection slice must be sent as [] (list[...] is not nullable).
func TestSetTerminalServersConfigNilListSendsEmptyArray(t *testing.T) {
	var raw []byte
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ = io.ReadAll(r.Body)
		_, _ = w.Write([]byte(`{"TERMINAL_SERVER_CONNECTIONS": []}`))
	})
	c, server := newConfigTestClient(handler)
	defer server.Close()
	if _, err := c.SetTerminalServersConfig(context.Background(), TerminalServersConfigForm{}); err != nil {
		t.Fatalf("SetTerminalServersConfig failed: %v", err)
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(body["TERMINAL_SERVER_CONNECTIONS"], []any{}) {
		t.Errorf("expected [], got %v", body["TERMINAL_SERVER_CONNECTIONS"])
	}
}

func TestVerifyTerminalServer(t *testing.T) {
	var body map[string]any
	status := http.StatusOK
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/configs/terminal_servers/verify" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		w.WriteHeader(status)
		if status == http.StatusOK {
			_, _ = w.Write([]byte(`{"status": true, "type": "orchestrator"}`))
			return
		}
		_, _ = w.Write([]byte(`{"detail": "Failed to connect to the terminal server"}`))
	})
	c, server := newConfigTestClient(handler)
	defer server.Close()

	key, auth := "k", "bearer"
	got, err := c.VerifyTerminalServer(context.Background(), TerminalServerConnection{URL: "http://term", Key: &key, AuthType: &auth})
	if err != nil {
		t.Fatalf("VerifyTerminalServer failed: %v", err)
	}
	if !got.Status || got.Type != "orchestrator" {
		t.Errorf("unexpected result %+v", got)
	}
	if body["url"] != "http://term" || body["key"] != "k" || body["auth_type"] != "bearer" {
		t.Errorf("unexpected body %v", body)
	}

	status = http.StatusBadRequest
	if _, err := c.VerifyTerminalServer(context.Background(), TerminalServerConnection{URL: "http://down"}); err == nil {
		t.Errorf("expected an error for HTTP 400")
	}
}

// TestTerminalServersConfigRawRoundTrip: the raw methods hit the same endpoint
// and pass extra per-connection keys through untouched in both directions.
func TestTerminalServersConfigRawRoundTrip(t *testing.T) {
	const stored = `{"TERMINAL_SERVER_CONNECTIONS":[{"id":"a","url":"http://t","policy":{"cpu":2},"custom":1.50}]}`
	var postedBody string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/configs/terminal_servers" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if r.Method == http.MethodPost {
			raw, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatal(err)
			}
			postedBody = strings.TrimSpace(string(raw))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(stored))
	})
	c, server := newConfigTestClient(handler)
	defer server.Close()

	got, err := c.GetTerminalServersConfigRaw(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != stored {
		t.Fatalf("raw GET altered the body: %s", got)
	}
	if _, err := c.SetTerminalServersConfigRaw(context.Background(), got); err != nil {
		t.Fatal(err)
	}
	if postedBody != stored {
		t.Errorf("raw POST altered the body:\n got %s\nwant %s", postedBody, stored)
	}
}
