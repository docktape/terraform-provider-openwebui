package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

func embeddingHCL(attrs string) string {
	return `
resource "openwebui_retrieval_embedding_config" "test" {
` + attrs + `
}
`
}

// stockEmbeddingJSON is a stock Open WebUI v0.11.0 GET /retrieval/embedding
// response with the local engine and one connection secret populated.
const stockEmbeddingJSON = `{
	"status": true,
	"RAG_EMBEDDING_ENGINE": "",
	"RAG_EMBEDDING_MODEL": "sentence-transformers/all-MiniLM-L6-v2",
	"RAG_EMBEDDING_BATCH_SIZE": 10,
	"ENABLE_ASYNC_EMBEDDING": true,
	"RAG_EMBEDDING_CONCURRENT_REQUESTS": 3,
	"openai_config": {"url": "https://api.openai.com/v1", "key": "sk-existing"},
	"ollama_config": {"url": "http://host.docker.internal:11434", "key": ""},
	"azure_openai_config": {"url": "", "key": "", "version": ""}
}`

// embeddingFakeBackend mimics Open WebUI v0.11.0's /retrieval/embedding pair.
// Validation follows EmbeddingModelUpdateForm (checked against the real
// Pydantic): RAG_EMBEDDING_ENGINE/RAG_EMBEDDING_MODEL are required strings,
// the nested url/key/version are required strings, the three other scalars
// take their defaults (1, true, 0) when omitted, and anything else is a 422.
// The handler then assigns the five scalars unconditionally (the model
// stripped) and, only while the new engine is ollama, openai or azure_openai,
// every nested object that was sent.
type embeddingFakeBackend struct {
	t     *testing.T
	mu    sync.Mutex
	state map[string]any
	posts []map[string]any
}

func newEmbeddingFakeBackend(t *testing.T, initial string) (*embeddingFakeBackend, *httptest.Server) {
	t.Helper()
	b := &embeddingFakeBackend{t: t}
	if err := json.Unmarshal([]byte(initial), &b.state); err != nil {
		t.Fatalf("fake backend state: %v", err)
	}
	return b, httptest.NewServer(b)
}

func (b *embeddingFakeBackend) reject(w http.ResponseWriter, loc string) {
	w.WriteHeader(http.StatusUnprocessableEntity)
	_, _ = w.Write([]byte(`{"detail": [{"loc": ["body", "` + loc + `"], "msg": "Input should be a valid value"}]}`))
}

func (b *embeddingFakeBackend) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b.mu.Lock()
	defer b.mu.Unlock()
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/api/v1/retrieval/embedding":
		_ = json.NewEncoder(w).Encode(b.state)
	case r.Method == http.MethodPost && r.URL.Path == "/api/v1/retrieval/embedding/update":
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		if err := json.Unmarshal(raw, &body); err != nil {
			b.t.Errorf("decode POST: %v", err)
		}
		b.posts = append(b.posts, body)
		for _, k := range []string{"RAG_EMBEDDING_ENGINE", "RAG_EMBEDDING_MODEL"} {
			if _, ok := body[k].(string); !ok {
				b.reject(w, k)
				return
			}
		}
		for _, form := range []struct {
			key    string
			fields []string
		}{{"openai_config", []string{"url", "key"}}, {"ollama_config", []string{"url", "key"}}, {"azure_openai_config", []string{"url", "key", "version"}}} {
			v, present := body[form.key]
			if !present || v == nil {
				continue
			}
			sub, ok := v.(map[string]any)
			if !ok {
				b.reject(w, form.key)
				return
			}
			for _, f := range form.fields {
				if _, ok := sub[f].(string); !ok {
					b.reject(w, form.key+"."+f)
					return
				}
			}
		}
		defaults := map[string]any{"RAG_EMBEDDING_BATCH_SIZE": float64(1), "ENABLE_ASYNC_EMBEDDING": true, "RAG_EMBEDDING_CONCURRENT_REQUESTS": float64(0)}
		for k, d := range defaults {
			v, present := body[k]
			if !present {
				v = d
			}
			b.state[k] = v
		}
		engine, _ := body["RAG_EMBEDDING_ENGINE"].(string)
		model, _ := body["RAG_EMBEDDING_MODEL"].(string)
		b.state["RAG_EMBEDDING_ENGINE"] = engine
		b.state["RAG_EMBEDDING_MODEL"] = strings.TrimSpace(model)
		if engine == "ollama" || engine == "openai" || engine == "azure_openai" {
			for _, k := range []string{"openai_config", "ollama_config", "azure_openai_config"} {
				if sub, ok := body[k].(map[string]any); ok {
					b.state[k] = sub
				}
			}
		}
		_ = json.NewEncoder(w).Encode(b.state)
	default:
		b.t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
	}
}

