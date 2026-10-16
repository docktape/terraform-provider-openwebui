package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
)

// RetrievalEmbeddingConfig is the GET /retrieval/embedding response
// (get_embedding_config). Every field is a pointer so that an absent or null
// key reads as nil. The integers are decoded leniently (see UnmarshalJSON).
// The response's constant `status: true` is not modelled.
type RetrievalEmbeddingConfig struct {
	RAGEmbeddingEngine             *string                     `json:"RAG_EMBEDDING_ENGINE"`
	RAGEmbeddingModel              *string                     `json:"RAG_EMBEDDING_MODEL"`
	RAGEmbeddingBatchSize          *int64                      `json:"-"`
	EnableAsyncEmbedding           *bool                       `json:"ENABLE_ASYNC_EMBEDDING"`
	RAGEmbeddingConcurrentRequests *int64                      `json:"-"`
	OpenAIConfig                   *EmbeddingConnectionConfig  `json:"openai_config"`
	OllamaConfig                   *EmbeddingConnectionConfig  `json:"ollama_config"`
	AzureOpenAIConfig              *EmbeddingAzureOpenAIConfig `json:"azure_openai_config"`
}

// EmbeddingConnectionConfig is the {url, key} object of the OpenAI and Ollama
// embedding connections (RAG_OPENAI_API_BASE_URL/RAG_OPENAI_API_KEY and
// RAG_OLLAMA_BASE_URL/RAG_OLLAMA_API_KEY). In an update a nil field keeps its
// current value.
type EmbeddingConnectionConfig struct {
	URL *string `json:"url,omitempty"`
	Key *string `json:"key,omitempty"`
}

// EmbeddingAzureOpenAIConfig is the {url, key, version} object of the Azure
// OpenAI embedding connection. In an update a nil field keeps its current
// value.
type EmbeddingAzureOpenAIConfig struct {
	URL     *string `json:"url,omitempty"`
	Key     *string `json:"key,omitempty"`
	Version *string `json:"version,omitempty"`
}

