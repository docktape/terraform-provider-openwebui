package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

// stockEmbeddingConfig is a stock Open WebUI v0.11.0 GET /retrieval/embedding
// response (get_embedding_config), with the local sentence-transformers
// engine and two connection secrets populated.
const stockEmbeddingConfig = `{
	"status": true,
	"RAG_EMBEDDING_ENGINE": "",
	"RAG_EMBEDDING_MODEL": "sentence-transformers/all-MiniLM-L6-v2",
	"RAG_EMBEDDING_BATCH_SIZE": 10,
	"ENABLE_ASYNC_EMBEDDING": true,
	"RAG_EMBEDDING_CONCURRENT_REQUESTS": 3,
	"openai_config": {"url": "https://api.openai.com/v1", "key": "sk-existing"},
	"ollama_config": {"url": "http://host.docker.internal:11434", "key": ""},
	"azure_openai_config": {"url": "", "key": "az-existing", "version": ""}
}`

// embeddingFakeServer serves GET /retrieval/embedding with the given body and
// records every POST /retrieval/embedding/update body.
func embeddingFakeServer(t *testing.T, get string, posts *[]map[string]any) (*Client, func()) {
	t.Helper()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/retrieval/embedding":
			_, _ = w.Write([]byte(get))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/retrieval/embedding/update":
			raw, _ := io.ReadAll(r.Body)
			var body map[string]any
			if err := json.Unmarshal(raw, &body); err != nil {
				t.Errorf("decode POST body: %v", err)
			}
			*posts = append(*posts, body)
			_, _ = w.Write([]byte(get))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	})
	c, server := newConfigTestClient(handler)
	return c, server.Close
}

func TestGetRetrievalEmbeddingConfig(t *testing.T) {
	var posts []map[string]any
	c, done := embeddingFakeServer(t, stockEmbeddingConfig, &posts)
	defer done()

	got, err := c.GetRetrievalEmbeddingConfig(context.Background())
	if err != nil {
		t.Fatalf("GetRetrievalEmbeddingConfig failed: %v", err)
	}
	if got.RAGEmbeddingEngine == nil || *got.RAGEmbeddingEngine != "" {
		t.Errorf("engine: %v", got.RAGEmbeddingEngine)
	}
	if got.RAGEmbeddingModel == nil || *got.RAGEmbeddingModel != "sentence-transformers/all-MiniLM-L6-v2" {
		t.Errorf("model: %v", got.RAGEmbeddingModel)
	}
	if got.RAGEmbeddingBatchSize == nil || *got.RAGEmbeddingBatchSize != 10 || got.RAGEmbeddingConcurrentRequests == nil || *got.RAGEmbeddingConcurrentRequests != 3 {
		t.Errorf("ints: %v %v", got.RAGEmbeddingBatchSize, got.RAGEmbeddingConcurrentRequests)
	}
	if got.EnableAsyncEmbedding == nil || !*got.EnableAsyncEmbedding {
		t.Errorf("async: %v", got.EnableAsyncEmbedding)
	}
	if got.OpenAIConfig == nil || *got.OpenAIConfig.URL != "https://api.openai.com/v1" || *got.OpenAIConfig.Key != "sk-existing" {
		t.Errorf("openai_config: %+v", got.OpenAIConfig)
	}
	if got.OllamaConfig == nil || *got.OllamaConfig.URL != "http://host.docker.internal:11434" || *got.OllamaConfig.Key != "" {
		t.Errorf("ollama_config: %+v", got.OllamaConfig)
	}
	if got.AzureOpenAIConfig == nil || *got.AzureOpenAIConfig.Key != "az-existing" || *got.AzureOpenAIConfig.Version != "" {
		t.Errorf("azure_openai_config: %+v", got.AzureOpenAIConfig)
	}
	if len(posts) != 0 {
		t.Errorf("a read must not POST")
	}
}