func (b *embeddingFakeBackend) snapshot() (map[string]any, []map[string]any) {
	b.mu.Lock()
	defer b.mu.Unlock()
	state, _ := json.Marshal(b.state)
	var copied map[string]any
	_ = json.Unmarshal(state, &copied)
	return copied, append([]map[string]any(nil), b.posts...)
}

func decodeStockEmbedding(t *testing.T, raw string) client.RetrievalEmbeddingConfig {
	t.Helper()
	var c client.RetrievalEmbeddingConfig
	if err := json.Unmarshal([]byte(raw), &c); err != nil {
		t.Fatalf("decode embedding config: %v", err)
	}
	return c
}

// fillEmbeddingModel sets every attribute to its typed unknown (as in a
// create plan) or null value, nested objects included.
func fillEmbeddingModel(unknown bool) retrievalEmbeddingConfigModel {
	if unknown {
		return retrievalEmbeddingConfigModel{
			ID:                             types.StringUnknown(),
			RAGEmbeddingEngine:             types.StringUnknown(),
			RAGEmbeddingModel:              types.StringUnknown(),
			RAGEmbeddingBatchSize:          types.Int64Unknown(),
			EnableAsyncEmbedding:           types.BoolUnknown(),
			RAGEmbeddingConcurrentRequests: types.Int64Unknown(),
			OpenAIConfig:                   types.ObjectUnknown(embeddingConnectionAttrTypes),
			OllamaConfig:                   types.ObjectUnknown(embeddingConnectionAttrTypes),
			AzureOpenAIConfig:              types.ObjectUnknown(embeddingAzureAttrTypes),
		}
	}
	return retrievalEmbeddingConfigModel{
		ID:                             types.StringNull(),
		RAGEmbeddingEngine:             types.StringNull(),
		RAGEmbeddingModel:              types.StringNull(),
		RAGEmbeddingBatchSize:          types.Int64Null(),
		EnableAsyncEmbedding:           types.BoolNull(),
		RAGEmbeddingConcurrentRequests: types.Int64Null(),
		OpenAIConfig:                   types.ObjectNull(embeddingConnectionAttrTypes),
		OllamaConfig:                   types.ObjectNull(embeddingConnectionAttrTypes),
		AzureOpenAIConfig:              types.ObjectNull(embeddingAzureAttrTypes),
	}
}

func connectionObject(t *testing.T, url, key types.String) types.Object {
	t.Helper()
	obj, diags := types.ObjectValue(embeddingConnectionAttrTypes, map[string]attr.Value{"url": url, "key": key})
	if diags.HasError() {
		t.Fatal(diags)
	}
	return obj
}

func marshalEmbeddingUpdate(t *testing.T, update client.RetrievalEmbeddingConfigUpdate) map[string]any {
	t.Helper()
	raw, err := json.Marshal(update)
	if err != nil {
		t.Fatalf("marshal update: %v", err)
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("decode update: %v", err)
	}
	return body
}

// TestRetrievalEmbeddingUpdateFromUnsetConfigIsEmpty: all-null and
// all-unknown configurations (nested objects included) change nothing.
func TestRetrievalEmbeddingUpdateFromUnsetConfigIsEmpty(t *testing.T) {
	for _, unknown := range []bool{true, false} {
		update, diags := retrievalEmbeddingConfigUpdateFromConfig(context.Background(), fillEmbeddingModel(unknown))
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if body := marshalEmbeddingUpdate(t, update); len(body) != 0 {
			t.Errorf("unknown=%v: nothing may be sent, got %v", unknown, body)
		}
	}
}

