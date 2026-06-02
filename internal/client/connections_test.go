package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestEncodeDecodeOpenAIConnections_roundtrip(t *testing.T) {
	entries := []OpenAIConnectionEntry{
		{
			URL: "https://api.openai.com/v1",
			Key: "sk-test",
			Config: OpenAIConnectionConfig{
				Enable:         true,
				Tags:           []string{"prod"},
				PrefixID:       "openai",
				ModelIDs:       []string{"gpt-4o"},
				ConnectionType: "external",
				AuthType:       "bearer",
				Provider:       "openai",
				APIVersion:     "",
				APIType:        "",
			},
		},
		{
			URL: "https://azure.example.com",
			Key: "azure-key",
			Config: OpenAIConnectionConfig{
				Enable:         false,
				Tags:           []string{},
				PrefixID:       "",
				ModelIDs:       []string{"gpt-4-deployment"},
				ConnectionType: "external",
				AuthType:       "bearer",
				Provider:       "azure",
				APIVersion:     "2024-02-01",
				APIType:        "",
			},
		},
	}

	wire := encodeOpenAIConnections(true, entries)

	if !wire.EnableOpenAIAPI {
		t.Error("EnableOpenAIAPI should be true")
	}
	if len(wire.BaseURLs) != 2 {
		t.Fatalf("want 2 BaseURLs, got %d", len(wire.BaseURLs))
	}
	if wire.BaseURLs[0] != "https://api.openai.com/v1" {
		t.Errorf("BaseURLs[0] = %q", wire.BaseURLs[0])
	}
	if wire.Keys[1] != "azure-key" {
		t.Errorf("Keys[1] = %q", wire.Keys[1])
	}
	if _, ok := wire.Configs["0"]; !ok {
		t.Error("expected config at key '0'")
	}
	if _, ok := wire.Configs["1"]; !ok {
		t.Error("expected config at key '1'")
	}

	enabled, decoded := decodeOpenAIConnections(wire)
	if !enabled {
		t.Error("enabled should be true")
	}
	if len(decoded) != 2 {
		t.Fatalf("want 2 entries, got %d", len(decoded))
	}
	if decoded[0].URL != entries[0].URL || decoded[0].Key != entries[0].Key {
		t.Errorf("decoded[0] URL/Key mismatch: %+v", decoded[0])
	}
	if !reflect.DeepEqual(decoded[0].Config.Tags, entries[0].Config.Tags) {
		t.Errorf("decoded[0].Config.Tags = %v, want %v", decoded[0].Config.Tags, entries[0].Config.Tags)
	}
	if decoded[1].Config.Provider != "azure" || decoded[1].Config.APIVersion != "2024-02-01" {
		t.Errorf("decoded[1] config mismatch: %+v", decoded[1].Config)
	}
	if decoded[1].Config.Enable {
		t.Error("decoded[1].Config.Enable should be false")
	}
}

func TestDecodeOpenAIConnections_missingConfig(t *testing.T) {
	wire := openAIConnectionsWire{
		EnableOpenAIAPI: false,
		BaseURLs:        []string{"http://localhost:8000"},
		Keys:            []string{""},
		Configs:         map[string]any{},
	}
	enabled, entries := decodeOpenAIConnections(wire)
	if enabled {
		t.Error("enabled should be false")
	}
	if len(entries) != 1 {
		t.Fatalf("want 1 entry, got %d", len(entries))
	}
	// Defaults should be applied
	if !entries[0].Config.Enable {
		t.Error("default Enable should be true")
	}
	if entries[0].Config.ConnectionType != "external" {
		t.Errorf("default ConnectionType = %q, want %q", entries[0].Config.ConnectionType, "external")
	}
	if entries[0].Config.AuthType != "bearer" {
		t.Errorf("default AuthType = %q, want %q", entries[0].Config.AuthType, "bearer")
	}
	if entries[0].Config.Tags == nil {
		t.Error("Tags should not be nil")
	}
	if entries[0].Config.ModelIDs == nil {
		t.Error("ModelIDs should not be nil")
	}
}

