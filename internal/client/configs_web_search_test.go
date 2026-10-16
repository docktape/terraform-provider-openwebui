package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// stockWebSearchObject is the `web` object of a stock Open WebUI v0.11.0 GET
// /retrieval/config response (all 74 keys), with two secrets populated, a
// non-null YOUTUBE_LOADER_TRANSLATION (not modelled by WebSearchConfig) and
// one key unknown to this client, standing in for a key added by a later
// Open WebUI release.
const stockWebSearchObject = `{
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
	"YOUCOM_API_KEY": "", "LINKUP_API_KEY": "", "LINKUP_SEARCH_PARAMS": {"depth": "standard"},
	"FUTURE_WEB_KEY": {"nested": [1, 2]}
}`

func stockRetrievalWithWeb(web string) string {
	return `{"status": true, "TOP_K": 3, "RAG_TEMPLATE": "### Task:", "web": ` + web + `}`
}

// webSearchFakeServer serves GET /retrieval/config with the given web object
// and records every POST /retrieval/config/update body.
func webSearchFakeServer(t *testing.T, web string, posts *[]map[string]any) (*Client, func()) {
	t.Helper()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/retrieval/config":
			_, _ = w.Write([]byte(stockRetrievalWithWeb(web)))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/retrieval/config/update":
			raw, _ := io.ReadAll(r.Body)
			var body map[string]any
			if err := json.Unmarshal(raw, &body); err != nil {
				t.Errorf("decode POST body: %v", err)
			}
			*posts = append(*posts, body)
			_, _ = w.Write([]byte(`{"status": true}`))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	})
	c, server := newConfigTestClient(handler)
	return c, server.Close
}

func TestGetWebSearchConfig(t *testing.T) {
	var posts []map[string]any
	c, done := webSearchFakeServer(t, stockWebSearchObject, &posts)
	defer done()

	got, err := c.GetWebSearchConfig(context.Background())
	if err != nil {
		t.Fatalf("GetWebSearchConfig failed: %v", err)
	}
	if got.EnableWebSearch == nil || !*got.EnableWebSearch || got.WebSearchEngine == nil || *got.WebSearchEngine != "tavily" {
		t.Errorf("unexpected toggles: %+v", got)
	}
	if got.WebSearchResultCount == nil || *got.WebSearchResultCount != 3 || got.PlaywrightTimeout == nil || *got.PlaywrightTimeout != 10000 {
		t.Errorf("unexpected ints: %v %v", got.WebSearchResultCount, got.PlaywrightTimeout)
	}
	if got.WebFetchMaxContentLength != nil {
		t.Errorf("null WEB_FETCH_MAX_CONTENT_LENGTH must stay nil, got %v", *got.WebFetchMaxContentLength)
	}
	if got.WebSearchDomainFilterList == nil || len(got.WebSearchDomainFilterList) != 0 {
		t.Errorf("[] must decode to an empty non-nil slice, got %#v", got.WebSearchDomainFilterList)
	}
	if !reflect.DeepEqual(got.YouTubeLoaderLanguage, []string{"en"}) {
		t.Errorf("YOUTUBE_LOADER_LANGUAGE: %#v", got.YouTubeLoaderLanguage)
	}
	if string(got.LinkupSearchParams) != `{"depth": "standard"}` {
		t.Errorf("LINKUP_SEARCH_PARAMS must be kept raw, got %s", got.LinkupSearchParams)
	}
	if got.TavilyAPIKey == nil || *got.TavilyAPIKey != "tvly-existing" {
		t.Error("TAVILY_API_KEY not decoded")
	}
	if len(posts) != 0 {
		t.Errorf("a read must not POST, got %v", posts)
	}
}

// TestGetWebSearchConfigLenientInts: the integer fields accept a JSON number,
// a numeric string or "" (as nil), like the other config readers.
func TestGetWebSearchConfigLenientInts(t *testing.T) {
	var posts []map[string]any
	c, done := webSearchFakeServer(t, `{"WEB_SEARCH_RESULT_COUNT": "5", "PLAYWRIGHT_TIMEOUT": "", "BRAVE_SEARCH_CONTEXT_TOKENS": 4096}`, &posts)
	defer done()
	got, err := c.GetWebSearchConfig(context.Background())
	if err != nil {
		t.Fatalf("GetWebSearchConfig failed: %v", err)
	}
	if got.WebSearchResultCount == nil || *got.WebSearchResultCount != 5 || got.PlaywrightTimeout != nil || *got.BraveSearchContextTokens != 4096 {
		t.Errorf("unexpected ints: %v %v %v", got.WebSearchResultCount, got.PlaywrightTimeout, got.BraveSearchContextTokens)
	}
	// Keys absent on an older server decode as nil.
	if got.OpenSERPBaseURL != nil || got.EnableWebSearchConfirmation != nil || got.YouTubeLoaderLanguage != nil {
		t.Error("absent keys must decode as nil")
	}

	c2, done2 := webSearchFakeServer(t, `{"WEB_SEARCH_RESULT_COUNT": "many"}`, &posts)
	defer done2()
	if _, err := c2.GetWebSearchConfig(context.Background()); err == nil || !strings.Contains(err.Error(), "WEB_SEARCH_RESULT_COUNT") {
		t.Errorf("expected a WEB_SEARCH_RESULT_COUNT decode error, got %v", err)
	}
}