// TestRetrievalEmbeddingUpdateFromKnownValues: known values map to the literal
// wire keys (false, 0 and "" included); a nested object sends only its set
// fields (the client completes it), and a null or unknown inner field none.
func TestRetrievalEmbeddingUpdateFromKnownValues(t *testing.T) {
	m := fillEmbeddingModel(false)
	m.RAGEmbeddingEngine = types.StringValue("")
	m.RAGEmbeddingModel = types.StringValue("sentence-transformers/all-MiniLM-L6-v2")
	m.RAGEmbeddingBatchSize = types.Int64Value(1)
	m.EnableAsyncEmbedding = types.BoolValue(false)
	m.RAGEmbeddingConcurrentRequests = types.Int64Value(0)
	m.OpenAIConfig = connectionObject(t, types.StringValue("https://api.openai.com/v1"), types.StringNull())
	m.OllamaConfig = connectionObject(t, types.StringUnknown(), types.StringValue(""))
	azure, diags := types.ObjectValue(embeddingAzureAttrTypes, map[string]attr.Value{"url": types.StringValue("u"), "key": types.StringValue("k"), "version": types.StringValue("v")})
	if diags.HasError() {
		t.Fatal(diags)
	}
	m.AzureOpenAIConfig = azure

	update, diags := retrievalEmbeddingConfigUpdateFromConfig(context.Background(), m)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	want := map[string]any{
		"RAG_EMBEDDING_ENGINE":              "",
		"RAG_EMBEDDING_MODEL":               "sentence-transformers/all-MiniLM-L6-v2",
		"RAG_EMBEDDING_BATCH_SIZE":          float64(1),
		"ENABLE_ASYNC_EMBEDDING":            false,
		"RAG_EMBEDDING_CONCURRENT_REQUESTS": float64(0),
		"openai_config":                     map[string]any{"url": "https://api.openai.com/v1"},
		"ollama_config":                     map[string]any{"key": ""},
		"azure_openai_config":               map[string]any{"url": "u", "key": "k", "version": "v"},
	}
	if got := marshalEmbeddingUpdate(t, update); !reflect.DeepEqual(got, want) {
		t.Errorf("update:\n got %v\nwant %v", got, want)
	}
}

// TestRetrievalEmbeddingModelFromAPI: populated nested objects flatten with
// their strings ("" kept), absent ones and null inner fields become null.
func TestRetrievalEmbeddingModelFromAPI(t *testing.T) {
	ctx := context.Background()
	m, diags := retrievalEmbeddingConfigModelFromAPI(ctx, decodeStockEmbedding(t, stockEmbeddingJSON))
	if diags.HasError() {
		t.Fatal(diags)
	}
	if m.ID.ValueString() != "retrieval_embedding" || m.RAGEmbeddingEngine.ValueString() != "" || m.RAGEmbeddingEngine.IsNull() {
		t.Errorf("id/engine: %v %v", m.ID, m.RAGEmbeddingEngine)
	}
	if m.RAGEmbeddingBatchSize.ValueInt64() != 10 || m.RAGEmbeddingConcurrentRequests.ValueInt64() != 3 || !m.EnableAsyncEmbedding.ValueBool() {
		t.Errorf("scalars: %v %v %v", m.RAGEmbeddingBatchSize, m.RAGEmbeddingConcurrentRequests, m.EnableAsyncEmbedding)
	}
	var openai embeddingConnectionModel
	if d := m.OpenAIConfig.As(ctx, &openai, objectAsOptions); d.HasError() {
		t.Fatal(d)
	}
	if openai.URL.ValueString() != "https://api.openai.com/v1" || openai.Key.ValueString() != "sk-existing" {
		t.Errorf("openai_config: %v", m.OpenAIConfig)
	}
	var azure embeddingAzureModel
	if d := m.AzureOpenAIConfig.As(ctx, &azure, objectAsOptions); d.HasError() {
		t.Fatal(d)
	}
	if azure.Version.IsNull() || azure.Version.ValueString() != "" {
		t.Errorf(`"" must stay "", got %v`, azure.Version)
	}

	partial, diags := retrievalEmbeddingConfigModelFromAPI(ctx, decodeStockEmbedding(t, `{"RAG_EMBEDDING_ENGINE": "openai", "RAG_EMBEDDING_MODEL": "m", "openai_config": {"url": null, "key": "k"}}`))
	if diags.HasError() {
		t.Fatal(diags)
	}
	if !partial.OllamaConfig.IsNull() || !partial.AzureOpenAIConfig.IsNull() || !partial.RAGEmbeddingBatchSize.IsNull() || !partial.EnableAsyncEmbedding.IsNull() {
		t.Errorf("absent values must be null: %+v", partial)
	}
	if d := partial.OpenAIConfig.As(ctx, &openai, objectAsOptions); d.HasError() {
		t.Fatal(d)
	}
	if !openai.URL.IsNull() || openai.Key.ValueString() != "k" {
		t.Errorf("a null inner field must be null: %v", partial.OpenAIConfig)
	}
}

// embeddingRaw encodes a model as a raw Terraform value of the resource schema.
func embeddingRaw(t *testing.T, m retrievalEmbeddingConfigModel) (tftypes.Value, tfsdk.State) {
	t.Helper()
	ctx := context.Background()
	var resp resource.SchemaResponse
	NewRetrievalEmbeddingConfigResource().Schema(ctx, resource.SchemaRequest{}, &resp)
	st := tfsdk.State{Schema: resp.Schema, Raw: tftypes.NewValue(resp.Schema.Type().TerraformType(ctx), nil)}
	if diags := st.Set(ctx, &m); diags.HasError() {
		t.Fatalf("encode model: %v", diags)
	}
	return st.Raw, st
}