// TestGetRetrievalEmbeddingConfigLenient covers the shapes the integers can
// take ("" and numeric strings from env-style values, null), and absent or
// null nested objects and nested fields.
func TestGetRetrievalEmbeddingConfigLenient(t *testing.T) {
	var posts []map[string]any
	c, done := embeddingFakeServer(t, `{
		"RAG_EMBEDDING_ENGINE": "openai", "RAG_EMBEDDING_MODEL": "text-embedding-3-small",
		"RAG_EMBEDDING_BATCH_SIZE": "", "RAG_EMBEDDING_CONCURRENT_REQUESTS": "4",
		"ENABLE_ASYNC_EMBEDDING": null,
		"openai_config": {"url": null, "key": "sk"},
		"ollama_config": null
	}`, &posts)
	defer done()

	got, err := c.GetRetrievalEmbeddingConfig(context.Background())
	if err != nil {
		t.Fatalf("GetRetrievalEmbeddingConfig failed: %v", err)
	}
	if got.RAGEmbeddingBatchSize != nil {
		t.Errorf(`"" must decode to nil, got %v`, *got.RAGEmbeddingBatchSize)
	}
	if got.RAGEmbeddingConcurrentRequests == nil || *got.RAGEmbeddingConcurrentRequests != 4 {
		t.Errorf(`"4" must decode to 4, got %v`, got.RAGEmbeddingConcurrentRequests)
	}
	if got.EnableAsyncEmbedding != nil {
		t.Errorf("null bool must decode to nil")
	}
	if got.OpenAIConfig == nil || got.OpenAIConfig.URL != nil || *got.OpenAIConfig.Key != "sk" {
		t.Errorf("openai_config: %+v", got.OpenAIConfig)
	}
	if got.OllamaConfig != nil || got.AzureOpenAIConfig != nil {
		t.Errorf("null and absent nested objects must decode to nil: %+v %+v", got.OllamaConfig, got.AzureOpenAIConfig)
	}

	c2, done2 := embeddingFakeServer(t, `{"RAG_EMBEDDING_ENGINE": "", "RAG_EMBEDDING_MODEL": "m", "RAG_EMBEDDING_BATCH_SIZE": "ten"}`, &posts)
	defer done2()
	if _, err := c2.GetRetrievalEmbeddingConfig(context.Background()); err == nil || !strings.Contains(err.Error(), "RAG_EMBEDDING_BATCH_SIZE") {
		t.Errorf("a non-numeric batch size must fail naming the key, got %v", err)
	}
}

// TestUpdateRetrievalEmbeddingConfigMergesScalars: the handler requires
// RAG_EMBEDDING_ENGINE and RAG_EMBEDDING_MODEL and resets the three other
// scalars to their Pydantic defaults when they are omitted, so an update of
// one scalar must send all five, the others as currently stored. Unchanged
// nested objects and `status` are not sent.
func TestUpdateRetrievalEmbeddingConfigMergesScalars(t *testing.T) {
	var posts []map[string]any
	c, done := embeddingFakeServer(t, stockEmbeddingConfig, &posts)
	defer done()

	size := int64(16)
	if err := c.UpdateRetrievalEmbeddingConfig(context.Background(), RetrievalEmbeddingConfigUpdate{RAGEmbeddingBatchSize: &size}); err != nil {
		t.Fatalf("UpdateRetrievalEmbeddingConfig failed: %v", err)
	}
	if len(posts) != 1 {
		t.Fatalf("expected one POST, got %d", len(posts))
	}
	want := map[string]any{
		"RAG_EMBEDDING_ENGINE":              "",
		"RAG_EMBEDDING_MODEL":               "sentence-transformers/all-MiniLM-L6-v2",
		"RAG_EMBEDDING_BATCH_SIZE":          float64(16),
		"ENABLE_ASYNC_EMBEDDING":            true,
		"RAG_EMBEDDING_CONCURRENT_REQUESTS": float64(3),
	}
	if !reflect.DeepEqual(posts[0], want) {
		t.Errorf("POST body:\n got %v\nwant %v", posts[0], want)
	}
}