func TestGetWebSearchConfigWithoutWebObject(t *testing.T) {
	var posts []map[string]any
	c, done := webSearchFakeServer(t, `null`, &posts)
	defer done()
	if _, err := c.GetWebSearchConfig(context.Background()); err == nil {
		t.Error("expected an error when the response has no web object")
	}
}

// TestUpdateWebSearchConfigMergesIntoCurrentWebObject is the core contract:
// Open WebUI replaces the whole web configuration with whatever `web` object
// it is sent (every WebConfig field left out is stored as None), so the
// client must POST the complete current object with only the updated keys
// changed. Every other key, including secrets, keys the client does not
// model (YOUTUBE_LOADER_TRANSLATION, FUTURE_WEB_KEY) and nulls, must be sent
// back verbatim, and no top-level retrieval key may be sent.
func TestUpdateWebSearchConfigMergesIntoCurrentWebObject(t *testing.T) {
	var posts []map[string]any
	c, done := webSearchFakeServer(t, stockWebSearchObject, &posts)
	defer done()

	count := int64(7)
	filters := []string{}
	err := c.UpdateWebSearchConfig(context.Background(), WebSearchConfigUpdate{
		WebSearchResultCount:      &count,
		WebSearchDomainFilterList: &filters,
	})
	if err != nil {
		t.Fatalf("UpdateWebSearchConfig failed: %v", err)
	}
	if len(posts) != 1 {
		t.Fatalf("expected exactly one POST, got %d", len(posts))
	}
	body := posts[0]
	if len(body) != 1 {
		t.Errorf("only `web` may be sent (top-level retrieval settings must be left alone), got keys %v", sortedKeys(body))
	}
	web, ok := body["web"].(map[string]any)
	if !ok {
		t.Fatalf("web must be an object, got %T", body["web"])
	}

	var want map[string]any
	if err := json.Unmarshal([]byte(stockWebSearchObject), &want); err != nil {
		t.Fatal(err)
	}
	want["WEB_SEARCH_RESULT_COUNT"] = float64(7)
	want["WEB_SEARCH_DOMAIN_FILTER_LIST"] = []any{}
	if !reflect.DeepEqual(web, want) {
		for _, k := range sortedKeys(want) {
			if !reflect.DeepEqual(web[k], want[k]) {
				t.Errorf("%s: sent %#v, want %#v", k, web[k], want[k])
			}
		}
		for _, k := range sortedKeys(web) {
			if _, ok := want[k]; !ok {
				t.Errorf("unexpected key %s sent", k)
			}
		}
	}
}

// TestUpdateWebSearchConfigEmptyUpdateSendsNothing: with nothing to change the
// client neither reads nor writes (a web-only POST still makes Open WebUI
// re-save and, with hybrid search on, reload the reranking model).
func TestUpdateWebSearchConfigEmptyUpdateSendsNothing(t *testing.T) {
	handler := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
	})
	c, server := newConfigTestClient(handler)
	defer server.Close()
	if err := c.UpdateWebSearchConfig(context.Background(), WebSearchConfigUpdate{}); err != nil {
		t.Fatalf("UpdateWebSearchConfig failed: %v", err)
	}
}

// TestUpdateWebSearchConfigRefusesWithoutCurrentWebObject: sending a partial
// web object would reset every other web setting, so the client must refuse
// when the server does not return the current one.
func TestUpdateWebSearchConfigRefusesWithoutCurrentWebObject(t *testing.T) {
	for _, web := range []string{`null`, `[]`, `"x"`} {
		var posts []map[string]any
		c, done := webSearchFakeServer(t, web, &posts)
		enable := true
		err := c.UpdateWebSearchConfig(context.Background(), WebSearchConfigUpdate{EnableWebSearch: &enable})
		done()
		if err == nil {
			t.Errorf("web=%s: expected an error", web)
		}
		if len(posts) != 0 {
			t.Errorf("web=%s: nothing may be POSTed, got %v", web, posts)
		}
	}
}