func TestEncodeDecodeOllamaConnections_roundtrip(t *testing.T) {
	entries := []OllamaConnectionEntry{
		{
			URL: "http://localhost:11434",
			Config: OllamaConnectionConfig{
				Enable:         true,
				Tags:           []string{"local"},
				PrefixID:       "llm",
				ModelIDs:       []string{"llama3.2"},
				ConnectionType: "local",
				Key:            "secret",
			},
		},
	}

	wire := encodeOllamaConnections(true, entries)
	if !wire.EnableOllamaAPI {
		t.Error("EnableOllamaAPI should be true")
	}
	if len(wire.BaseURLs) != 1 || wire.BaseURLs[0] != "http://localhost:11434" {
		t.Errorf("unexpected BaseURLs: %v", wire.BaseURLs)
	}
	if _, ok := wire.Configs["0"]; !ok {
		t.Error("expected config at key '0'")
	}

	enabled, decoded := decodeOllamaConnections(wire)
	if !enabled || len(decoded) != 1 {
		t.Fatalf("unexpected: enabled=%v len=%d", enabled, len(decoded))
	}
	if decoded[0].URL != "http://localhost:11434" {
		t.Errorf("URL = %q", decoded[0].URL)
	}
	if decoded[0].Config.Key != "secret" {
		t.Errorf("Key = %q", decoded[0].Config.Key)
	}
	if decoded[0].Config.PrefixID != "llm" {
		t.Errorf("PrefixID = %q", decoded[0].Config.PrefixID)
	}
}

func newConnectionsTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	c, err := NewClient(server.URL, "test-token", false)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func TestGetOpenAIConnections(t *testing.T) {
	c := newConnectionsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/openai/config" || r.Method != http.MethodGet {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("missing auth: %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"ENABLE_OPENAI_API": true,
			"OPENAI_API_BASE_URLS": ["https://api.openai.com/v1"],
			"OPENAI_API_KEYS": ["sk-abc"],
			"OPENAI_API_CONFIGS": {
				"0": {"enable": true, "tags": ["prod"], "prefix_id": "openai", "model_ids": [], "connection_type": "external", "auth_type": "bearer", "provider": "", "api_version": "", "api_type": ""}
			}
		}`))
	})

	enabled, entries, err := c.GetOpenAIConnections(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !enabled {
		t.Error("enabled should be true")
	}
	if len(entries) != 1 {
		t.Fatalf("want 1 entry, got %d", len(entries))
	}
	if entries[0].URL != "https://api.openai.com/v1" || entries[0].Key != "sk-abc" {
		t.Errorf("unexpected entry: %+v", entries[0])
	}
	if entries[0].Config.PrefixID != "openai" {
		t.Errorf("PrefixID = %q", entries[0].Config.PrefixID)
	}
	if len(entries[0].Config.Tags) != 1 || entries[0].Config.Tags[0] != "prod" {
		t.Errorf("Tags = %v", entries[0].Config.Tags)
	}
}

func TestSetOpenAIConnections(t *testing.T) {
	var gotBody openAIConnectionsWire
	c := newConnectionsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/openai/config/update" || r.Method != http.MethodPost {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"ENABLE_OPENAI_API": true,
			"OPENAI_API_BASE_URLS": ["https://api.openai.com/v1"],
			"OPENAI_API_KEYS": ["sk-abc"],
			"OPENAI_API_CONFIGS": {"0": {"enable": true, "tags": [], "prefix_id": "", "model_ids": [], "connection_type": "external", "auth_type": "bearer", "provider": "", "api_version": "", "api_type": ""}}
		}`))
	})

	entries := []OpenAIConnectionEntry{{URL: "https://api.openai.com/v1", Key: "sk-abc", Config: OpenAIConnectionConfig{Enable: true, ConnectionType: "external", AuthType: "bearer", Tags: []string{}, ModelIDs: []string{}}}}
	enabled, result, err := c.SetOpenAIConnections(context.Background(), true, entries)
	if err != nil {
		t.Fatal(err)
	}
	if !enabled || len(result) != 1 {
		t.Errorf("unexpected: enabled=%v len=%d", enabled, len(result))
	}
	if gotBody.BaseURLs[0] != "https://api.openai.com/v1" || gotBody.Keys[0] != "sk-abc" {
		t.Errorf("sent body: %+v", gotBody)
	}
}
