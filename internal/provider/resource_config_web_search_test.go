package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

func webSearchHCL(attrs string) string {
	return `
resource "openwebui_web_search_config" "test" {
` + attrs + `
}
`
}

// stockWebJSON is the `web` object of a stock Open WebUI v0.11.0 GET
// /retrieval/config response (all 74 keys), with one secret populated and a
// non-null YOUTUBE_LOADER_TRANSLATION, which the resource does not model.
const stockWebJSON = `{
	"ENABLE_WEB_SEARCH": true, "ENABLE_WEB_SEARCH_CONFIRMATION": false,
	"WEB_SEARCH_CONFIRMATION_CONTENT": "Your query will be sent to the configured web search provider.",
	"WEB_SEARCH_ENGINE": "tavily", "WEB_SEARCH_TRUST_ENV": true, "WEB_SEARCH_RESULT_COUNT": 3,
	"WEB_SEARCH_CONCURRENT_REQUESTS": 0, "WEB_FETCH_MAX_CONTENT_LENGTH": null, "WEB_LOADER_CONCURRENT_REQUESTS": 10,
	"WEB_SEARCH_DOMAIN_FILTER_LIST": [], "BYPASS_WEB_SEARCH_EMBEDDING_AND_RETRIEVAL": false,
	"BYPASS_WEB_SEARCH_WEB_LOADER": false, "OLLAMA_CLOUD_WEB_SEARCH_API_KEY": "",
	"SEARXNG_QUERY_URL": "", "SEARXNG_LANGUAGE": "all", "OPENSERP_BASE_URL": "http://localhost:7000",
	"YACY_QUERY_URL": "", "YACY_USERNAME": "", "YACY_PASSWORD": "",
	"GOOGLE_PSE_API_KEY": "", "GOOGLE_PSE_ENGINE_ID": "", "BRAVE_SEARCH_API_KEY": "", "BRAVE_SEARCH_CONTEXT_TOKENS": 8192,
	"KAGI_SEARCH_API_KEY": "", "MOJEEK_SEARCH_API_KEY": "", "BOCHA_SEARCH_API_KEY": "",
	"SERPSTACK_API_KEY": "", "SERPSTACK_HTTPS": true, "SERPER_API_KEY": "", "SERPHOUSE_API_KEY": "",
	"SERPHOUSE_DOMAIN": "google.com", "SERPLY_API_KEY": "", "DDGS_BACKEND": "auto", "TAVILY_API_KEY": "tvly-existing",
	"SEARCHAPI_API_KEY": "", "SEARCHAPI_ENGINE": "", "SERPAPI_API_KEY": "", "SERPAPI_ENGINE": "",
	"JINA_API_KEY": "", "JINA_API_BASE_URL": "", "BING_SEARCH_V7_ENDPOINT": "https://api.bing.microsoft.com/v7.0/search",
	"BING_SEARCH_V7_SUBSCRIPTION_KEY": "", "EXA_API_KEY": "", "PERPLEXITY_API_KEY": "", "PERPLEXITY_MODEL": "sonar",
	"PERPLEXITY_SEARCH_CONTEXT_USAGE": "medium", "PERPLEXITY_SEARCH_API_URL": "https://api.perplexity.ai/search",
	"MICROSOFT_WEB_IQ_API_BASE_URL": "https://api.microsoft.ai/v3", "MICROSOFT_WEB_IQ_API_KEY": "",
	"MICROSOFT_WEB_IQ_LANGUAGE": "en", "SOUGOU_API_SID": "", "SOUGOU_API_SK": "",
	"WEB_LOADER_ENGINE": "playwright", "WEB_LOADER_TIMEOUT": "", "ENABLE_WEB_LOADER_SSL_VERIFICATION": true,
	"PLAYWRIGHT_WS_URL": "ws://playwright:3000", "PLAYWRIGHT_TIMEOUT": 10000,
	"FIRECRAWL_API_KEY": "", "FIRECRAWL_API_BASE_URL": "https://api.firecrawl.dev", "FIRECRAWL_TIMEOUT": "",
	"TAVILY_EXTRACT_DEPTH": "basic", "EXTERNAL_WEB_SEARCH_URL": "", "EXTERNAL_WEB_SEARCH_API_KEY": "",
	"EXTERNAL_WEB_LOADER_URL": "", "EXTERNAL_WEB_LOADER_API_KEY": "",
	"YOUTUBE_LOADER_LANGUAGE": ["en"], "YOUTUBE_LOADER_PROXY_URL": "", "YOUTUBE_LOADER_TRANSLATION": "de",
	"YANDEX_WEB_SEARCH_URL": "", "YANDEX_WEB_SEARCH_API_KEY": "", "YANDEX_WEB_SEARCH_CONFIG": "",
	"YOUCOM_API_KEY": "", "LINKUP_API_KEY": "", "LINKUP_SEARCH_PARAMS": {"depth": "standard"}
}`