// TestUpdateWebSearchConfigOmitsNullNonNullableKeys: exactly two WebConfig
// fields have a non-Optional annotation on every release from v0.9.6 to
// v0.11.0, WEB_SEARCH_DOMAIN_FILTER_LIST (`list[str | None] = []`) and
// YOUTUBE_LOADER_LANGUAGE (`list[str | None] = None`). Pydantic rejects an
// explicit null for them ("Input should be a valid list", HTTP 422, confirmed
// against a live v0.11.0), while an omitted key takes the default. The server
// stores null for YOUTUBE_LOADER_LANGUAGE whenever another client saves a web
// object without it, so a null read back must be left out, never echoed. A
// null in any nullable field is echoed as null.
func TestUpdateWebSearchConfigOmitsNullNonNullableKeys(t *testing.T) {
	for _, key := range []string{"WEB_SEARCH_DOMAIN_FILTER_LIST", "YOUTUBE_LOADER_LANGUAGE"} {
		t.Run(key, func(t *testing.T) {
			var posts []map[string]any
			web := `{"` + key + `": null, "WEB_FETCH_MAX_CONTENT_LENGTH": null, "LINKUP_SEARCH_PARAMS": null, "YOUTUBE_LOADER_TRANSLATION": null, "WEB_SEARCH_ENGINE": "searxng"}`
			c, done := webSearchFakeServer(t, web, &posts)
			defer done()
			enable := true
			if err := c.UpdateWebSearchConfig(context.Background(), WebSearchConfigUpdate{EnableWebSearch: &enable}); err != nil {
				t.Fatalf("UpdateWebSearchConfig failed: %v", err)
			}
			sent, ok := posts[0]["web"].(map[string]any)
			if !ok {
				t.Fatalf("web must be an object, got %T", posts[0]["web"])
			}
			if _, present := sent[key]; present {
				t.Errorf("a null %s must be left out (an explicit null is an HTTP 422), got %v", key, sent)
			}
			for _, nullable := range []string{"WEB_FETCH_MAX_CONTENT_LENGTH", "LINKUP_SEARCH_PARAMS", "YOUTUBE_LOADER_TRANSLATION"} {
				if v, present := sent[nullable]; !present || v != nil {
					t.Errorf("a nullable null %s must be echoed as null, got %v (present=%v)", nullable, v, present)
				}
			}
			if sent["ENABLE_WEB_SEARCH"] != true || sent["WEB_SEARCH_ENGINE"] != "searxng" {
				t.Errorf("unexpected web: %v", sent)
			}
		})
	}

	// A set value of a non-nullable key is still sent, and an update of it
	// replaces a null read back.
	var posts []map[string]any
	c, done := webSearchFakeServer(t, `{"YOUTUBE_LOADER_LANGUAGE": null, "WEB_SEARCH_DOMAIN_FILTER_LIST": ["a.example"]}`, &posts)
	defer done()
	langs := []string{"de"}
	if err := c.UpdateWebSearchConfig(context.Background(), WebSearchConfigUpdate{YouTubeLoaderLanguage: &langs}); err != nil {
		t.Fatalf("UpdateWebSearchConfig failed: %v", err)
	}
	sent, _ := posts[0]["web"].(map[string]any)
	if !reflect.DeepEqual(sent["YOUTUBE_LOADER_LANGUAGE"], []any{"de"}) || !reflect.DeepEqual(sent["WEB_SEARCH_DOMAIN_FILTER_LIST"], []any{"a.example"}) {
		t.Errorf("unexpected web: %v", sent)
	}
}

