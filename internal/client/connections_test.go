package client

import (
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