// webSearchFakeBackend mimics Open WebUI v0.11.0's /retrieval/config pair:
// GET returns the stored object; POST merges top-level fields per field (only
// non-null ones) and, when `web` is present, REPLACES the web object the way
// update_rag_config does: every WebConfig field (the keys of the initial web
// object) is taken from the posted object, a missing one becomes its Pydantic
// default (null, or [] for WEB_SEARCH_DOMAIN_FILTER_LIST),
// BRAVE_SEARCH_CONTEXT_TOKENS is kept when null, and unknown keys are ignored.
// Like Pydantic, it rejects the whole request with HTTP 422 when one of the two
// non-Optional WebConfig fields (webSearchNonNullableKeys) is an explicit null.
type webSearchFakeBackend struct {
	t     *testing.T
	mu    sync.Mutex
	state map[string]any
	posts []map[string]any
	gets  int
}

func newWebSearchFakeBackend(t *testing.T, web string) (*webSearchFakeBackend, *httptest.Server) {
	t.Helper()
	b := &webSearchFakeBackend{t: t}
	if err := json.Unmarshal([]byte(`{"status": true, "TOP_K": 3, "RAG_TEMPLATE": "### Task:", "web": `+web+`}`), &b.state); err != nil {
		t.Fatalf("fake backend state: %v", err)
	}
	return b, httptest.NewServer(b)
}

func (b *webSearchFakeBackend) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b.mu.Lock()
	defer b.mu.Unlock()
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/api/v1/retrieval/config":
		b.gets++
		_ = json.NewEncoder(w).Encode(b.state)
	case r.Method == http.MethodPost && r.URL.Path == "/api/v1/retrieval/config/update":
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		if err := json.Unmarshal(raw, &body); err != nil {
			b.t.Errorf("decode POST: %v", err)
		}
		b.posts = append(b.posts, body)
		// Request validation (Pydantic) happens before the handler runs.
		if posted, ok := body["web"].(map[string]any); ok {
			for _, k := range webSearchNonNullableKeys {
				if v, present := posted[k]; present && v == nil {
					w.WriteHeader(http.StatusUnprocessableEntity)
					_, _ = w.Write([]byte(`{"detail": [{"type": "list_type", "loc": ["body", "web", "` + k + `"], "msg": "Input should be a valid list", "input": null}]}`))
					return
				}
			}
		}
		for k, v := range body {
			if k != "web" && v != nil {
				b.state[k] = v
			}
		}
		if posted, ok := body["web"].(map[string]any); ok {
			current, _ := b.state["web"].(map[string]any)
			next := map[string]any{}
			for k := range current {
				next[k] = posted[k]
			}
			if _, ok := posted["WEB_SEARCH_DOMAIN_FILTER_LIST"]; !ok {
				next["WEB_SEARCH_DOMAIN_FILTER_LIST"] = []any{}
			}
			if posted["BRAVE_SEARCH_CONTEXT_TOKENS"] == nil {
				next["BRAVE_SEARCH_CONTEXT_TOKENS"] = current["BRAVE_SEARCH_CONTEXT_TOKENS"]
			}
			b.state["web"] = next
		}
		_, _ = w.Write([]byte(`{"status": true}`))
	default:
		b.t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
	}
}

// webSearchNonNullableKeys are the WebConfig fields whose annotation is not
// Optional (`list[str | None]`) on every release from v0.9.6 to v0.11.0.
var webSearchNonNullableKeys = []string{"WEB_SEARCH_DOMAIN_FILTER_LIST", "YOUTUBE_LOADER_LANGUAGE"}

func (b *webSearchFakeBackend) web() map[string]any {
	b.mu.Lock()
	defer b.mu.Unlock()
	web, _ := b.state["web"].(map[string]any)
	return web
}

func decodeStockWeb(t *testing.T, web string) client.WebSearchConfig {
	t.Helper()
	var c client.WebSearchConfig
	if err := json.Unmarshal([]byte(web), &c); err != nil {
		t.Fatalf("decode web: %v", err)
	}
	return c
}

// fillWebSearchModel sets every attribute of a model to its typed unknown
// (unknown=true, as in a create plan) or null value.
func fillWebSearchModel(unknown bool) webSearchConfigModel {
	var m webSearchConfigModel
	v := reflect.ValueOf(&m).Elem()
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		switch f.Interface().(type) {
		case types.String:
			f.Set(reflect.ValueOf(map[bool]types.String{true: types.StringUnknown(), false: types.StringNull()}[unknown]))
		case types.Bool:
			f.Set(reflect.ValueOf(map[bool]types.Bool{true: types.BoolUnknown(), false: types.BoolNull()}[unknown]))
		case types.Int64:
			f.Set(reflect.ValueOf(map[bool]types.Int64{true: types.Int64Unknown(), false: types.Int64Null()}[unknown]))
		case types.List:
			f.Set(reflect.ValueOf(map[bool]types.List{true: types.ListUnknown(types.StringType), false: types.ListNull(types.StringType)}[unknown]))
		}
	}
	return m
}