// TestWebSearchConfigUpdateWireKeys pins the literal wire key of every
// WebSearchConfigUpdate field against the v0.11.0 WebConfig field list
// (backend/open_webui/routers/retrieval.py), minus YOUTUBE_LOADER_TRANSLATION,
// which is deliberately not modelled.
func TestWebSearchConfigUpdateWireKeys(t *testing.T) {
	var form WebSearchConfigUpdate
	v := reflect.ValueOf(&form).Elem()
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		switch f.Kind() {
		case reflect.Pointer:
			f.Set(reflect.New(f.Type().Elem()))
		case reflect.Slice: // json.RawMessage
			f.Set(reflect.ValueOf(json.RawMessage(`{}`)))
		}
	}
	raw, err := json.Marshal(form)
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"ENABLE_WEB_SEARCH", "ENABLE_WEB_SEARCH_CONFIRMATION", "WEB_SEARCH_CONFIRMATION_CONTENT", "WEB_SEARCH_ENGINE",
		"WEB_SEARCH_TRUST_ENV", "WEB_SEARCH_RESULT_COUNT", "WEB_SEARCH_CONCURRENT_REQUESTS", "WEB_SEARCH_DOMAIN_FILTER_LIST",
		"WEB_FETCH_MAX_CONTENT_LENGTH", "WEB_LOADER_CONCURRENT_REQUESTS", "BYPASS_WEB_SEARCH_EMBEDDING_AND_RETRIEVAL",
		"BYPASS_WEB_SEARCH_WEB_LOADER", "OLLAMA_CLOUD_WEB_SEARCH_API_KEY", "SEARXNG_QUERY_URL", "SEARXNG_LANGUAGE",
		"OPENSERP_BASE_URL", "YACY_QUERY_URL", "YACY_USERNAME", "YACY_PASSWORD", "GOOGLE_PSE_API_KEY", "GOOGLE_PSE_ENGINE_ID",
		"BRAVE_SEARCH_API_KEY", "BRAVE_SEARCH_CONTEXT_TOKENS", "KAGI_SEARCH_API_KEY", "MOJEEK_SEARCH_API_KEY",
		"BOCHA_SEARCH_API_KEY", "SERPSTACK_API_KEY", "SERPSTACK_HTTPS", "SERPER_API_KEY", "SERPHOUSE_API_KEY",
		"SERPHOUSE_DOMAIN", "SERPLY_API_KEY", "DDGS_BACKEND", "TAVILY_API_KEY", "SEARCHAPI_API_KEY", "SEARCHAPI_ENGINE",
		"SERPAPI_API_KEY", "SERPAPI_ENGINE", "JINA_API_KEY", "JINA_API_BASE_URL", "BING_SEARCH_V7_ENDPOINT",
		"BING_SEARCH_V7_SUBSCRIPTION_KEY", "EXA_API_KEY", "PERPLEXITY_API_KEY", "PERPLEXITY_MODEL",
		"PERPLEXITY_SEARCH_CONTEXT_USAGE", "PERPLEXITY_SEARCH_API_URL", "MICROSOFT_WEB_IQ_API_BASE_URL",
		"MICROSOFT_WEB_IQ_API_KEY", "MICROSOFT_WEB_IQ_LANGUAGE", "SOUGOU_API_SID", "SOUGOU_API_SK", "WEB_LOADER_ENGINE",
		"WEB_LOADER_TIMEOUT", "ENABLE_WEB_LOADER_SSL_VERIFICATION", "PLAYWRIGHT_WS_URL", "PLAYWRIGHT_TIMEOUT",
		"FIRECRAWL_API_KEY", "FIRECRAWL_API_BASE_URL", "FIRECRAWL_TIMEOUT", "TAVILY_EXTRACT_DEPTH",
		"EXTERNAL_WEB_SEARCH_URL", "EXTERNAL_WEB_SEARCH_API_KEY", "EXTERNAL_WEB_LOADER_URL", "EXTERNAL_WEB_LOADER_API_KEY",
		"YOUTUBE_LOADER_LANGUAGE", "YOUTUBE_LOADER_PROXY_URL", "YANDEX_WEB_SEARCH_URL", "YANDEX_WEB_SEARCH_API_KEY",
		"YANDEX_WEB_SEARCH_CONFIG", "YOUCOM_API_KEY", "LINKUP_API_KEY", "LINKUP_SEARCH_PARAMS",
	}
	sort.Strings(want)
	if got := sortedKeys(body); !reflect.DeepEqual(got, want) {
		t.Errorf("wire keys differ\n got: %v\nwant: %v", got, want)
	}
}

// TestRetrievalWritesAreSerialized: Open WebUI's update handler loads, merges
// and re-saves every retrieval setting, so overlapping updates from the
// retrieval and web search resources (applied in parallel by Terraform) could
// write back each other's stale values. Through one client, no request of a
// web search update (its read or its write) may overlap a retrieval update.
func TestRetrievalWritesAreSerialized(t *testing.T) {
	var inFlight, overlaps atomic.Int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if inFlight.Add(1) > 1 {
			overlaps.Add(1)
		}
		time.Sleep(10 * time.Millisecond)
		inFlight.Add(-1)
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(stockRetrievalWithWeb(stockWebSearchObject)))
			return
		}
		_, _ = w.Write([]byte(`{"status": true}`))
	})
	c, server := newConfigTestClient(handler)
	defer server.Close()

	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			topK := int64(5)
			if err := c.UpdateRetrievalConfig(context.Background(), RetrievalConfigUpdateForm{TopK: &topK}); err != nil {
				t.Errorf("UpdateRetrievalConfig: %v", err)
			}
		}()
		go func() {
			defer wg.Done()
			enable := true
			if err := c.UpdateWebSearchConfig(context.Background(), WebSearchConfigUpdate{EnableWebSearch: &enable}); err != nil {
				t.Errorf("UpdateWebSearchConfig: %v", err)
			}
		}()
	}
	wg.Wait()
	if n := overlaps.Load(); n != 0 {
		t.Errorf("%d requests overlapped another update", n)
	}
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
