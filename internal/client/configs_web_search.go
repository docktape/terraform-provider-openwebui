package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// WebSearchConfig is the nested `web` object (Open WebUI's WebConfig) of the
// GET /retrieval/config response. Every field is a pointer, nil slice or raw
// JSON so that a key absent on an older server, or served as null, reads as
// nil. Lists keep the null/[] distinction. The integer fields are decoded
// leniently (see UnmarshalJSON).
//
// YOUTUBE_LOADER_TRANSLATION is deliberately not modelled: Open WebUI keeps it
// in process memory only (app.state, reset to null on restart) and nothing in
// v0.11.0 reads it. UpdateWebSearchConfig sends it back unchanged.
type WebSearchConfig struct {
	EnableWebSearch                      *bool           `json:"ENABLE_WEB_SEARCH"`
	EnableWebSearchConfirmation          *bool           `json:"ENABLE_WEB_SEARCH_CONFIRMATION"`
	WebSearchConfirmationContent         *string         `json:"WEB_SEARCH_CONFIRMATION_CONTENT"`
	WebSearchEngine                      *string         `json:"WEB_SEARCH_ENGINE"`
	WebSearchTrustEnv                    *bool           `json:"WEB_SEARCH_TRUST_ENV"`
	WebSearchResultCount                 *int64          `json:"-"`
	WebSearchConcurrentRequests          *int64          `json:"-"`
	WebSearchDomainFilterList            []string        `json:"WEB_SEARCH_DOMAIN_FILTER_LIST"`
	WebFetchMaxContentLength             *int64          `json:"-"`
	WebLoaderConcurrentRequests          *int64          `json:"-"`
	BypassWebSearchEmbeddingAndRetrieval *bool           `json:"BYPASS_WEB_SEARCH_EMBEDDING_AND_RETRIEVAL"`
	BypassWebSearchWebLoader             *bool           `json:"BYPASS_WEB_SEARCH_WEB_LOADER"`
	OllamaCloudWebSearchAPIKey           *string         `json:"OLLAMA_CLOUD_WEB_SEARCH_API_KEY"`
	SearXNGQueryURL                      *string         `json:"SEARXNG_QUERY_URL"`
	SearXNGLanguage                      *string         `json:"SEARXNG_LANGUAGE"`
	OpenSERPBaseURL                      *string         `json:"OPENSERP_BASE_URL"`
	YaCyQueryURL                         *string         `json:"YACY_QUERY_URL"`
	YaCyUsername                         *string         `json:"YACY_USERNAME"`
	YaCyPassword                         *string         `json:"YACY_PASSWORD"`
	GooglePSEAPIKey                      *string         `json:"GOOGLE_PSE_API_KEY"`
	GooglePSEEngineID                    *string         `json:"GOOGLE_PSE_ENGINE_ID"`
	BraveSearchAPIKey                    *string         `json:"BRAVE_SEARCH_API_KEY"`
	BraveSearchContextTokens             *int64          `json:"-"`
	KagiSearchAPIKey                     *string         `json:"KAGI_SEARCH_API_KEY"`
	MojeekSearchAPIKey                   *string         `json:"MOJEEK_SEARCH_API_KEY"`
	BochaSearchAPIKey                    *string         `json:"BOCHA_SEARCH_API_KEY"`
	SerpstackAPIKey                      *string         `json:"SERPSTACK_API_KEY"`
	SerpstackHTTPS                       *bool           `json:"SERPSTACK_HTTPS"`
	SerperAPIKey                         *string         `json:"SERPER_API_KEY"`
	SerphouseAPIKey                      *string         `json:"SERPHOUSE_API_KEY"`
	SerphouseDomain                      *string         `json:"SERPHOUSE_DOMAIN"`
	SerplyAPIKey                         *string         `json:"SERPLY_API_KEY"`
	DDGSBackend                          *string         `json:"DDGS_BACKEND"`
	TavilyAPIKey                         *string         `json:"TAVILY_API_KEY"`
	SearchAPIAPIKey                      *string         `json:"SEARCHAPI_API_KEY"`
	SearchAPIEngine                      *string         `json:"SEARCHAPI_ENGINE"`
	SerpAPIAPIKey                        *string         `json:"SERPAPI_API_KEY"`
	SerpAPIEngine                        *string         `json:"SERPAPI_ENGINE"`
	JinaAPIKey                           *string         `json:"JINA_API_KEY"`
	JinaAPIBaseURL                       *string         `json:"JINA_API_BASE_URL"`
	BingSearchV7Endpoint                 *string         `json:"BING_SEARCH_V7_ENDPOINT"`
	BingSearchV7SubscriptionKey          *string         `json:"BING_SEARCH_V7_SUBSCRIPTION_KEY"`
	ExaAPIKey                            *string         `json:"EXA_API_KEY"`
	PerplexityAPIKey                     *string         `json:"PERPLEXITY_API_KEY"`
	PerplexityModel                      *string         `json:"PERPLEXITY_MODEL"`
	PerplexitySearchContextUsage         *string         `json:"PERPLEXITY_SEARCH_CONTEXT_USAGE"`
	PerplexitySearchAPIURL               *string         `json:"PERPLEXITY_SEARCH_API_URL"`
	MicrosoftWebIQAPIBaseURL             *string         `json:"MICROSOFT_WEB_IQ_API_BASE_URL"`
	MicrosoftWebIQAPIKey                 *string         `json:"MICROSOFT_WEB_IQ_API_KEY"`
	MicrosoftWebIQLanguage               *string         `json:"MICROSOFT_WEB_IQ_LANGUAGE"`
	SougouAPISID                         *string         `json:"SOUGOU_API_SID"`
	SougouAPISK                          *string         `json:"SOUGOU_API_SK"`
	WebLoaderEngine                      *string         `json:"WEB_LOADER_ENGINE"`
	WebLoaderTimeout                     *string         `json:"WEB_LOADER_TIMEOUT"`
	EnableWebLoaderSSLVerification       *bool           `json:"ENABLE_WEB_LOADER_SSL_VERIFICATION"`
	PlaywrightWSURL                      *string         `json:"PLAYWRIGHT_WS_URL"`
	PlaywrightTimeout                    *int64          `json:"-"`
	FirecrawlAPIKey                      *string         `json:"FIRECRAWL_API_KEY"`
	FirecrawlAPIBaseURL                  *string         `json:"FIRECRAWL_API_BASE_URL"`
	FirecrawlTimeout                     *string         `json:"FIRECRAWL_TIMEOUT"`
	TavilyExtractDepth                   *string         `json:"TAVILY_EXTRACT_DEPTH"`
	ExternalWebSearchURL                 *string         `json:"EXTERNAL_WEB_SEARCH_URL"`
	ExternalWebSearchAPIKey              *string         `json:"EXTERNAL_WEB_SEARCH_API_KEY"`
	ExternalWebLoaderURL                 *string         `json:"EXTERNAL_WEB_LOADER_URL"`
	ExternalWebLoaderAPIKey              *string         `json:"EXTERNAL_WEB_LOADER_API_KEY"`
	YouTubeLoaderLanguage                []string        `json:"YOUTUBE_LOADER_LANGUAGE"`
	YouTubeLoaderProxyURL                *string         `json:"YOUTUBE_LOADER_PROXY_URL"`
	YandexWebSearchURL                   *string         `json:"YANDEX_WEB_SEARCH_URL"`
	YandexWebSearchAPIKey                *string         `json:"YANDEX_WEB_SEARCH_API_KEY"`
	YandexWebSearchConfig                *string         `json:"YANDEX_WEB_SEARCH_CONFIG"`
	YouComAPIKey                         *string         `json:"YOUCOM_API_KEY"`
	LinkupAPIKey                         *string         `json:"LINKUP_API_KEY"`
	LinkupSearchParams                   json.RawMessage `json:"LINKUP_SEARCH_PARAMS"`
}

