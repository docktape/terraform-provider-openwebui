package client

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
)

// OpenAIConnectionConfig holds per-connection settings stored in OPENAI_API_CONFIGS[idx].
type OpenAIConnectionConfig struct {
	Enable         bool           `json:"enable"`
	Tags           []string       `json:"tags"`
	PrefixID       string         `json:"prefix_id"`
	ModelIDs       []string       `json:"model_ids"`
	ConnectionType string         `json:"connection_type"`
	AuthType       string         `json:"auth_type"`
	Headers        map[string]any `json:"headers,omitempty"`
	Provider       string         `json:"provider"`
	APIVersion     string         `json:"api_version"`
	APIType        string         `json:"api_type"`
}

// OpenAIConnectionEntry is one OpenAI-compatible connection.
type OpenAIConnectionEntry struct {
	URL    string
	Key    string
	Config OpenAIConnectionConfig
}

// OllamaConnectionConfig holds per-connection settings stored in OLLAMA_API_CONFIGS[idx].
// Key is stored inside the config dict (not in a parallel array like OpenAI keys).
type OllamaConnectionConfig struct {
	Enable         bool     `json:"enable"`
	Tags           []string `json:"tags"`
	PrefixID       string   `json:"prefix_id"`
	ModelIDs       []string `json:"model_ids"`
	ConnectionType string   `json:"connection_type"`
	Key            string   `json:"key,omitempty"`
}

// OllamaConnectionEntry is one Ollama connection.
type OllamaConnectionEntry struct {
	URL    string
	Config OllamaConnectionConfig
}

// openAIConnectionsWire is the JSON shape of GET/POST /openai/config.
type openAIConnectionsWire struct {
	EnableOpenAIAPI bool           `json:"ENABLE_OPENAI_API"`
	BaseURLs        []string       `json:"OPENAI_API_BASE_URLS"`
	Keys            []string       `json:"OPENAI_API_KEYS"`
	Configs         map[string]any `json:"OPENAI_API_CONFIGS"`
}

// ollamaConnectionsWire is the JSON shape of GET/POST /ollama/config.
type ollamaConnectionsWire struct {
	EnableOllamaAPI bool           `json:"ENABLE_OLLAMA_API"`
	BaseURLs        []string       `json:"OLLAMA_BASE_URLS"`
	Configs         map[string]any `json:"OLLAMA_API_CONFIGS"`
}

// openAIVerifyForm is the body for POST /openai/verify.
type openAIVerifyForm struct {
	URL    string                  `json:"url"`
	Key    string                  `json:"key"`
	Config *OpenAIConnectionConfig `json:"config,omitempty"`
}

// ollamaVerifyForm is the body for POST /ollama/verify.
type ollamaVerifyForm struct {
	URL string  `json:"url"`
	Key *string `json:"key,omitempty"`
}

func encodeOpenAIConnections(enabled bool, entries []OpenAIConnectionEntry) openAIConnectionsWire {
	urls := make([]string, len(entries))
	keys := make([]string, len(entries))
	configs := make(map[string]any, len(entries))
	for i, e := range entries {
		urls[i] = e.URL
		keys[i] = e.Key
		b, _ := json.Marshal(e.Config)
		var m map[string]any
		_ = json.Unmarshal(b, &m)
		if m == nil {
			m = map[string]any{}
		}
		configs[strconv.Itoa(i)] = m
	}
	return openAIConnectionsWire{
		EnableOpenAIAPI: enabled,
		BaseURLs:        urls,
		Keys:            keys,
		Configs:         configs,
	}
}

func decodeOpenAIConnections(wire openAIConnectionsWire) (bool, []OpenAIConnectionEntry) {
	n := len(wire.BaseURLs)
	entries := make([]OpenAIConnectionEntry, n)
	for i := 0; i < n; i++ {
		key := ""
		if i < len(wire.Keys) {
			key = wire.Keys[i]
		}
		cfg := OpenAIConnectionConfig{
			Enable:         true,
			Tags:           []string{},
			ModelIDs:       []string{},
			ConnectionType: "external",
			AuthType:       "bearer",
		}
		if raw, ok := wire.Configs[strconv.Itoa(i)]; ok {
			b, err := json.Marshal(raw)
			if err == nil {
				_ = json.Unmarshal(b, &cfg)
			}
		}
		if cfg.Tags == nil {
			cfg.Tags = []string{}
		}
		if cfg.ModelIDs == nil {
			cfg.ModelIDs = []string{}
		}
		entries[i] = OpenAIConnectionEntry{URL: wire.BaseURLs[i], Key: key, Config: cfg}
	}
	return wire.EnableOpenAIAPI, entries
}