// TestRetrievalEmbeddingUpdateOmitsUnconfiguredAttributes drives the real
// Update with a configuration that sets only rag_embedding_batch_size and
// openai_config.url, while the plan (as Terraform builds it with
// UseStateForUnknown) carries prior-state values for everything else, two of
// which were changed outside Terraform since (the concurrency and the OpenAI
// key). The POST must carry the configured values and, for the required and
// defaulted scalars, the server's current values (never the stale plan), and
// openai_config completed with the server's key. The state keeps the planned
// values of the drifted attributes with value-free warnings, one of them on
// the nested path.
func TestRetrievalEmbeddingUpdateOmitsUnconfiguredAttributes(t *testing.T) {
	ctx := context.Background()
	server := strings.Replace(stockEmbeddingJSON, `"RAG_EMBEDDING_ENGINE": ""`, `"RAG_EMBEDDING_ENGINE": "openai"`, 1)
	backend, srv := newEmbeddingFakeBackend(t, server)
	defer srv.Close()
	apiClient, err := client.NewClient(srv.URL, "test-token", false)
	if err != nil {
		t.Fatal(err)
	}

	prior, diags := retrievalEmbeddingConfigModelFromAPI(ctx, decodeStockEmbedding(t, server))
	if diags.HasError() {
		t.Fatal(diags)
	}
	prior.RAGEmbeddingConcurrentRequests = types.Int64Value(8)                                                              // server: 3
	prior.OpenAIConfig = connectionObject(t, types.StringValue("https://old.example/v1"), types.StringValue("sk-previous")) // server key: sk-existing

	config := fillEmbeddingModel(false)
	config.RAGEmbeddingBatchSize = types.Int64Value(32)
	config.OpenAIConfig = connectionObject(t, types.StringValue("https://proxy.example/v1"), types.StringNull())
	plan := prior
	plan.RAGEmbeddingBatchSize = types.Int64Value(32)
	plan.OpenAIConfig = connectionObject(t, types.StringValue("https://proxy.example/v1"), types.StringValue("sk-previous"))

	configRaw, _ := embeddingRaw(t, config)
	planRaw, _ := embeddingRaw(t, plan)
	priorRaw, priorState := embeddingRaw(t, prior)
	req := resource.UpdateRequest{
		Config: tfsdk.Config{Schema: priorState.Schema, Raw: configRaw},
		Plan:   tfsdk.Plan{Schema: priorState.Schema, Raw: planRaw},
		State:  tfsdk.State{Schema: priorState.Schema, Raw: priorRaw},
	}
	resp := resource.UpdateResponse{State: tfsdk.State{Schema: priorState.Schema, Raw: planRaw}}
	(&retrievalEmbeddingConfigResource{client: apiClient}).Update(ctx, req, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Update failed: %v", resp.Diagnostics)
	}

	stored, posts := backend.snapshot()
	if len(posts) != 1 {
		t.Fatalf("expected one POST, got %d", len(posts))
	}
	want := map[string]any{
		"RAG_EMBEDDING_ENGINE":              "openai",
		"RAG_EMBEDDING_MODEL":               "sentence-transformers/all-MiniLM-L6-v2",
		"RAG_EMBEDDING_BATCH_SIZE":          float64(32),
		"ENABLE_ASYNC_EMBEDDING":            true,
		"RAG_EMBEDDING_CONCURRENT_REQUESTS": float64(3),
		"openai_config":                     map[string]any{"url": "https://proxy.example/v1", "key": "sk-existing"},
	}
	if !reflect.DeepEqual(posts[0], want) {
		t.Errorf("POST body:\n got %v\nwant %v", posts[0], want)
	}
	if stored["RAG_EMBEDDING_CONCURRENT_REQUESTS"] != float64(3) {
		t.Errorf("the out-of-band concurrency must survive, got %v", stored["RAG_EMBEDDING_CONCURRENT_REQUESTS"])
	}

	var state retrievalEmbeddingConfigModel
	resp.Diagnostics.Append(resp.State.Get(ctx, &state)...)
	var openai embeddingConnectionModel
	resp.Diagnostics.Append(state.OpenAIConfig.As(ctx, &openai, objectAsOptions)...)
	if state.RAGEmbeddingBatchSize.ValueInt64() != 32 || state.RAGEmbeddingConcurrentRequests.ValueInt64() != 8 ||
		openai.URL.ValueString() != "https://proxy.example/v1" || openai.Key.ValueString() != "sk-previous" {
		t.Errorf("unexpected state: batch=%v concurrent=%v openai=%v", state.RAGEmbeddingBatchSize, state.RAGEmbeddingConcurrentRequests, openai.URL)
	}
	if resp.Diagnostics.WarningsCount() != 2 {
		t.Errorf("expected 2 warnings, got %v", resp.Diagnostics)
	}
	var nestedWarning bool
	for _, d := range resp.Diagnostics {
		if strings.Contains(d.Detail(), "sk-") {
			t.Errorf("warning must not print values: %s", d.Detail())
		}
		if strings.Contains(d.Detail(), "openai_config.key is not set") {
			nestedWarning = true
		}
	}
	if !nestedWarning {
		t.Errorf("expected a warning naming openai_config.key, got %v", resp.Diagnostics)
	}
}