// TestUpdateRetrievalEmbeddingConfigNestedForEngine: a changed nested object
// is sent complete (url, key[, version] are required strings in the form),
// unset nested fields as currently stored, together with the engine that
// makes Open WebUI store it.
func TestUpdateRetrievalEmbeddingConfigNestedForEngine(t *testing.T) {
	var posts []map[string]any
	c, done := embeddingFakeServer(t, stockEmbeddingConfig, &posts)
	defer done()

	engine, model, url, version := "azure_openai", "text-embedding-3-small", "https://example.openai.azure.com", "2024-02-01"
	update := RetrievalEmbeddingConfigUpdate{
		RAGEmbeddingEngine: &engine,
		RAGEmbeddingModel:  &model,
		AzureOpenAIConfig:  &EmbeddingAzureOpenAIConfig{URL: &url, Version: &version},
	}
	if err := c.UpdateRetrievalEmbeddingConfig(context.Background(), update); err != nil {
		t.Fatalf("UpdateRetrievalEmbeddingConfig failed: %v", err)
	}
	if len(posts) != 1 {
		t.Fatalf("expected one POST, got %d", len(posts))
	}
	body := posts[0]
	wantAzure := map[string]any{"url": url, "key": "az-existing", "version": version}
	if !reflect.DeepEqual(body["azure_openai_config"], wantAzure) {
		t.Errorf("azure_openai_config: got %v, want %v", body["azure_openai_config"], wantAzure)
	}
	if _, ok := body["openai_config"]; ok {
		t.Errorf("an unchanged openai_config must not be sent")
	}
	if _, ok := body["ollama_config"]; ok {
		t.Errorf("an unchanged ollama_config must not be sent")
	}
	if body["RAG_EMBEDDING_ENGINE"] != "azure_openai" || body["RAG_EMBEDDING_MODEL"] != model || body["RAG_EMBEDDING_BATCH_SIZE"] != float64(10) {
		t.Errorf("scalars: %v", body)
	}
	if _, ok := body["status"]; ok {
		t.Errorf("status must not be sent")
	}
}

// TestUpdateRetrievalEmbeddingConfigRejectsIgnoredNested: Open WebUI v0.9.6
// to v0.11.0 store a sent nested connection object only while the engine
// after the update is ollama, openai or azure_openai, and drop it silently
// for the local engine. A change that would be dropped is refused before
// anything is sent, and the error never prints the values.
func TestUpdateRetrievalEmbeddingConfigRejectsIgnoredNested(t *testing.T) {
	for name, update := range map[string]RetrievalEmbeddingConfigUpdate{
		"local engine stays": {OpenAIConfig: &EmbeddingConnectionConfig{Key: strPtr("sk-new-secret")}},
		"switch to local":    {RAGEmbeddingEngine: strPtr(""), OllamaConfig: &EmbeddingConnectionConfig{URL: strPtr("http://ollama:11434"), Key: strPtr("sk-new-secret")}},
	} {
		t.Run(name, func(t *testing.T) {
			var posts []map[string]any
			server := stockEmbeddingConfig
			if update.RAGEmbeddingEngine != nil {
				server = strings.Replace(server, `"RAG_EMBEDDING_ENGINE": ""`, `"RAG_EMBEDDING_ENGINE": "openai"`, 1)
			}
			c, done := embeddingFakeServer(t, server, &posts)
			defer done()
			err := c.UpdateRetrievalEmbeddingConfig(context.Background(), update)
			if err == nil || !strings.Contains(err.Error(), "only stored by Open WebUI while RAG_EMBEDDING_ENGINE is") {
				t.Fatalf("expected a refusal, got %v", err)
			}
			if strings.Contains(err.Error(), "sk-") || strings.Contains(err.Error(), "ollama:11434") {
				t.Errorf("the error must not print values: %v", err)
			}
			if len(posts) != 0 {
				t.Errorf("nothing may be POSTed, got %v", posts)
			}
		})
	}
}

// TestUpdateRetrievalEmbeddingConfigNestedForAnyRemoteEngine: v0.11.0 stores
// every sent nested object while any remote engine is selected, so an
// ollama_config change with the openai engine is sent (complete).
func TestUpdateRetrievalEmbeddingConfigNestedForAnyRemoteEngine(t *testing.T) {
	var posts []map[string]any
	c, done := embeddingFakeServer(t, strings.Replace(stockEmbeddingConfig, `"RAG_EMBEDDING_ENGINE": ""`, `"RAG_EMBEDDING_ENGINE": "openai"`, 1), &posts)
	defer done()
	if err := c.UpdateRetrievalEmbeddingConfig(context.Background(), RetrievalEmbeddingConfigUpdate{OllamaConfig: &EmbeddingConnectionConfig{Key: strPtr("ollama-key")}}); err != nil {
		t.Fatalf("UpdateRetrievalEmbeddingConfig failed: %v", err)
	}
	if len(posts) != 1 {
		t.Fatalf("expected one POST, got %d", len(posts))
	}
	want := map[string]any{"url": "http://host.docker.internal:11434", "key": "ollama-key"}
	if !reflect.DeepEqual(posts[0]["ollama_config"], want) {
		t.Errorf("ollama_config: got %v, want %v", posts[0]["ollama_config"], want)
	}
	if posts[0]["RAG_EMBEDDING_ENGINE"] != "openai" {
		t.Errorf("engine: %v", posts[0]["RAG_EMBEDDING_ENGINE"])
	}
}