func marshalWebUpdate(t *testing.T, update client.WebSearchConfigUpdate) map[string]any {
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

func TestWebSearchUpdateFromUnsetConfigIsEmpty(t *testing.T) {
	for _, unknown := range []bool{true, false} {
		update, diags := webSearchConfigUpdateFromConfig(context.Background(), fillWebSearchModel(unknown))
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if body := marshalWebUpdate(t, update); len(body) != 0 {
			t.Errorf("unknown=%v: an unset configuration must change nothing, got %v", unknown, body)
		}
	}
}

func TestWebSearchUpdateFromKnownValues(t *testing.T) {
	m := fillWebSearchModel(false)
	m.EnableWebSearch = types.BoolValue(false)
	m.WebSearchEngine = types.StringValue("")
	m.WebSearchResultCount = types.Int64Value(0)
	m.WebFetchMaxContentLength = types.Int64Value(4000)
	m.WebSearchDomainFilterList = stringList(t)
	m.YouTubeLoaderLanguage = stringList(t, "de", "en")
	m.LinkupSearchParamsJSON = types.StringValue(`{ "depth": "deep",  "outputType": "searchResults" }`)
	m.TavilyAPIKey = types.StringValue("tvly-secret")
	m.FirecrawlTimeout = types.StringValue("30")
	m.YandexWebSearchConfig = types.StringValue(`{"query": {"searchType": "SEARCH_TYPE_COM"}}`)

	update, diags := webSearchConfigUpdateFromConfig(context.Background(), m)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	body := marshalWebUpdate(t, update)
	want := map[string]string{
		"ENABLE_WEB_SEARCH":             `false`,
		"WEB_SEARCH_ENGINE":             `""`,
		"WEB_SEARCH_RESULT_COUNT":       `0`,
		"WEB_FETCH_MAX_CONTENT_LENGTH":  `4000`,
		"WEB_SEARCH_DOMAIN_FILTER_LIST": `[]`,
		"YOUTUBE_LOADER_LANGUAGE":       `["de","en"]`,
		"LINKUP_SEARCH_PARAMS":          `{"depth":"deep","outputType":"searchResults"}`,
		"TAVILY_API_KEY":                `"tvly-secret"`,
		"FIRECRAWL_TIMEOUT":             `"30"`,
		"YANDEX_WEB_SEARCH_CONFIG":      `"{\"query\": {\"searchType\": \"SEARCH_TYPE_COM\"}}"`,
	}
	if len(body) != len(want) {
		t.Errorf("expected %d keys, got %d: %v", len(want), len(body), body)
	}
	for k, w := range want {
		got, _ := json.Marshal(body[k])
		if string(got) != w {
			t.Errorf("%s: got %s, want %s", k, got, w)
		}
	}
}

func TestWebSearchModelFromAPIShapes(t *testing.T) {
	m, diags := webSearchConfigModelFromAPI(context.Background(), decodeStockWeb(t, stockWebJSON))
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if m.ID.ValueString() != "web_search" {
		t.Errorf("id: %v", m.ID)
	}
	if !m.EnableWebSearch.ValueBool() || m.WebSearchEngine.ValueString() != "tavily" || m.TavilyAPIKey.ValueString() != "tvly-existing" {
		t.Errorf("scalars: %v %v", m.EnableWebSearch, m.WebSearchEngine)
	}
	if m.WebSearchResultCount.ValueInt64() != 3 || m.PlaywrightTimeout.ValueInt64() != 10000 || m.BraveSearchContextTokens.ValueInt64() != 8192 {
		t.Errorf("ints: %v %v %v", m.WebSearchResultCount, m.PlaywrightTimeout, m.BraveSearchContextTokens)
	}
	if !m.WebFetchMaxContentLength.IsNull() {
		t.Errorf("null WEB_FETCH_MAX_CONTENT_LENGTH must be null, got %v", m.WebFetchMaxContentLength)
	}
	if m.WebSearchDomainFilterList.IsNull() || len(m.WebSearchDomainFilterList.Elements()) != 0 {
		t.Errorf("[] must be an empty list, got %v", m.WebSearchDomainFilterList)
	}
	if !m.YouTubeLoaderLanguage.Equal(stringList(t, "en")) {
		t.Errorf("youtube languages: %v", m.YouTubeLoaderLanguage)
	}
	if m.LinkupSearchParamsJSON.ValueString() != `{"depth":"standard"}` {
		t.Errorf("linkup params: %v", m.LinkupSearchParamsJSON)
	}

	// A null list stays null.
	nullList, _ := webSearchConfigModelFromAPI(context.Background(), decodeStockWeb(t, `{"YOUTUBE_LOADER_LANGUAGE": null}`))
	if !nullList.YouTubeLoaderLanguage.IsNull() {
		t.Errorf("null list must be null, got %v", nullList.YouTubeLoaderLanguage)
	}
}

// TestWebSearchModelFromOlderServer: the keys absent on Open WebUI v0.9.6
// (confirmation, Serphouse, Microsoft Web IQ: v0.10.0; OpenSERP: v0.11.0)
// read as null.
func TestWebSearchModelFromOlderServer(t *testing.T) {
	m, diags := webSearchConfigModelFromAPI(context.Background(), decodeStockWeb(t, `{"ENABLE_WEB_SEARCH": false, "WEB_SEARCH_RESULT_COUNT": 3}`))
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	for name, v := range map[string]interface{ IsNull() bool }{
		"enable_web_search_confirmation": m.EnableWebSearchConfirmation, "web_search_confirmation_content": m.WebSearchConfirmationContent,
		"serphouse_api_key": m.SerphouseAPIKey, "serphouse_domain": m.SerphouseDomain,
		"microsoft_web_iq_api_base_url": m.MicrosoftWebIQAPIBaseURL, "microsoft_web_iq_api_key": m.MicrosoftWebIQAPIKey,
		"microsoft_web_iq_language": m.MicrosoftWebIQLanguage, "openserp_base_url": m.OpenSERPBaseURL,
		"linkup_search_params_json": m.LinkupSearchParamsJSON, "web_search_domain_filter_list": m.WebSearchDomainFilterList,
	} {
		if !v.IsNull() {
			t.Errorf("%s must be null", name)
		}
	}
}

func TestWebSearchApplyResultChecks(t *testing.T) {
	older, _ := webSearchConfigModelFromAPI(context.Background(), decodeStockWeb(t, `{"ENABLE_WEB_SEARCH": true}`))
	config := fillWebSearchModel(false)
	if diags := checkWebSearchApplyResult(config, older); diags.HasError() {
		t.Fatalf("unconfigured attributes must not be checked: %v", diags)
	}
	config.OpenSERPBaseURL = types.StringValue("http://openserp:7000")
	config.SerphouseDomain = types.StringValue("bing.com")
	diags := checkWebSearchApplyResult(config, older)
	if diags.ErrorsCount() != 2 {
		t.Fatalf("expected 2 version errors, got %v", diags)
	}
	details := diags[0].Detail() + diags[1].Detail()
	if !strings.Contains(details, "openserp_base_url was added in Open WebUI v0.11.0") || !strings.Contains(details, "serphouse_domain was added in Open WebUI v0.10.0") {
		t.Errorf("unexpected details: %s", details)
	}

	current, _ := webSearchConfigModelFromAPI(context.Background(), decodeStockWeb(t, stockWebJSON))
	if diags := checkWebSearchApplyResult(config, current); diags.HasError() {
		t.Errorf("a current server must pass: %v", diags)
	}
}

// webSearchRaw encodes a model as a raw Terraform value of the resource schema.
func webSearchRaw(t *testing.T, m webSearchConfigModel) (tftypes.Value, tfsdk.State) {
	t.Helper()
	ctx := context.Background()
	var resp resource.SchemaResponse
	NewWebSearchConfigResource().Schema(ctx, resource.SchemaRequest{}, &resp)
	st := tfsdk.State{Schema: resp.Schema, Raw: tftypes.NewValue(resp.Schema.Type().TerraformType(ctx), nil)}
	if diags := st.Set(ctx, &m); diags.HasError() {
		t.Fatalf("encode model: %v", diags)
	}
	return st.Raw, st
}

// TestWebSearchUpdateOneAttributePreservesWebObject drives the resource's
// Update with a configuration that sets only web_search_result_count, while
// the plan (as Terraform builds it with UseStateForUnknown) carries
// prior-state values for every other attribute, two of which were changed
// outside Terraform since (enable_web_search and a secret). The POST must
// carry only `web`, and that web object must be the server's current one,
// key for key and value for value (including the unmodelled
// YOUTUBE_LOADER_TRANSLATION, nulls and secrets), with only
// WEB_SEARCH_RESULT_COUNT changed. The out-of-band values must survive on the
// server, the state must keep the planned values with value-free warnings.
func TestWebSearchUpdateOneAttributePreservesWebObject(t *testing.T) {
	ctx := context.Background()
	backend, server := newWebSearchFakeBackend(t, stockWebJSON)
	defer server.Close()
	apiClient, err := client.NewClient(server.URL, "test-token", false)
	if err != nil {
		t.Fatal(err)
	}

	prior, diags := webSearchConfigModelFromAPI(ctx, decodeStockWeb(t, stockWebJSON))
	if diags.HasError() {
		t.Fatal(diags)
	}
	prior.EnableWebSearch = types.BoolValue(false)          // server: true
	prior.TavilyAPIKey = types.StringValue("tvly-previous") // server: tvly-existing

	config := fillWebSearchModel(false)
	config.WebSearchResultCount = types.Int64Value(7)
	plan := prior
	plan.WebSearchResultCount = types.Int64Value(7)

	configRaw, _ := webSearchRaw(t, config)
	planRaw, _ := webSearchRaw(t, plan)
	priorRaw, priorState := webSearchRaw(t, prior)
	req := resource.UpdateRequest{
		Config: tfsdk.Config{Schema: priorState.Schema, Raw: configRaw},
		Plan:   tfsdk.Plan{Schema: priorState.Schema, Raw: planRaw},
		State:  tfsdk.State{Schema: priorState.Schema, Raw: priorRaw},
	}
	resp := resource.UpdateResponse{State: tfsdk.State{Schema: priorState.Schema, Raw: planRaw}}
	(&webSearchConfigResource{client: apiClient}).Update(ctx, req, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Update failed: %v", resp.Diagnostics)
	}

	if len(backend.posts) != 1 {
		t.Fatalf("expected one POST, got %d", len(backend.posts))
	}
	body := backend.posts[0]
	if len(body) != 1 || body["web"] == nil {
		t.Fatalf("only `web` may be sent, got keys %v", mapKeys(body))
	}
	var want map[string]any
	if err := json.Unmarshal([]byte(stockWebJSON), &want); err != nil {
		t.Fatal(err)
	}
	want["WEB_SEARCH_RESULT_COUNT"] = float64(7)
	if sent, _ := body["web"].(map[string]any); !reflect.DeepEqual(sent, want) {
		for _, k := range mapKeys(want) {
			if !reflect.DeepEqual(sent[k], want[k]) {
				t.Errorf("web key %s: sent %#v, want %#v", k, sent[k], want[k])
			}
		}
		if len(sent) != len(want) {
			t.Errorf("sent %d web keys, want %d", len(sent), len(want))
		}
	}
	if web := backend.web(); web["ENABLE_WEB_SEARCH"] != true || web["TAVILY_API_KEY"] != "tvly-existing" || web["YOUTUBE_LOADER_TRANSLATION"] != "de" {
		t.Errorf("out-of-band and unmodelled values must survive on the server: %v %v %v", web["ENABLE_WEB_SEARCH"], web["TAVILY_API_KEY"], web["YOUTUBE_LOADER_TRANSLATION"])
	}

	var state webSearchConfigModel
	resp.Diagnostics.Append(resp.State.Get(ctx, &state)...)
	if state.WebSearchResultCount.ValueInt64() != 7 || state.EnableWebSearch.ValueBool() || state.TavilyAPIKey.ValueString() != "tvly-previous" {
		t.Errorf("unexpected state: count=%v enable=%v", state.WebSearchResultCount, state.EnableWebSearch)
	}
	if resp.Diagnostics.WarningsCount() != 2 {
		t.Errorf("expected 2 warnings, got %v", resp.Diagnostics)
	}
	for _, d := range resp.Diagnostics {
		if strings.Contains(d.Detail(), "tvly-") {
			t.Errorf("warning must not print values: %s", d.Detail())
		}
	}
}

// TestWebSearchCreateWithoutAttributesSendsNothing: an empty configuration
// reads the settings and never POSTs.
func TestWebSearchCreateWithoutAttributesSendsNothing(t *testing.T) {
	backend, server := newWebSearchFakeBackend(t, stockWebJSON)
	defer server.Close()
	apiClient, err := client.NewClient(server.URL, "test-token", false)
	if err != nil {
		t.Fatal(err)
	}
	state, diags := applyWebSearchConfig(context.Background(), apiClient, fillWebSearchModel(false), fillWebSearchModel(true))
	if diags.HasError() {
		t.Fatalf("apply failed: %v", diags)
	}
	if len(backend.posts) != 0 {
		t.Errorf("nothing may be POSTed, got %v", backend.posts)
	}
	if state.WebSearchEngine.ValueString() != "tavily" || state.TavilyAPIKey.ValueString() != "tvly-existing" {
		t.Errorf("state must come from the server: %v", state.WebSearchEngine)
	}
}

// TestWebSearchApplyWarnsWhenHybridSearchTurnedOff: a web-only POST makes
// Open WebUI reload the reranking model while hybrid search is on, and a
// failed reload turns hybrid search off; the apply warns (without values) and
// still succeeds. A POST that leaves hybrid search on does not warn.
func TestWebSearchApplyWarnsWhenHybridSearchTurnedOff(t *testing.T) {
	for _, turnedOff := range []bool{true, false} {
		hybrid := true
		var mu sync.Mutex
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			defer mu.Unlock()
			switch {
			case r.Method == http.MethodGet && r.URL.Path == "/api/v1/retrieval/config":
				_, _ = fmt.Fprintf(w, `{"status": true, "ENABLE_RAG_HYBRID_SEARCH": %t, "web": %s}`, hybrid, stockWebJSON)
			case r.Method == http.MethodPost && r.URL.Path == "/api/v1/retrieval/config/update":
				if turnedOff {
					hybrid = false
				}
				_, _ = fmt.Fprintf(w, `{"status": true, "ENABLE_RAG_HYBRID_SEARCH": %t}`, hybrid)
			default:
				t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			}
		}))
		apiClient, err := client.NewClient(server.URL, "test-token", false)
		if err != nil {
			t.Fatal(err)
		}
		config := fillWebSearchModel(false)
		config.WebSearchResultCount = types.Int64Value(9)
		plan := fillWebSearchModel(true)
		plan.WebSearchResultCount = types.Int64Value(9)
		_, diags := applyWebSearchConfig(context.Background(), apiClient, config, plan)
		server.Close()
		if diags.HasError() {
			t.Fatalf("turnedOff=%v: apply failed: %v", turnedOff, diags)
		}
		var warned bool
		for _, d := range diags {
			if d.Summary() == "Hybrid search was turned off by Open WebUI" {
				warned = true
				if strings.Contains(d.Detail(), "tvly-") {
					t.Errorf("the warning must not print values: %s", d.Detail())
				}
			}
		}
		if warned != turnedOff {
			t.Errorf("turnedOff=%v: warning emitted=%v (%v)", turnedOff, warned, diags)
		}
	}
}