// TestRetrievalEmbeddingCreateWithoutAttributesSendsNothing: an empty
// configuration reads the settings and never POSTs (a POST reloads the
// embedding model).
func TestRetrievalEmbeddingCreateWithoutAttributesSendsNothing(t *testing.T) {
	backend, srv := newEmbeddingFakeBackend(t, stockEmbeddingJSON)
	defer srv.Close()
	apiClient, err := client.NewClient(srv.URL, "test-token", false)
	if err != nil {
		t.Fatal(err)
	}
	state, diags := applyRetrievalEmbeddingConfig(context.Background(), apiClient, fillEmbeddingModel(false), fillEmbeddingModel(true))
	if diags.HasError() {
		t.Fatalf("apply failed: %v", diags)
	}
	if _, posts := backend.snapshot(); len(posts) != 0 {
		t.Errorf("nothing may be POSTed, got %v", posts)
	}
	if state.RAGEmbeddingModel.ValueString() != "sentence-transformers/all-MiniLM-L6-v2" || state.OpenAIConfig.IsNull() {
		t.Errorf("state must come from the server: %v %v", state.RAGEmbeddingModel, state.OpenAIConfig)
	}
}

// TestRetrievalEmbeddingConfigThroughTerraform runs real terraform plans and
// applies against the fake backend: an empty configuration (nothing sent,
// nested objects read into state), a scalar change, a switch to the openai
// engine with a populated openai_config, a nested object with an unconfigured
// key (kept) and a url unknown at plan time, removal of the nested block (kept
// on the server and in state), and a change of a nested object the engine
// would drop because the engine goes back to the local one (refused before
// anything is written). Each apply is followed by an empty-plan check.
func TestRetrievalEmbeddingConfigThroughTerraform(t *testing.T) {
	backend, srv := newEmbeddingFakeBackend(t, stockEmbeddingJSON)
	defer srv.Close()
	t.Setenv("OPENWEBUI_ENDPOINT", srv.URL)
	t.Setenv("OPENWEBUI_TOKEN", "test-token")
	const name = "openwebui_retrieval_embedding_config.test"

	empty := embeddingHCL(``)
	batch := embeddingHCL(`
  rag_embedding_batch_size          = 16
  rag_embedding_concurrent_requests = 0
`)
	openai := embeddingHCL(`
  rag_embedding_engine = "openai"
  rag_embedding_model  = "text-embedding-3-small"
  openai_config = {
    url = "https://proxy.example/v1"
    key = "sk-configured"
  }
`)
	unknownURL := `
resource "terraform_data" "url" {
  input = "https://late.example/v1"
}
` + embeddingHCL(`
  rag_embedding_engine = "openai"
  rag_embedding_model  = "text-embedding-3-small"
  openai_config = {
    url = terraform_data.url.output
  }
`)
	dropped := `
resource "terraform_data" "url" {
  input = "https://late.example/v1"
}
` + embeddingHCL(`
  rag_embedding_engine = "openai"
  rag_embedding_model  = "text-embedding-3-small"
`)
	// ollama_config with the openai engine: v0.11.0 stores every block
	// while any remote engine is selected.
	otherBlock := `
resource "terraform_data" "url" {
  input = "https://late.example/v1"
}
` + embeddingHCL(`
  rag_embedding_engine = "openai"
  rag_embedding_model  = "text-embedding-3-small"
  ollama_config = {
    url = "http://ollama.example:11434"
  }
`)
	// Back to the local engine while changing a block: refused.
	ignored := `
resource "terraform_data" "url" {
  input = "https://late.example/v1"
}
` + embeddingHCL(`
  rag_embedding_engine = ""
  rag_embedding_model  = "sentence-transformers/all-MiniLM-L6-v2"
  ollama_config = {
    url = "http://other.example:11434"
  }
`)
	postCount := func(want int) tfresource.TestCheckFunc {
		return func(_ *terraform.State) error {
			if _, posts := backend.snapshot(); len(posts) != want {
				t.Errorf("expected %d POSTs so far, got %d", want, len(posts))
			}
			return nil
		}
	}

	tfresource.UnitTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []tfresource.TestStep{
			{
				Config: empty,
				Check: tfresource.ComposeTestCheckFunc(
					tfresource.TestCheckResourceAttr(name, "id", "retrieval_embedding"),
					tfresource.TestCheckResourceAttr(name, "rag_embedding_engine", ""),
					tfresource.TestCheckResourceAttr(name, "rag_embedding_batch_size", "10"),
					tfresource.TestCheckResourceAttr(name, "openai_config.url", "https://api.openai.com/v1"),
					tfresource.TestCheckResourceAttr(name, "openai_config.key", "sk-existing"),
					tfresource.TestCheckResourceAttr(name, "azure_openai_config.version", ""),
					postCount(0),
				),
			},
			{Config: empty, PlanOnly: true},
			{
				Config: batch,
				Check: tfresource.ComposeTestCheckFunc(
					tfresource.TestCheckResourceAttr(name, "rag_embedding_batch_size", "16"),
					tfresource.TestCheckResourceAttr(name, "rag_embedding_concurrent_requests", "0"),
					tfresource.TestCheckResourceAttr(name, "rag_embedding_model", "sentence-transformers/all-MiniLM-L6-v2"),
					postCount(1),
				),
			},
			{Config: batch, PlanOnly: true},
			{
				Config: openai,
				Check: tfresource.ComposeTestCheckFunc(
					tfresource.TestCheckResourceAttr(name, "rag_embedding_engine", "openai"),
					tfresource.TestCheckResourceAttr(name, "openai_config.url", "https://proxy.example/v1"),
					tfresource.TestCheckResourceAttr(name, "openai_config.key", "sk-configured"),
					// No longer configured: kept as stored.
					tfresource.TestCheckResourceAttr(name, "rag_embedding_batch_size", "16"),
					func(_ *terraform.State) error {
						stored, _ := backend.snapshot()
						if !reflect.DeepEqual(stored["openai_config"], map[string]any{"url": "https://proxy.example/v1", "key": "sk-configured"}) || stored["RAG_EMBEDDING_BATCH_SIZE"] != float64(16) {
							t.Errorf("unexpected server state: %v", stored)
						}
						return nil
					},
				),
			},
			{Config: openai, PlanOnly: true},
			{
				Config: unknownURL,
				Check: tfresource.ComposeTestCheckFunc(
					tfresource.TestCheckResourceAttr(name, "openai_config.url", "https://late.example/v1"),
					tfresource.TestCheckResourceAttr(name, "openai_config.key", "sk-configured"),
					func(_ *terraform.State) error {
						_, posts := backend.snapshot()
						last := posts[len(posts)-1]
						if !reflect.DeepEqual(last["openai_config"], map[string]any{"url": "https://late.example/v1", "key": "sk-configured"}) {
							t.Errorf("openai_config must be sent complete with the stored key, got %v", last["openai_config"])
						}
						if _, ok := last["ollama_config"]; ok {
							t.Errorf("an unconfigured ollama_config must not be sent")
						}
						return nil
					},
				),
			},
			{Config: unknownURL, PlanOnly: true},
			{
				// The block is removed: nothing changes, so nothing is sent.
				Config: dropped,
				Check: tfresource.ComposeTestCheckFunc(
					tfresource.TestCheckResourceAttr(name, "openai_config.url", "https://late.example/v1"),
					tfresource.TestCheckResourceAttr(name, "openai_config.key", "sk-configured"),
					postCount(3),
				),
			},
			{Config: dropped, PlanOnly: true},
			{
				Config: otherBlock,
				Check: tfresource.ComposeTestCheckFunc(
					tfresource.TestCheckResourceAttr(name, "ollama_config.url", "http://ollama.example:11434"),
					tfresource.TestCheckResourceAttr(name, "ollama_config.key", ""),
					tfresource.TestCheckResourceAttr(name, "openai_config.url", "https://late.example/v1"),
					postCount(4),
				),
			},
			{Config: otherBlock, PlanOnly: true},
			{
				Config:      ignored,
				ExpectError: regexp.MustCompile(`(?s)ollama_config\s+is\s+only\s+stored\s+by\s+Open\s+WebUI\s+while\s+RAG_EMBEDDING_ENGINE\s+is`),
			},
		},
	})
	stored, posts := backend.snapshot()
	if len(posts) != 4 {
		t.Errorf("the refused change must not be POSTed, got %d POSTs", len(posts))
	}
	if !reflect.DeepEqual(stored["ollama_config"], map[string]any{"url": "http://ollama.example:11434", "key": ""}) || stored["RAG_EMBEDDING_ENGINE"] != "openai" {
		t.Errorf("the refused apply must not change anything, got %v", stored)
	}
	for i, p := range posts {
		if _, ok := p["status"]; ok {
			t.Errorf("POST %d must not send status", i)
		}
	}
}