func encodeOllamaConnections(enabled bool, entries []OllamaConnectionEntry) ollamaConnectionsWire {
	urls := make([]string, len(entries))
	configs := make(map[string]any, len(entries))
	for i, e := range entries {
		urls[i] = e.URL
		b, _ := json.Marshal(e.Config)
		var m map[string]any
		_ = json.Unmarshal(b, &m)
		if m == nil {
			m = map[string]any{}
		}
		configs[strconv.Itoa(i)] = m
	}
	return ollamaConnectionsWire{
		EnableOllamaAPI: enabled,
		BaseURLs:        urls,
		Configs:         configs,
	}
}

func decodeOllamaConnections(wire ollamaConnectionsWire) (bool, []OllamaConnectionEntry) {
	n := len(wire.BaseURLs)
	entries := make([]OllamaConnectionEntry, n)
	for i := 0; i < n; i++ {
		cfg := OllamaConnectionConfig{
			Enable:         true,
			Tags:           []string{},
			ModelIDs:       []string{},
			ConnectionType: "local",
		}
		if raw, ok := wire.Configs[strconv.Itoa(i)]; ok {
			b, err := json.Marshal(raw)
			if err == nil {
				_ = json.Unmarshal(b, &cfg)
			}
		}
		if cfg.Tags == nil {
			cfg.Tags = []string{}
		}
		if cfg.ModelIDs == nil {
			cfg.ModelIDs = []string{}
		}
		entries[i] = OllamaConnectionEntry{URL: wire.BaseURLs[i], Config: cfg}
	}
	return wire.EnableOllamaAPI, entries
}

// GetOpenAIConnections retrieves the current OpenAI-compatible connections configuration.
func (c *Client) GetOpenAIConnections(ctx context.Context) (bool, []OpenAIConnectionEntry, error) {
	var wire openAIConnectionsWire
	if err := c.doRaw(ctx, http.MethodGet, c.rootURL+"/openai/config", nil, &wire); err != nil {
		return false, nil, err
	}
	enabled, entries := decodeOpenAIConnections(wire)
	return enabled, entries, nil
}

// SetOpenAIConnections updates the OpenAI-compatible connections configuration.
func (c *Client) SetOpenAIConnections(ctx context.Context, enabled bool, entries []OpenAIConnectionEntry) (bool, []OpenAIConnectionEntry, error) {
	wire := encodeOpenAIConnections(enabled, entries)
	var result openAIConnectionsWire
	if err := c.doRaw(ctx, http.MethodPost, c.rootURL+"/openai/config/update", wire, &result); err != nil {
		return false, nil, err
	}
	retEnabled, retEntries := decodeOpenAIConnections(result)
	return retEnabled, retEntries, nil
}

// GetOllamaConnections retrieves the current Ollama connections configuration.
func (c *Client) GetOllamaConnections(ctx context.Context) (bool, []OllamaConnectionEntry, error) {
	var wire ollamaConnectionsWire
	if err := c.doRaw(ctx, http.MethodGet, c.rootURL+"/ollama/config", nil, &wire); err != nil {
		return false, nil, err
	}
	enabled, entries := decodeOllamaConnections(wire)
	return enabled, entries, nil
}

// SetOllamaConnections updates the Ollama connections configuration.
func (c *Client) SetOllamaConnections(ctx context.Context, enabled bool, entries []OllamaConnectionEntry) (bool, []OllamaConnectionEntry, error) {
	wire := encodeOllamaConnections(enabled, entries)
	var result ollamaConnectionsWire
	if err := c.doRaw(ctx, http.MethodPost, c.rootURL+"/ollama/config/update", wire, &result); err != nil {
		return false, nil, err
	}
	retEnabled, retEntries := decodeOllamaConnections(result)
	return retEnabled, retEntries, nil
}

// VerifyOpenAIConnection tests that an OpenAI-compatible endpoint is reachable.
func (c *Client) VerifyOpenAIConnection(ctx context.Context, apiURL, key string, config *OpenAIConnectionConfig) error {
	form := openAIVerifyForm{URL: apiURL, Key: key, Config: config}
	return c.doRaw(ctx, http.MethodPost, c.rootURL+"/openai/verify", form, nil)
}

// VerifyOllamaConnection tests that an Ollama endpoint is reachable.
func (c *Client) VerifyOllamaConnection(ctx context.Context, apiURL string, key *string) error {
	form := ollamaVerifyForm{URL: apiURL, Key: key}
	return c.doRaw(ctx, http.MethodPost, c.rootURL+"/ollama/verify", form, nil)
}