// TestWebSearchConfigListsAndJSONThroughTerraform runs real terraform plans
// and applies against the fake backend to cover the list shapes end to end:
// a null server list read into state, [] and a populated list round-tripping,
// a list that is unknown at plan time (from terraform_data), and JSON text
// kept as written. Each apply is followed by an empty-plan check.
func TestWebSearchConfigListsAndJSONThroughTerraform(t *testing.T) {
	// Both non-nullable lists start as null on the server: YOUTUBE_LOADER_LANGUAGE
	// is stored as null whenever another client saves a web object without
	// it. Echoing either null back would be an HTTP 422 (see the fake).
	web := strings.Replace(stockWebJSON, `"YOUTUBE_LOADER_LANGUAGE": ["en"]`, `"YOUTUBE_LOADER_LANGUAGE": null`, 1)
	web = strings.Replace(web, `"WEB_SEARCH_DOMAIN_FILTER_LIST": []`, `"WEB_SEARCH_DOMAIN_FILTER_LIST": null`, 1)
	backend, server := newWebSearchFakeBackend(t, web)
	defer server.Close()
	t.Setenv("OPENWEBUI_ENDPOINT", server.URL)
	t.Setenv("OPENWEBUI_TOKEN", "test-token")
	const name = "openwebui_web_search_config.test"

	empty := webSearchHCL(``)
	otherOnly := webSearchHCL(`
  web_search_result_count = 4
`)
	lists := `
resource "terraform_data" "lang" {
  input = "de"
}
` + webSearchHCL(`
  web_search_domain_filter_list = []
  youtube_loader_language       = [terraform_data.lang.output, "en"]
  linkup_search_params_json     = "{ \"depth\": \"deep\" }"
  web_search_result_count       = 5
`)
	populated := `
resource "terraform_data" "lang" {
  input = "de"
}
` + webSearchHCL(`
  web_search_domain_filter_list = ["example.com", "!blocked.example"]
  linkup_search_params_json     = "{\"depth\":\"deep\"}"
`)

	tfresource.UnitTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []tfresource.TestStep{
			{
				Config: empty,
				Check: tfresource.ComposeTestCheckFunc(
					tfresource.TestCheckResourceAttr(name, "id", "web_search"),
					tfresource.TestCheckNoResourceAttr(name, "youtube_loader_language"),
					tfresource.TestCheckNoResourceAttr(name, "web_search_domain_filter_list"),
					tfresource.TestCheckResourceAttr(name, "web_search_engine", "tavily"),
					func(_ *terraform.State) error {
						if len(backend.posts) != 0 {
							t.Errorf("an empty configuration must not POST, got %d", len(backend.posts))
						}
						return nil
					},
				),
			},
			{Config: empty, PlanOnly: true},
			{
				// Neither list is configured and both are null on the server:
				// they must be left out of the web object, not echoed as null.
				Config: otherOnly,
				Check: tfresource.ComposeTestCheckFunc(
					tfresource.TestCheckResourceAttr(name, "web_search_result_count", "4"),
					tfresource.TestCheckNoResourceAttr(name, "youtube_loader_language"),
					// Omitted, the domain filter takes its Pydantic default []
					// on the server. Terraform planned the prior null for the
					// unconfigured attribute, so the state keeps null (with a
					// warning) and the next refresh reads [].
					tfresource.TestCheckNoResourceAttr(name, "web_search_domain_filter_list"),
					func(_ *terraform.State) error {
						last, _ := backend.posts[len(backend.posts)-1]["web"].(map[string]any)
						for _, k := range webSearchNonNullableKeys {
							if _, present := last[k]; present {
								t.Errorf("a null %s must not be sent", k)
							}
						}
						w := backend.web()
						if w["YOUTUBE_LOADER_LANGUAGE"] != nil {
							t.Errorf("YOUTUBE_LOADER_LANGUAGE must stay null on the server, got %v", w["YOUTUBE_LOADER_LANGUAGE"])
						}
						if !reflect.DeepEqual(w["WEB_SEARCH_DOMAIN_FILTER_LIST"], []any{}) {
							t.Errorf("WEB_SEARCH_DOMAIN_FILTER_LIST must take its default [], got %v", w["WEB_SEARCH_DOMAIN_FILTER_LIST"])
						}
						return nil
					},
				),
			},
			{Config: otherOnly, PlanOnly: true},
			{
				Config: lists,
				Check: tfresource.ComposeTestCheckFunc(
					tfresource.TestCheckResourceAttr(name, "web_search_domain_filter_list.#", "0"),
					tfresource.TestCheckResourceAttr(name, "youtube_loader_language.#", "2"),
					tfresource.TestCheckResourceAttr(name, "youtube_loader_language.0", "de"),
					tfresource.TestCheckResourceAttr(name, "linkup_search_params_json", "{ \"depth\": \"deep\" }"),
					tfresource.TestCheckResourceAttr(name, "web_search_result_count", "5"),
					func(_ *terraform.State) error {
						w := backend.web()
						if !reflect.DeepEqual(w["YOUTUBE_LOADER_LANGUAGE"], []any{"de", "en"}) || w["TAVILY_API_KEY"] != "tvly-existing" || w["YOUTUBE_LOADER_TRANSLATION"] != "de" {
							t.Errorf("unexpected server web: %v %v", w["YOUTUBE_LOADER_LANGUAGE"], w["YOUTUBE_LOADER_TRANSLATION"])
						}
						return nil
					},
				),
			},
			{Config: lists, PlanOnly: true},
			{
				// youtube_loader_language is no longer configured: it keeps
				// the server value and must not be sent differently.
				Config: populated,
				Check: tfresource.ComposeTestCheckFunc(
					tfresource.TestCheckResourceAttr(name, "web_search_domain_filter_list.#", "2"),
					tfresource.TestCheckResourceAttr(name, "web_search_domain_filter_list.1", "!blocked.example"),
					tfresource.TestCheckResourceAttr(name, "youtube_loader_language.#", "2"),
					tfresource.TestCheckResourceAttr(name, "linkup_search_params_json", "{\"depth\":\"deep\"}"),
					tfresource.TestCheckResourceAttr(name, "web_search_result_count", "5"),
				),
			},
			{Config: populated, PlanOnly: true},
		},
	})
	if backend.state["TOP_K"] != float64(3) || backend.state["RAG_TEMPLATE"] != "### Task:" {
		t.Errorf("top-level retrieval settings must be untouched: %v", backend.state)
	}
	for i, p := range backend.posts {
		if len(p) != 1 || p["web"] == nil {
			t.Errorf("POST %d must carry only web, got keys %v", i, mapKeys(p))
		}
	}
}