// TestRetrievalEmbeddingNullAndUnknownBlocksThroughTerraform: a connection
// object the server does not return is a null block in state (and an empty
// plan); a whole block unknown at plan time (from another resource) is
// applied and stored, completed with the server's current values for the
// fields it leaves out (here it sets them all, so the POST carries exactly
// those values); the plan after it is empty.
func TestRetrievalEmbeddingNullAndUnknownBlocksThroughTerraform(t *testing.T) {
	var server map[string]any
	if err := json.Unmarshal([]byte(stockEmbeddingJSON), &server); err != nil {
		t.Fatal(err)
	}
	delete(server, "azure_openai_config")
	raw, _ := json.Marshal(server)
	backend, srv := newEmbeddingFakeBackend(t, string(raw))
	defer srv.Close()
	t.Setenv("OPENWEBUI_ENDPOINT", srv.URL)
	t.Setenv("OPENWEBUI_TOKEN", "test-token")
	const name = "openwebui_retrieval_embedding_config.test"

	unknownBlock := `
resource "terraform_data" "azure" {
  input = {
    url     = "https://example.openai.azure.com"
    key     = "az-configured"
    version = "2024-02-01"
  }
}
` + embeddingHCL(`
  rag_embedding_engine = "azure_openai"
  rag_embedding_model  = "text-embedding-3-small"
  azure_openai_config  = terraform_data.azure.output
`)
	tfresource.UnitTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []tfresource.TestStep{
			{
				Config: embeddingHCL(``),
				Check: tfresource.ComposeTestCheckFunc(
					tfresource.TestCheckNoResourceAttr(name, "azure_openai_config"),
					tfresource.TestCheckNoResourceAttr(name, "azure_openai_config.url"),
					tfresource.TestCheckResourceAttr(name, "ollama_config.key", ""),
				),
			},
			{Config: embeddingHCL(``), PlanOnly: true},
			{
				Config: unknownBlock,
				Check: tfresource.ComposeTestCheckFunc(
					tfresource.TestCheckResourceAttr(name, "azure_openai_config.url", "https://example.openai.azure.com"),
					tfresource.TestCheckResourceAttr(name, "azure_openai_config.key", "az-configured"),
					tfresource.TestCheckResourceAttr(name, "azure_openai_config.version", "2024-02-01"),
					func(_ *terraform.State) error {
						stored, posts := backend.snapshot()
						want := map[string]any{"url": "https://example.openai.azure.com", "key": "az-configured", "version": "2024-02-01"}
						if len(posts) != 1 || !reflect.DeepEqual(posts[0]["azure_openai_config"], want) || !reflect.DeepEqual(stored["azure_openai_config"], want) {
							t.Errorf("azure_openai_config: posts %v, stored %v", posts, stored["azure_openai_config"])
						}
						return nil
					},
				),
			},
			{Config: unknownBlock, PlanOnly: true},
		},
	})
}