// UnmarshalJSON decodes the two integers via decodeIntOrEmptyString (a JSON
// number, a numeric string, or ""/null as nil).
func (e *RetrievalEmbeddingConfig) UnmarshalJSON(data []byte) error {
	type alias RetrievalEmbeddingConfig
	aux := struct {
		*alias
		RAGEmbeddingBatchSize          json.RawMessage `json:"RAG_EMBEDDING_BATCH_SIZE"`
		RAGEmbeddingConcurrentRequests json.RawMessage `json:"RAG_EMBEDDING_CONCURRENT_REQUESTS"`
	}{alias: (*alias)(e)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	var err error
	if e.RAGEmbeddingBatchSize, err = decodeIntOrEmptyString(aux.RAGEmbeddingBatchSize); err != nil {
		return fmt.Errorf("RAG_EMBEDDING_BATCH_SIZE: %w", err)
	}
	if e.RAGEmbeddingConcurrentRequests, err = decodeIntOrEmptyString(aux.RAGEmbeddingConcurrentRequests); err != nil {
		return fmt.Errorf("RAG_EMBEDDING_CONCURRENT_REQUESTS: %w", err)
	}
	return nil
}

// RetrievalEmbeddingConfigUpdate holds the embedding settings to change. A nil
// field keeps its current Open WebUI value; a nested object with only some
// fields set keeps the others.
type RetrievalEmbeddingConfigUpdate struct {
	RAGEmbeddingEngine             *string                     `json:"RAG_EMBEDDING_ENGINE,omitempty"`
	RAGEmbeddingModel              *string                     `json:"RAG_EMBEDDING_MODEL,omitempty"`
	RAGEmbeddingBatchSize          *int64                      `json:"RAG_EMBEDDING_BATCH_SIZE,omitempty"`
	EnableAsyncEmbedding           *bool                       `json:"ENABLE_ASYNC_EMBEDDING,omitempty"`
	RAGEmbeddingConcurrentRequests *int64                      `json:"RAG_EMBEDDING_CONCURRENT_REQUESTS,omitempty"`
	OpenAIConfig                   *EmbeddingConnectionConfig  `json:"openai_config,omitempty"`
	OllamaConfig                   *EmbeddingConnectionConfig  `json:"ollama_config,omitempty"`
	AzureOpenAIConfig              *EmbeddingAzureOpenAIConfig `json:"azure_openai_config,omitempty"`
}

// GetRetrievalEmbeddingConfig retrieves the embedding model configuration.
// The connection keys are returned in plain text.
func (c *Client) GetRetrievalEmbeddingConfig(ctx context.Context) (*RetrievalEmbeddingConfig, error) {
	var resp RetrievalEmbeddingConfig
	if err := c.do(ctx, http.MethodGet, "retrieval/embedding", nil, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdateRetrievalEmbeddingConfig changes the embedding settings set in update
// and keeps every other one.
//
// POST /retrieval/embedding/update (EmbeddingModelUpdateForm) is not a
// partial update: RAG_EMBEDDING_ENGINE and RAG_EMBEDDING_MODEL are required,
// and RAG_EMBEDDING_BATCH_SIZE, ENABLE_ASYNC_EMBEDDING and
// RAG_EMBEDDING_CONCURRENT_REQUESTS are reset to their defaults (1, true, 0)
// when left out. The handler assigns all five unconditionally. This method
// therefore reads the current settings, overlays the fields set in update and
// sends all five scalars. The nested connection objects are different: an
// omitted object keeps its stored values, and a sent one must be complete
// (url, key and version are required strings), so only a changed object is
// sent, completed with its stored values.
//
// Open WebUI stores the sent nested objects only while the engine after the
// update is ollama, openai or azure_openai (any of the three objects for any
// of these engines), and drops them silently for the local
// sentence-transformers engine (""), on every release from v0.9.6 to v0.11.0.
// A changed object is therefore refused, before anything is sent, unless the
// engine after this update is one of the three remote engines.
//
// Every POST makes Open WebUI unload and reload the embedding model (see the
// resource documentation), so nothing is sent when the update changes
// nothing: no field set, or every field set to its current value.
// The endpoint stores only the embedding keys and is unrelated to
// /retrieval/config/update (whose handler never assigns them), so it does not
// take the retrieval write lock.
func (c *Client) UpdateRetrievalEmbeddingConfig(ctx context.Context, update RetrievalEmbeddingConfigUpdate) error {
	encoded, err := json.Marshal(update)
	if err != nil {
		return err
	}
	var changes map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &changes); err != nil {
		return err
	}
	if len(changes) == 0 {
		return nil
	}

	var current map[string]json.RawMessage
	if err := c.do(ctx, http.MethodGet, "retrieval/embedding", nil, nil, &current); err != nil {
		return err
	}
	body, changed, err := mergeRetrievalEmbeddingForm(current, changes)
	if err != nil {
		return err
	}
	if !changed {
		return nil
	}
	return c.do(ctx, http.MethodPost, "retrieval/embedding/update", nil, body, nil)
}

// embeddingNestedForms lists the nested connection objects with their
// required fields.
var embeddingNestedForms = []struct {
	key    string
	fields []string
}{
	{"openai_config", []string{"url", "key"}},
	{"ollama_config", []string{"url", "key"}},
	{"azure_openai_config", []string{"url", "key", "version"}},
}

// embeddingRemoteEngines are the engines for which update_embedding_config
// stores the nested connection objects.
var embeddingRemoteEngines = map[string]bool{"ollama": true, "openai": true, "azure_openai": true}

// mergeRetrievalEmbeddingForm builds the complete update form from the
// current raw GET response and the raw changes (see
// UpdateRetrievalEmbeddingConfig). changed reports whether the form differs
// from the current settings. Error messages name keys, never values.
func mergeRetrievalEmbeddingForm(current, changes map[string]json.RawMessage) (body map[string]json.RawMessage, changed bool, err error) {
	body = map[string]json.RawMessage{}
	pick := func(key string) (json.RawMessage, bool) {
		if v, ok := changes[key]; ok {
			return v, true
		}
		v, ok := current[key]
		return v, ok
	}

	// Required strings: there is no default to fall back on.
	for _, key := range []string{"RAG_EMBEDDING_ENGINE", "RAG_EMBEDDING_MODEL"} {
		v, ok := pick(key)
		if !ok || !isJSONString(v) {
			return nil, false, fmt.Errorf("the embedding settings read from Open WebUI have no %s string to keep; refusing to send an update without it (set it explicitly)", key)
		}
		body[key] = v
		changed = changed || !sameJSONValue(v, current[key])
	}
	// Optional scalars (`int | None`, `bool | None`): an unset one is echoed
	// as stored, null included. A stored value Pydantic would reject (such as
	// "" for an int) is left out, so its form default applies.
	for _, s := range []struct {
		key   string
		valid func(json.RawMessage) bool
	}{
		{"RAG_EMBEDDING_BATCH_SIZE", isJSONIntOrNull},
		{"ENABLE_ASYNC_EMBEDDING", isJSONBoolOrNull},
		{"RAG_EMBEDDING_CONCURRENT_REQUESTS", isJSONIntOrNull},
	} {
		if v, ok := changes[s.key]; ok {
			body[s.key] = v
			changed = changed || !sameJSONValue(v, current[s.key])
		} else if v, ok := current[s.key]; ok && s.valid(v) {
			body[s.key] = v
		} else {
			// Left out: Open WebUI stores the form default instead.
			changed = true
		}
	}

	var engine string
	if err := json.Unmarshal(body["RAG_EMBEDDING_ENGINE"], &engine); err != nil {
		return nil, false, fmt.Errorf("decode RAG_EMBEDDING_ENGINE: %w", err)
	}
	for _, form := range embeddingNestedForms {
		raw, ok := changes[form.key]
		if !ok {
			continue
		}
		var sent, stored map[string]json.RawMessage
		if err := json.Unmarshal(raw, &sent); err != nil {
			return nil, false, fmt.Errorf("decode %s: %w", form.key, err)
		}
		if isJSONObjectText(current[form.key]) {
			if err := json.Unmarshal(current[form.key], &stored); err != nil {
				return nil, false, fmt.Errorf("decode current %s: %w", form.key, err)
			}
		}
		merged := map[string]json.RawMessage{}
		same := true
		for _, field := range form.fields {
			v, ok := sent[field]
			if !ok {
				v = stored[field]
			}
			if !isJSONString(v) {
				// A required string: a null or missing stored value is sent as "".
				v = json.RawMessage(`""`)
			}
			merged[field] = v
			if s, ok := stored[field]; !ok || !sameJSONString(s, v) {
				same = false
			}
		}
		if same {
			continue
		}
		if !embeddingRemoteEngines[engine] {
			return nil, false, fmt.Errorf("%s is only stored by Open WebUI while RAG_EMBEDDING_ENGINE is \"ollama\", \"openai\" or \"azure_openai\", but the engine after this update is %q (\"\" is the local sentence-transformers engine), so the change would be silently dropped; select a remote engine in the same update or leave %s unchanged", form.key, engine, form.key)
		}
		encoded, err := json.Marshal(merged)
		if err != nil {
			return nil, false, err
		}
		body[form.key] = encoded
		changed = true
	}
	return body, changed, nil
}

// sameJSONValue reports whether a and b decode to the same JSON value (a
// missing value equals nothing).
func sameJSONValue(a, b json.RawMessage) bool {
	if len(a) == 0 || len(b) == 0 {
		return len(a) == len(b)
	}
	var x, y any
	if json.Unmarshal(a, &x) != nil || json.Unmarshal(b, &y) != nil {
		return false
	}
	return reflect.DeepEqual(x, y)
}

func isJSONString(raw json.RawMessage) bool {
	trimmed := bytes.TrimSpace(raw)
	return len(trimmed) > 0 && trimmed[0] == '"'
}

// sameJSONString reports whether a and b are JSON strings with the same
// decoded value (Go and Python escape characters such as & differently).
func sameJSONString(a, b json.RawMessage) bool {
	var x, y string
	if json.Unmarshal(a, &x) != nil || json.Unmarshal(b, &y) != nil {
		return false
	}
	return x == y
}

func isJSONNull(raw json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(raw), []byte("null"))
}

// isJSONIntOrNull reports whether Pydantic accepts raw for `int | None`: null,
// a whole JSON number, or a string holding one.
func isJSONIntOrNull(raw json.RawMessage) bool {
	if isJSONNull(raw) {
		return true
	}
	var n int64
	if json.Unmarshal(raw, &n) == nil {
		return true
	}
	var s string
	if json.Unmarshal(raw, &s) != nil {
		return false
	}
	_, err := strconv.ParseInt(s, 10, 64)
	return err == nil
}

// isJSONBoolOrNull reports whether raw is null or a JSON boolean.
func isJSONBoolOrNull(raw json.RawMessage) bool {
	if isJSONNull(raw) {
		return true
	}
	var b bool
	return json.Unmarshal(raw, &b) == nil
}