// TestUpdateRetrievalEmbeddingConfigNoChangeSendsNothing: every POST reloads
// the embedding model, so an update whose fields all equal the stored values
// (an unchanged nested object included, for any engine) sends nothing.
func TestUpdateRetrievalEmbeddingConfigNoChangeSendsNothing(t *testing.T) {
	var posts []map[string]any
	c, done := embeddingFakeServer(t, stockEmbeddingConfig, &posts)
	defer done()
	update := RetrievalEmbeddingConfigUpdate{
		RAGEmbeddingEngine:             strPtr(""),
		RAGEmbeddingModel:              strPtr("sentence-transformers/all-MiniLM-L6-v2"),
		RAGEmbeddingBatchSize:          int64Ptr(10),
		EnableAsyncEmbedding:           boolPtr(true),
		RAGEmbeddingConcurrentRequests: int64Ptr(3),
		OpenAIConfig:                   &EmbeddingConnectionConfig{URL: strPtr("https://api.openai.com/v1"), Key: strPtr("sk-existing")},
	}
	if err := c.UpdateRetrievalEmbeddingConfig(context.Background(), update); err != nil {
		t.Fatalf("UpdateRetrievalEmbeddingConfig failed: %v", err)
	}
	if len(posts) != 0 {
		t.Errorf("an unchanged update must not POST, got %v", posts)
	}
}

// TestUpdateRetrievalEmbeddingConfigUnchangedNestedIsNotSent: a nested object
// equal to the stored one is a no-op, so it is accepted for any engine and
// left out of the request.
func TestUpdateRetrievalEmbeddingConfigUnchangedNestedIsNotSent(t *testing.T) {
	var posts []map[string]any
	c, done := embeddingFakeServer(t, stockEmbeddingConfig, &posts)
	defer done()
	size := int64(11)
	update := RetrievalEmbeddingConfigUpdate{
		RAGEmbeddingBatchSize: &size,
		OpenAIConfig:          &EmbeddingConnectionConfig{URL: strPtr("https://api.openai.com/v1"), Key: strPtr("sk-existing")},
	}
	if err := c.UpdateRetrievalEmbeddingConfig(context.Background(), update); err != nil {
		t.Fatalf("UpdateRetrievalEmbeddingConfig failed: %v", err)
	}
	if len(posts) != 1 {
		t.Fatalf("expected one POST, got %d", len(posts))
	}
	if _, ok := posts[0]["openai_config"]; ok {
		t.Errorf("an unchanged openai_config must not be sent")
	}

	// Python's json does not escape & (Go's encoder writes &): the
	// values are compared decoded.
	var more []map[string]any
	c2, done2 := embeddingFakeServer(t, strings.Replace(stockEmbeddingConfig, `"https://api.openai.com/v1"`, `"https://proxy.example/v1?a=1&b=2"`, 1), &more)
	defer done2()
	update.OpenAIConfig.URL = strPtr("https://proxy.example/v1?a=1&b=2")
	if err := c2.UpdateRetrievalEmbeddingConfig(context.Background(), update); err != nil {
		t.Fatalf("UpdateRetrievalEmbeddingConfig failed: %v", err)
	}
	if len(more) != 1 {
		t.Fatalf("expected one POST, got %d", len(more))
	}
	if _, ok := more[0]["openai_config"]; ok {
		t.Errorf("an unchanged openai_config with & must not be sent")
	}
}

// TestUpdateRetrievalEmbeddingConfigEmptySendsNothing: nothing to change
// means no request at all (every POST reloads the embedding model).
func TestUpdateRetrievalEmbeddingConfigEmptySendsNothing(t *testing.T) {
	handler := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
	})
	c, server := newConfigTestClient(handler)
	defer server.Close()
	if err := c.UpdateRetrievalEmbeddingConfig(context.Background(), RetrievalEmbeddingConfigUpdate{}); err != nil {
		t.Fatalf("UpdateRetrievalEmbeddingConfig failed: %v", err)
	}
}