// TestRetrievalEmbeddingApplyResultChecks: a configured value the server did
// not store is a clear error, an unconfigured one never is.
func TestRetrievalEmbeddingApplyResultChecks(t *testing.T) {
	ctx := context.Background()
	fresh, _ := retrievalEmbeddingConfigModelFromAPI(ctx, decodeStockEmbedding(t, stockEmbeddingJSON))
	if diags := checkRetrievalEmbeddingApplyResult(fillEmbeddingModel(false), fresh); diags.HasError() {
		t.Errorf("an unset configuration must pass: %v", diags)
	}
	config := fillEmbeddingModel(false)
	config.RAGEmbeddingModel = types.StringValue("other-model")
	config.OpenAIConfig = connectionObject(t, types.StringNull(), types.StringValue("sk-not-stored"))
	diags := checkRetrievalEmbeddingApplyResult(config, fresh)
	if diags.ErrorsCount() != 2 {
		t.Fatalf("expected 2 errors, got %v", diags)
	}
	for _, d := range diags {
		if strings.Contains(d.Detail(), "sk-") || strings.Contains(d.Detail(), "other-model") {
			t.Errorf("errors must not print values: %s", d.Detail())
		}
	}
}

func TestRetrievalEmbeddingConfigValidators(t *testing.T) {
	for _, ok := range []string{"", "ollama", "openai", "azure_openai"} {
		planOnlyValid(t, embeddingHCL(`  rag_embedding_engine = "`+ok+`"`))
	}
	for _, bad := range []string{"sentence-transformers", "OpenAI", "azure"} {
		planOnlyRejected(t, embeddingHCL(`  rag_embedding_engine = "`+bad+`"`), `(?s)rag_embedding_engine.*value must be one of`)
	}
	planOnlyValid(t, embeddingHCL(`  rag_embedding_model = "BAAI/bge-m3"`))
	planOnlyValid(t, embeddingHCL(`  rag_embedding_model = ""`))
	for _, bad := range []string{" BAAI/bge-m3", "BAAI/bge-m3\\n"} {
		planOnlyRejected(t, embeddingHCL(`  rag_embedding_model = "`+bad+`"`), `(?s)rag_embedding_model.*start\s+or\s+end\s+with\s+whitespace`)
	}
	planOnlyValid(t, embeddingHCL(`  rag_embedding_batch_size = 1`))
	planOnlyRejected(t, embeddingHCL(`  rag_embedding_batch_size = 0`), `(?s)rag_embedding_batch_size.*at\s+least\s+1`)
	planOnlyValid(t, embeddingHCL(`  rag_embedding_concurrent_requests = 0`))
	planOnlyRejected(t, embeddingHCL(`  rag_embedding_concurrent_requests = -1`), `(?s)rag_embedding_concurrent_requests.*at\s+least\s+0`)
}