// TestReconcileKeepsPlannedNull: an unconfigured attribute planned as null
// (the prior state) that the server now reports non-null keeps null with a
// value-free warning, since Terraform requires the planned null; here the
// domain filter list, which Open WebUI turns from null into [] when it is
// left out of a web object.
func TestReconcileKeepsPlannedNull(t *testing.T) {
	fresh, _ := webSearchConfigModelFromAPI(context.Background(), decodeStockWeb(t, stockWebJSON))
	config := fillWebSearchModel(false)
	plan := fresh
	plan.WebSearchDomainFilterList = types.ListNull(types.StringType)
	got, diags := reconcileUnconfiguredAttributes(config, plan, fresh, "test")
	if diags.HasError() || diags.WarningsCount() != 1 {
		t.Fatalf("expected exactly one warning, got %v", diags)
	}
	if !got.WebSearchDomainFilterList.IsNull() {
		t.Errorf("the planned null must be kept, got %v", got.WebSearchDomainFilterList)
	}
}

func TestWebSearchConfigValidators(t *testing.T) {
	for _, attr := range []string{"web_search_concurrent_requests", "web_fetch_max_content_length", "web_loader_concurrent_requests"} {
		planOnlyValid(t, webSearchHCL(`  `+attr+` = 0`))
		planOnlyRejected(t, webSearchHCL(`  `+attr+` = -1`), `(?s)`+attr+`.*at\s+least\s+0`)
	}

	for _, ok := range []string{"", "10", "2.5"} {
		planOnlyValid(t, webSearchHCL(`  web_loader_timeout = "`+ok+`"`))
	}
	for _, bad := range []string{"10s", " 10", "-1", "+1", "1e3", ".5", "1."} {
		planOnlyRejected(t, webSearchHCL(`  web_loader_timeout = "`+bad+`"`), `(?s)web_loader_timeout.*plain\s+decimal\s+number`)
	}
	for _, ok := range []string{"", "30"} {
		planOnlyValid(t, webSearchHCL(`  firecrawl_timeout = "`+ok+`"`))
	}
	for _, bad := range []string{"2.5", "30s", "-1", "+30", "3e1"} {
		planOnlyRejected(t, webSearchHCL(`  firecrawl_timeout = "`+bad+`"`), `(?s)firecrawl_timeout.*whole\s+number`)
	}

	planOnlyValid(t, webSearchHCL(`  yandex_web_search_config = ""`))
	planOnlyValid(t, webSearchHCL(`  yandex_web_search_config = jsonencode({ query = { searchType = "SEARCH_TYPE_COM" } })`))
	for _, bad := range []string{`[]`, `{`, `x`} {
		planOnlyRejected(t, webSearchHCL(`  yandex_web_search_config = "`+bad+`"`), `(?s)yandex_web_search_config.*JSON\s+object`)
	}

	planOnlyValid(t, webSearchHCL(`  linkup_search_params_json = "{}"`))
	for _, bad := range []string{`null`, `[]`, `{`, ``} {
		planOnlyRejected(t, webSearchHCL(`  linkup_search_params_json = "`+bad+`"`), `(?s)linkup_search_params_json.*JSON\s+object`)
	}
}