// UnmarshalJSON decodes the integer fields via decodeIntOrEmptyString (a
// JSON number, a numeric string, or ""/null as nil). They are int-typed in
// WebConfig and int() of their env default, but the shared retrieval readers
// all tolerate the string shapes Open WebUI has served for other int fields.
func (w *WebSearchConfig) UnmarshalJSON(data []byte) error {
	type alias WebSearchConfig
	aux := struct {
		*alias
		WebSearchResultCount        json.RawMessage `json:"WEB_SEARCH_RESULT_COUNT"`
		WebSearchConcurrentRequests json.RawMessage `json:"WEB_SEARCH_CONCURRENT_REQUESTS"`
		WebFetchMaxContentLength    json.RawMessage `json:"WEB_FETCH_MAX_CONTENT_LENGTH"`
		WebLoaderConcurrentRequests json.RawMessage `json:"WEB_LOADER_CONCURRENT_REQUESTS"`
		BraveSearchContextTokens    json.RawMessage `json:"BRAVE_SEARCH_CONTEXT_TOKENS"`
		PlaywrightTimeout           json.RawMessage `json:"PLAYWRIGHT_TIMEOUT"`
	}{alias: (*alias)(w)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	ints := []struct {
		key string
		raw json.RawMessage
		dst **int64
	}{
		{"WEB_SEARCH_RESULT_COUNT", aux.WebSearchResultCount, &w.WebSearchResultCount},
		{"WEB_SEARCH_CONCURRENT_REQUESTS", aux.WebSearchConcurrentRequests, &w.WebSearchConcurrentRequests},
		{"WEB_FETCH_MAX_CONTENT_LENGTH", aux.WebFetchMaxContentLength, &w.WebFetchMaxContentLength},
		{"WEB_LOADER_CONCURRENT_REQUESTS", aux.WebLoaderConcurrentRequests, &w.WebLoaderConcurrentRequests},
		{"BRAVE_SEARCH_CONTEXT_TOKENS", aux.BraveSearchContextTokens, &w.BraveSearchContextTokens},
		{"PLAYWRIGHT_TIMEOUT", aux.PlaywrightTimeout, &w.PlaywrightTimeout},
	}
	for _, field := range ints {
		v, err := decodeIntOrEmptyString(field.raw)
		if err != nil {
			return fmt.Errorf("%s: %w", field.key, err)
		}
		*field.dst = v
	}
	return nil
}

// WebSearchConfigUpdate holds the web search settings to change. A nil field
// (nil pointer, empty raw JSON) is left out and keeps its current Open WebUI
// value; lists are pointers so an explicit [] can be set.
type WebSearchConfigUpdate struct {
	EnableWebSearch                      *bool           `json:"ENABLE_WEB_SEARCH,omitempty"`
	EnableWebSearchConfirmation          *bool           `json:"ENABLE_WEB_SEARCH_CONFIRMATION,omitempty"`
	WebSearchConfirmationContent         *string         `json:"WEB_SEARCH_CONFIRMATION_CONTENT,omitempty"`
	WebSearchEngine                      *string         `json:"WEB_SEARCH_ENGINE,omitempty"`
	WebSearchTrustEnv                    *bool           `json:"WEB_SEARCH_TRUST_ENV,omitempty"`
	WebSearchResultCount                 *int64          `json:"WEB_SEARCH_RESULT_COUNT,omitempty"`
	WebSearchConcurrentRequests          *int64          `json:"WEB_SEARCH_CONCURRENT_REQUESTS,omitempty"`
	WebSearchDomainFilterList            *[]string       `json:"WEB_SEARCH_DOMAIN_FILTER_LIST,omitempty"`
	WebFetchMaxContentLength             *int64          `json:"WEB_FETCH_MAX_CONTENT_LENGTH,omitempty"`
	WebLoaderConcurrentRequests          *int64          `json:"WEB_LOADER_CONCURRENT_REQUESTS,omitempty"`
	BypassWebSearchEmbeddingAndRetrieval *bool           `json:"BYPASS_WEB_SEARCH_EMBEDDING_AND_RETRIEVAL,omitempty"`
	BypassWebSearchWebLoader             *bool           `json:"BYPASS_WEB_SEARCH_WEB_LOADER,omitempty"`
	OllamaCloudWebSearchAPIKey           *string         `json:"OLLAMA_CLOUD_WEB_SEARCH_API_KEY,omitempty"`
	SearXNGQueryURL                      *string         `json:"SEARXNG_QUERY_URL,omitempty"`
	SearXNGLanguage                      *string         `json:"SEARXNG_LANGUAGE,omitempty"`
	OpenSERPBaseURL                      *string         `json:"OPENSERP_BASE_URL,omitempty"`
	YaCyQueryURL                         *string         `json:"YACY_QUERY_URL,omitempty"`
	YaCyUsername                         *string         `json:"YACY_USERNAME,omitempty"`
	YaCyPassword                         *string         `json:"YACY_PASSWORD,omitempty"`
	GooglePSEAPIKey                      *string         `json:"GOOGLE_PSE_API_KEY,omitempty"`
	GooglePSEEngineID                    *string         `json:"GOOGLE_PSE_ENGINE_ID,omitempty"`
	BraveSearchAPIKey                    *string         `json:"BRAVE_SEARCH_API_KEY,omitempty"`
	BraveSearchContextTokens             *int64          `json:"BRAVE_SEARCH_CONTEXT_TOKENS,omitempty"`
	KagiSearchAPIKey                     *string         `json:"KAGI_SEARCH_API_KEY,omitempty"`
	MojeekSearchAPIKey                   *string         `json:"MOJEEK_SEARCH_API_KEY,omitempty"`
	BochaSearchAPIKey                    *string         `json:"BOCHA_SEARCH_API_KEY,omitempty"`
	SerpstackAPIKey                      *string         `json:"SERPSTACK_API_KEY,omitempty"`
	SerpstackHTTPS                       *bool           `json:"SERPSTACK_HTTPS,omitempty"`
	SerperAPIKey                         *string         `json:"SERPER_API_KEY,omitempty"`
	SerphouseAPIKey                      *string         `json:"SERPHOUSE_API_KEY,omitempty"`
	SerphouseDomain                      *string         `json:"SERPHOUSE_DOMAIN,omitempty"`
	SerplyAPIKey                         *string         `json:"SERPLY_API_KEY,omitempty"`
	DDGSBackend                          *string         `json:"DDGS_BACKEND,omitempty"`
	TavilyAPIKey                         *string         `json:"TAVILY_API_KEY,omitempty"`
	SearchAPIAPIKey                      *string         `json:"SEARCHAPI_API_KEY,omitempty"`
	SearchAPIEngine                      *string         `json:"SEARCHAPI_ENGINE,omitempty"`
	SerpAPIAPIKey                        *string         `json:"SERPAPI_API_KEY,omitempty"`
	SerpAPIEngine                        *string         `json:"SERPAPI_ENGINE,omitempty"`
	JinaAPIKey                           *string         `json:"JINA_API_KEY,omitempty"`
	JinaAPIBaseURL                       *string         `json:"JINA_API_BASE_URL,omitempty"`
	BingSearchV7Endpoint                 *string         `json:"BING_SEARCH_V7_ENDPOINT,omitempty"`
	BingSearchV7SubscriptionKey          *string         `json:"BING_SEARCH_V7_SUBSCRIPTION_KEY,omitempty"`
	ExaAPIKey                            *string         `json:"EXA_API_KEY,omitempty"`
	PerplexityAPIKey                     *string         `json:"PERPLEXITY_API_KEY,omitempty"`
	PerplexityModel                      *string         `json:"PERPLEXITY_MODEL,omitempty"`
	PerplexitySearchContextUsage         *string         `json:"PERPLEXITY_SEARCH_CONTEXT_USAGE,omitempty"`
	PerplexitySearchAPIURL               *string         `json:"PERPLEXITY_SEARCH_API_URL,omitempty"`
	MicrosoftWebIQAPIBaseURL             *string         `json:"MICROSOFT_WEB_IQ_API_BASE_URL,omitempty"`
	MicrosoftWebIQAPIKey                 *string         `json:"MICROSOFT_WEB_IQ_API_KEY,omitempty"`
	MicrosoftWebIQLanguage               *string         `json:"MICROSOFT_WEB_IQ_LANGUAGE,omitempty"`
	SougouAPISID                         *string         `json:"SOUGOU_API_SID,omitempty"`
	SougouAPISK                          *string         `json:"SOUGOU_API_SK,omitempty"`
	WebLoaderEngine                      *string         `json:"WEB_LOADER_ENGINE,omitempty"`
	WebLoaderTimeout                     *string         `json:"WEB_LOADER_TIMEOUT,omitempty"`
	EnableWebLoaderSSLVerification       *bool           `json:"ENABLE_WEB_LOADER_SSL_VERIFICATION,omitempty"`
	PlaywrightWSURL                      *string         `json:"PLAYWRIGHT_WS_URL,omitempty"`
	PlaywrightTimeout                    *int64          `json:"PLAYWRIGHT_TIMEOUT,omitempty"`
	FirecrawlAPIKey                      *string         `json:"FIRECRAWL_API_KEY,omitempty"`
	FirecrawlAPIBaseURL                  *string         `json:"FIRECRAWL_API_BASE_URL,omitempty"`
	FirecrawlTimeout                     *string         `json:"FIRECRAWL_TIMEOUT,omitempty"`
	TavilyExtractDepth                   *string         `json:"TAVILY_EXTRACT_DEPTH,omitempty"`
	ExternalWebSearchURL                 *string         `json:"EXTERNAL_WEB_SEARCH_URL,omitempty"`
	ExternalWebSearchAPIKey              *string         `json:"EXTERNAL_WEB_SEARCH_API_KEY,omitempty"`
	ExternalWebLoaderURL                 *string         `json:"EXTERNAL_WEB_LOADER_URL,omitempty"`
	ExternalWebLoaderAPIKey              *string         `json:"EXTERNAL_WEB_LOADER_API_KEY,omitempty"`
	YouTubeLoaderLanguage                *[]string       `json:"YOUTUBE_LOADER_LANGUAGE,omitempty"`
	YouTubeLoaderProxyURL                *string         `json:"YOUTUBE_LOADER_PROXY_URL,omitempty"`
	YandexWebSearchURL                   *string         `json:"YANDEX_WEB_SEARCH_URL,omitempty"`
	YandexWebSearchAPIKey                *string         `json:"YANDEX_WEB_SEARCH_API_KEY,omitempty"`
	YandexWebSearchConfig                *string         `json:"YANDEX_WEB_SEARCH_CONFIG,omitempty"`
	YouComAPIKey                         *string         `json:"YOUCOM_API_KEY,omitempty"`
	LinkupAPIKey                         *string         `json:"LINKUP_API_KEY,omitempty"`
	LinkupSearchParams                   json.RawMessage `json:"LINKUP_SEARCH_PARAMS,omitempty"`
}

// GetWebSearchConfig retrieves the web search configuration, the `web` object
// of GET /retrieval/config. API keys are returned in plain text.
func (c *Client) GetWebSearchConfig(ctx context.Context) (*WebSearchConfig, error) {
	current, err := c.GetRetrievalConfig(ctx)
	if err != nil {
		return nil, err
	}
	if !isJSONObjectText(current.Web) {
		return nil, errors.New("the retrieval config response has no web search (web) object")
	}
	var web WebSearchConfig
	if err := json.Unmarshal(current.Web, &web); err != nil {
		return nil, fmt.Errorf("decode web search config: %w", err)
	}
	return &web, nil
}

// UpdateWebSearchConfig changes the web search settings set in update and
// keeps every other one.
//
// POST /retrieval/config/update replaces the whole web search configuration
// with the `web` object it is sent: the handler assigns every WebConfig field
// unconditionally (`config.X = form_data.web.X`), so a field left out of the
// object is stored as None (only BRAVE_SEARCH_CONTEXT_TOKENS is skipped when
// None). This method therefore reads the current `web` object, overlays the
// fields set in update, and POSTs the complete object as a web-only request
// (no top-level retrieval keys, which the handler then keeps). Keys this
// client does not model, nulls and secrets are sent back exactly as read.
// The read and the write hold the client's retrieval write lock, so other
// updates through this client cannot interleave, but the sequence is not
// atomic on the server: a web search change made by another client in
// between is overwritten.
//
// An update with no fields set sends nothing at all.
func (c *Client) UpdateWebSearchConfig(ctx context.Context, update WebSearchConfigUpdate) error {
	changes, err := json.Marshal(update)
	if err != nil {
		return err
	}
	var overlay map[string]json.RawMessage
	if err := json.Unmarshal(changes, &overlay); err != nil {
		return err
	}
	if len(overlay) == 0 {
		return nil
	}

	// Hold the retrieval write lock across the read and the write, so no
	// other update from this provider lands in between.
	c.retrievalMu.Lock()
	defer c.retrievalMu.Unlock()
	current, err := c.GetRetrievalConfig(ctx)
	if err != nil {
		return err
	}
	merged, err := mergeWebSearchObject(current.Web, overlay)
	if err != nil {
		return err
	}
	return c.postRetrievalConfig(ctx, RetrievalConfigUpdateForm{Web: merged})
}

// mergeWebSearchObject overlays changes onto the current raw `web` object.
// It refuses to proceed without a current object: sending only the changed
// keys would reset every other web search setting.
//
// WEB_SEARCH_DOMAIN_FILTER_LIST is `list[str | None] = []` in WebConfig (not
// nullable), so a null read back is left out instead of echoed, which would
// be an HTTP 422; Pydantic then applies its default [].
func mergeWebSearchObject(current json.RawMessage, changes map[string]json.RawMessage) (json.RawMessage, error) {
	if !isJSONObjectText(current) {
		return nil, errors.New("the retrieval config response has no web search (web) object; refusing to send a partial one, which would reset every other web search setting")
	}
	var web map[string]json.RawMessage
	if err := json.Unmarshal(current, &web); err != nil {
		return nil, fmt.Errorf("decode current web search config: %w", err)
	}
	if v, ok := web["WEB_SEARCH_DOMAIN_FILTER_LIST"]; ok && bytes.Equal(bytes.TrimSpace(v), []byte("null")) {
		delete(web, "WEB_SEARCH_DOMAIN_FILTER_LIST")
	}
	for k, v := range changes {
		web[k] = v
	}
	return json.Marshal(web)
}

// isJSONObjectText reports whether raw is a JSON object.
func isJSONObjectText(raw json.RawMessage) bool {
	trimmed := bytes.TrimSpace(raw)
	return len(trimmed) > 0 && trimmed[0] == '{'
}