// TestRetrievalEmbeddingConfigSchemaShape pins the attributes (id plus the
// five scalars and three nested objects of EmbeddingModelUpdateForm), the
// Description/MarkdownDescription rule on every attribute including the
// nested ones, the Optional+Computed shape and that exactly the three nested
// `key` attributes are Sensitive.
func TestRetrievalEmbeddingConfigSchemaShape(t *testing.T) {
	var resp resource.SchemaResponse
	NewRetrievalEmbeddingConfigResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)
	want := []string{"azure_openai_config", "enable_async_embedding", "id", "ollama_config", "openai_config",
		"rag_embedding_batch_size", "rag_embedding_concurrent_requests", "rag_embedding_engine", "rag_embedding_model"}
	got := make([]string, 0, len(resp.Schema.Attributes))
	for name := range resp.Schema.Attributes {
		got = append(got, name)
	}
	if strings.Join(sortedStrings(got), ",") != strings.Join(want, ",") {
		t.Fatalf("attributes: got %v, want %v", sortedStrings(got), want)
	}
	nestedFields := map[string][]string{
		"openai_config":       {"key", "url"},
		"ollama_config":       {"key", "url"},
		"azure_openai_config": {"key", "url", "version"},
	}
	sensitive := 0
	check := func(name string, a schema.Attribute, isID bool) {
		if a.GetDescription() == "" || a.GetMarkdownDescription() == "" {
			t.Errorf("attribute %q needs both Description and MarkdownDescription", name)
		}
		if !isID && (!a.IsOptional() || !a.IsComputed()) {
			t.Errorf("attribute %q must be Optional+Computed", name)
		}
		if a.IsSensitive() {
			sensitive++
			if !strings.HasSuffix(name, ".key") {
				t.Errorf("attribute %q must not be Sensitive", name)
			}
		} else if strings.HasSuffix(name, ".key") {
			t.Errorf("attribute %q must be Sensitive", name)
		}
	}
	for name, a := range resp.Schema.Attributes {
		check(name, a, name == "id")
		fields, nested := nestedFields[name]
		if !nested {
			continue
		}
		obj, ok := a.(schema.SingleNestedAttribute)
		if !ok {
			t.Fatalf("attribute %q must be a single nested attribute", name)
		}
		inner := make([]string, 0, len(obj.Attributes))
		for innerName, innerAttr := range obj.Attributes {
			inner = append(inner, innerName)
			check(name+"."+innerName, innerAttr, false)
		}
		if !reflect.DeepEqual(sortedStrings(inner), fields) {
			t.Errorf("%s fields: got %v, want %v", name, sortedStrings(inner), fields)
		}
	}
	if sensitive != 3 {
		t.Errorf("expected exactly 3 Sensitive attributes, got %d", sensitive)
	}
	// Every attribute exists unchanged since Open WebUI v0.9.6 (the
	// provider's floor), so none carries an "added in" compatibility note.
	if strings.Contains(resp.Schema.MarkdownDescription, "added in") {
		t.Errorf("unexpected compatibility note in the resource description")
	}
	for name, a := range resp.Schema.Attributes {
		if strings.Contains(a.GetMarkdownDescription(), "added in") {
			t.Errorf("unexpected compatibility note on %q", name)
		}
	}
}