// TestWebSearchConfigSchemaShape pins the attribute count (id plus the 73
// modelled WebConfig fields), the Description/MarkdownDescription rule, the
// Optional+Computed shape and the exact set of Sensitive attributes.
func TestWebSearchConfigSchemaShape(t *testing.T) {
	var resp resource.SchemaResponse
	NewWebSearchConfigResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if len(resp.Schema.Attributes) != 74 {
		t.Fatalf("expected id plus 73 attributes, got %d", len(resp.Schema.Attributes))
	}
	sensitive := map[string]bool{
		"ollama_cloud_web_search_api_key": true, "yacy_password": true, "google_pse_api_key": true,
		"brave_search_api_key": true, "kagi_search_api_key": true, "mojeek_search_api_key": true,
		"bocha_search_api_key": true, "serpstack_api_key": true, "serper_api_key": true, "serphouse_api_key": true,
		"serply_api_key": true, "tavily_api_key": true, "searchapi_api_key": true, "serpapi_api_key": true,
		"jina_api_key": true, "bing_search_v7_subscription_key": true, "exa_api_key": true, "perplexity_api_key": true,
		"microsoft_web_iq_api_key": true, "sougou_api_sid": true, "sougou_api_sk": true, "playwright_ws_url": true,
		"firecrawl_api_key": true, "external_web_search_api_key": true, "external_web_loader_api_key": true,
		"youtube_loader_proxy_url": true, "yandex_web_search_api_key": true, "youcom_api_key": true, "linkup_api_key": true,
	}
	if len(sensitive) != 29 {
		t.Fatalf("sensitive set has %d entries", len(sensitive))
	}
	for name, a := range resp.Schema.Attributes {
		if a.GetDescription() == "" || a.GetMarkdownDescription() == "" {
			t.Errorf("attribute %q needs both Description and MarkdownDescription", name)
		}
		if a.IsSensitive() != sensitive[name] {
			t.Errorf("attribute %q: unexpected Sensitive=%v", name, a.IsSensitive())
		}
		if name == "id" {
			continue
		}
		if !a.IsOptional() || !a.IsComputed() {
			t.Errorf("attribute %q must be Optional+Computed", name)
		}
	}
}

func mapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