// TestUpdateRetrievalEmbeddingConfigCurrentShapes: unset nullable scalars are
// echoed as stored (null included, Pydantic accepts it), a value Pydantic
// would reject ("" for an int) is left out so its default applies, a null
// nested field is sent as "" (the form's fields are required strings), and
// the engine and model must be readable when not set.
func TestUpdateRetrievalEmbeddingConfigCurrentShapes(t *testing.T) {
	var posts []map[string]any
	c, done := embeddingFakeServer(t, `{
		"RAG_EMBEDDING_ENGINE": "openai", "RAG_EMBEDDING_MODEL": "text-embedding-3-small",
		"RAG_EMBEDDING_BATCH_SIZE": "", "RAG_EMBEDDING_CONCURRENT_REQUESTS": "4",
		"ENABLE_ASYNC_EMBEDDING": null,
		"openai_config": {"url": null, "key": "sk-existing"}
	}`, &posts)
	defer done()
	key := "sk-new"
	if err := c.UpdateRetrievalEmbeddingConfig(context.Background(), RetrievalEmbeddingConfigUpdate{OpenAIConfig: &EmbeddingConnectionConfig{Key: &key}}); err != nil {
		t.Fatalf("UpdateRetrievalEmbeddingConfig failed: %v", err)
	}
	want := map[string]any{
		"RAG_EMBEDDING_ENGINE":              "openai",
		"RAG_EMBEDDING_MODEL":               "text-embedding-3-small",
		"RAG_EMBEDDING_CONCURRENT_REQUESTS": "4",
		"ENABLE_ASYNC_EMBEDDING":            nil,
		"openai_config":                     map[string]any{"url": "", "key": "sk-new"},
	}
	if len(posts) != 1 || !reflect.DeepEqual(posts[0], want) {
		t.Fatalf("POST body:\n got %v\nwant %v", posts, want)
	}

	var none []map[string]any
	c2, done2 := embeddingFakeServer(t, `{"RAG_EMBEDDING_MODEL": "m"}`, &none)
	defer done2()
	size := int64(2)
	err := c2.UpdateRetrievalEmbeddingConfig(context.Background(), RetrievalEmbeddingConfigUpdate{RAGEmbeddingBatchSize: &size})
	if err == nil || !strings.Contains(err.Error(), "RAG_EMBEDDING_ENGINE") {
		t.Errorf("a missing engine must be refused, got %v", err)
	}
	if len(none) != 0 {
		t.Errorf("nothing may be POSTed, got %v", none)
	}
}

// TestRetrievalEmbeddingConfigUpdateWireKeys pins the literal wire keys of a
// fully populated update.
func TestRetrievalEmbeddingConfigUpdateWireKeys(t *testing.T) {
	b, i, s := true, int64(1), "x"
	raw, err := json.Marshal(RetrievalEmbeddingConfigUpdate{
		RAGEmbeddingEngine: &s, RAGEmbeddingModel: &s, RAGEmbeddingBatchSize: &i, EnableAsyncEmbedding: &b,
		RAGEmbeddingConcurrentRequests: &i,
		OpenAIConfig:                   &EmbeddingConnectionConfig{URL: &s, Key: &s},
		OllamaConfig:                   &EmbeddingConnectionConfig{URL: &s, Key: &s},
		AzureOpenAIConfig:              &EmbeddingAzureOpenAIConfig{URL: &s, Key: &s, Version: &s},
	})
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	want := []string{"ENABLE_ASYNC_EMBEDDING", "RAG_EMBEDDING_BATCH_SIZE", "RAG_EMBEDDING_CONCURRENT_REQUESTS", "RAG_EMBEDDING_ENGINE", "RAG_EMBEDDING_MODEL", "azure_openai_config", "ollama_config", "openai_config"}
	if got := sortedKeys(body); !reflect.DeepEqual(got, want) {
		t.Errorf("wire keys: got %v, want %v", got, want)
	}
	azure, _ := body["azure_openai_config"].(map[string]any)
	if got := sortedKeys(azure); !reflect.DeepEqual(got, []string{"key", "url", "version"}) {
		t.Errorf("azure keys: %v", got)
	}
	openai, _ := body["openai_config"].(map[string]any)
	if got := sortedKeys(openai); !reflect.DeepEqual(got, []string{"key", "url"}) {
		t.Errorf("openai keys: %v", got)
	}
}

func strPtr(s string) *string { return &s }

func int64Ptr(n int64) *int64 { return &n }

func boolPtr(b bool) *bool { return &b }
