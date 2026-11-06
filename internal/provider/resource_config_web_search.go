package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

var _ resource.Resource = &webSearchConfigResource{}
var _ resource.ResourceWithConfigure = &webSearchConfigResource{}
var _ resource.ResourceWithImportState = &webSearchConfigResource{}

// webLoaderTimeoutValidator mirrors the loaders' float()/int() conversion of
// WEB_LOADER_TIMEOUT: any other value is silently ignored by Open WebUI.
var webLoaderTimeoutValidator = stringvalidator.RegexMatches(
	regexp.MustCompile(`^$|^[0-9]+(\.[0-9]+)?$`),
	"must be empty or a plain decimal number of seconds such as 10 or 2.5 (no sign, exponent or unit; Open WebUI ignores anything else)",
)

// firecrawlTimeoutValidator mirrors the Firecrawl loader's int() conversion
// of FIRECRAWL_TIMEOUT: any other value is silently ignored by Open WebUI.
var firecrawlTimeoutValidator = stringvalidator.RegexMatches(
	regexp.MustCompile(`^[0-9]*$`),
	"must be empty or a plain whole number of seconds such as 30 (no sign, fraction, exponent or unit; Open WebUI ignores anything else)",
)

// emptyOrJSONObjectValidator accepts "" or JSON object text: the Yandex
// engine json.loads() a non-empty YANDEX_WEB_SEARCH_CONFIG and uses it as the
// request body.
var emptyOrJSONObjectValidator = stringvalidator.Any(stringvalidator.OneOf(""), jsonObjectValidator())

type webSearchConfigResource struct {
	client *client.Client
}

type webSearchConfigModel struct {
	ID                                   types.String `tfsdk:"id"`
	EnableWebSearch                      types.Bool   `tfsdk:"enable_web_search"`
	EnableWebSearchConfirmation          types.Bool   `tfsdk:"enable_web_search_confirmation"`
	WebSearchConfirmationContent         types.String `tfsdk:"web_search_confirmation_content"`
	WebSearchEngine                      types.String `tfsdk:"web_search_engine"`
	WebSearchTrustEnv                    types.Bool   `tfsdk:"web_search_trust_env"`
	WebSearchResultCount                 types.Int64  `tfsdk:"web_search_result_count"`
	WebSearchConcurrentRequests          types.Int64  `tfsdk:"web_search_concurrent_requests"`
	WebSearchDomainFilterList            types.List   `tfsdk:"web_search_domain_filter_list"`
	WebFetchMaxContentLength             types.Int64  `tfsdk:"web_fetch_max_content_length"`
	WebLoaderConcurrentRequests          types.Int64  `tfsdk:"web_loader_concurrent_requests"`
	BypassWebSearchEmbeddingAndRetrieval types.Bool   `tfsdk:"bypass_web_search_embedding_and_retrieval"`
	BypassWebSearchWebLoader             types.Bool   `tfsdk:"bypass_web_search_web_loader"`
	OllamaCloudWebSearchAPIKey           types.String `tfsdk:"ollama_cloud_web_search_api_key"`
	SearXNGQueryURL                      types.String `tfsdk:"searxng_query_url"`
	SearXNGLanguage                      types.String `tfsdk:"searxng_language"`
	OpenSERPBaseURL                      types.String `tfsdk:"openserp_base_url"`
	YaCyQueryURL                         types.String `tfsdk:"yacy_query_url"`
	YaCyUsername                         types.String `tfsdk:"yacy_username"`
	YaCyPassword                         types.String `tfsdk:"yacy_password"`
	GooglePSEAPIKey                      types.String `tfsdk:"google_pse_api_key"`
	GooglePSEEngineID                    types.String `tfsdk:"google_pse_engine_id"`
	BraveSearchAPIKey                    types.String `tfsdk:"brave_search_api_key"`
	BraveSearchContextTokens             types.Int64  `tfsdk:"brave_search_context_tokens"`
	KagiSearchAPIKey                     types.String `tfsdk:"kagi_search_api_key"`
	MojeekSearchAPIKey                   types.String `tfsdk:"mojeek_search_api_key"`
	BochaSearchAPIKey                    types.String `tfsdk:"bocha_search_api_key"`
	SerpstackAPIKey                      types.String `tfsdk:"serpstack_api_key"`
	SerpstackHTTPS                       types.Bool   `tfsdk:"serpstack_https"`
	SerperAPIKey                         types.String `tfsdk:"serper_api_key"`
	SerphouseAPIKey                      types.String `tfsdk:"serphouse_api_key"`
	SerphouseDomain                      types.String `tfsdk:"serphouse_domain"`
	SerplyAPIKey                         types.String `tfsdk:"serply_api_key"`
	DDGSBackend                          types.String `tfsdk:"ddgs_backend"`
	TavilyAPIKey                         types.String `tfsdk:"tavily_api_key"`
	SearchAPIAPIKey                      types.String `tfsdk:"searchapi_api_key"`
	SearchAPIEngine                      types.String `tfsdk:"searchapi_engine"`
	SerpAPIAPIKey                        types.String `tfsdk:"serpapi_api_key"`
	SerpAPIEngine                        types.String `tfsdk:"serpapi_engine"`
	JinaAPIKey                           types.String `tfsdk:"jina_api_key"`
	JinaAPIBaseURL                       types.String `tfsdk:"jina_api_base_url"`
	BingSearchV7Endpoint                 types.String `tfsdk:"bing_search_v7_endpoint"`
	BingSearchV7SubscriptionKey          types.String `tfsdk:"bing_search_v7_subscription_key"`
	ExaAPIKey                            types.String `tfsdk:"exa_api_key"`
	PerplexityAPIKey                     types.String `tfsdk:"perplexity_api_key"`
	PerplexityModel                      types.String `tfsdk:"perplexity_model"`
	PerplexitySearchContextUsage         types.String `tfsdk:"perplexity_search_context_usage"`
	PerplexitySearchAPIURL               types.String `tfsdk:"perplexity_search_api_url"`
	MicrosoftWebIQAPIBaseURL             types.String `tfsdk:"microsoft_web_iq_api_base_url"`
	MicrosoftWebIQAPIKey                 types.String `tfsdk:"microsoft_web_iq_api_key"`
	MicrosoftWebIQLanguage               types.String `tfsdk:"microsoft_web_iq_language"`
	SougouAPISID                         types.String `tfsdk:"sougou_api_sid"`
	SougouAPISK                          types.String `tfsdk:"sougou_api_sk"`
	WebLoaderEngine                      types.String `tfsdk:"web_loader_engine"`
	WebLoaderTimeout                     types.String `tfsdk:"web_loader_timeout"`
	EnableWebLoaderSSLVerification       types.Bool   `tfsdk:"enable_web_loader_ssl_verification"`
	PlaywrightWSURL                      types.String `tfsdk:"playwright_ws_url"`
	PlaywrightTimeout                    types.Int64  `tfsdk:"playwright_timeout"`
	FirecrawlAPIKey                      types.String `tfsdk:"firecrawl_api_key"`
	FirecrawlAPIBaseURL                  types.String `tfsdk:"firecrawl_api_base_url"`
	FirecrawlTimeout                     types.String `tfsdk:"firecrawl_timeout"`
	TavilyExtractDepth                   types.String `tfsdk:"tavily_extract_depth"`
	ExternalWebSearchURL                 types.String `tfsdk:"external_web_search_url"`
	ExternalWebSearchAPIKey              types.String `tfsdk:"external_web_search_api_key"`
	ExternalWebLoaderURL                 types.String `tfsdk:"external_web_loader_url"`
	ExternalWebLoaderAPIKey              types.String `tfsdk:"external_web_loader_api_key"`
	YouTubeLoaderLanguage                types.List   `tfsdk:"youtube_loader_language"`
	YouTubeLoaderProxyURL                types.String `tfsdk:"youtube_loader_proxy_url"`
	YandexWebSearchURL                   types.String `tfsdk:"yandex_web_search_url"`
	YandexWebSearchAPIKey                types.String `tfsdk:"yandex_web_search_api_key"`
	YandexWebSearchConfig                types.String `tfsdk:"yandex_web_search_config"`
	YouComAPIKey                         types.String `tfsdk:"youcom_api_key"`
	LinkupAPIKey                         types.String `tfsdk:"linkup_api_key"`
	LinkupSearchParamsJSON               types.String `tfsdk:"linkup_search_params_json"`
}

// NewWebSearchConfigResource constructs the web search config resource.
func NewWebSearchConfigResource() resource.Resource {
	return &webSearchConfigResource{}
}

// Metadata sets the resource type name.
func (r *webSearchConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_web_search_config"
}

// Schema defines the web search config schema.
func (r *webSearchConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	str := func(desc string, validators ...validator.String) schema.StringAttribute {
		return schema.StringAttribute{Optional: true, Computed: true, Description: desc, MarkdownDescription: desc, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}, Validators: validators}
	}
	secret := func(desc string, validators ...validator.String) schema.StringAttribute {
		return schema.StringAttribute{Optional: true, Computed: true, Sensitive: true, Description: desc, MarkdownDescription: desc, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}, Validators: validators}
	}
	b := func(desc string) schema.BoolAttribute {
		return schema.BoolAttribute{Optional: true, Computed: true, Description: desc, MarkdownDescription: desc, PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()}}
	}
	i := func(desc string, validators ...validator.Int64) schema.Int64Attribute {
		return schema.Int64Attribute{Optional: true, Computed: true, Description: desc, MarkdownDescription: desc, PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}, Validators: validators}
	}
	list := func(desc string) schema.ListAttribute {
		return schema.ListAttribute{ElementType: types.StringType, Optional: true, Computed: true, Description: desc, MarkdownDescription: desc, PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()}}
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages Open WebUI's web search settings: the web search toggles, the search engine and its provider credentials, and the web page loader. These are the nested `web` settings of the retrieval config endpoint; the document retrieval (RAG) settings of the same endpoint are managed by `openwebui_retrieval_config` and are never sent by this resource.\n\n" +
			"~> Open WebUI replaces the whole web search configuration on every update. This resource therefore reads the current settings, changes only the attributes set in your configuration, and sends everything else back exactly as read. Any attribute you leave out keeps its current Open WebUI value, and its state shows that value. Removing an attribute from your configuration therefore does not reset it; set it to the value you want instead (`\"\"` for strings, `[]` for lists, `{}` for JSON objects). The read and the write are separate requests: the provider keeps its own updates (including those of `openwebui_retrieval_config`) from running in between, but a web search change made by someone else in between is overwritten.\n\n" +
			"~> The API keys, passwords, `playwright_ws_url` and `youtube_loader_proxy_url` are stored in Terraform state because Open WebUI returns them in plain text. Protect your state accordingly. Of the URL attributes, only `playwright_ws_url` and `youtube_loader_proxy_url` are marked sensitive; the others (such as `searxng_query_url`, `external_web_search_url` and `external_web_loader_url`) are shown in plain text in plans and outputs, so do not embed credentials in them; use the matching `*_api_key` attributes instead.\n\n" +
			"~> If Open WebUI reports a different value during an apply for an attribute you do not set (because it was changed outside Terraform after the last refresh, or by Open WebUI during this apply), the apply leaves it unchanged and keeps the previous state value with a warning; the next refresh shows the server value.\n\n" +
			"~> Every update also makes Open WebUI re-save its retrieval settings and, while hybrid search is enabled, reload the reranking model inside the update request (see `openwebui_retrieval_config`). If that reload fails, Open WebUI turns hybrid search off (`enable_rag_hybrid_search`, managed by `openwebui_retrieval_config`), and the apply warns about it. The update request waits up to 10 minutes for Open WebUI to load the reranking model. If a download takes longer, the apply fails, but Open WebUI still completes the change, so re-running `terraform plan`/`terraform apply` converges. Open WebUI handles no other request while it loads a model, so other Open WebUI resources in the same apply (whose requests keep the provider's 30-second timeout) can time out meanwhile; re-running converges as well. Pre-downloading the model (for example by selecting it once in the Open WebUI admin page) avoids the wait.\n\n" +
			"-> An apply that changes nothing (no attribute configured, or every configured value equal to the current one) sends nothing. `YOUTUBE_LOADER_TRANSLATION` is not managed: Open WebUI keeps it in memory only and does not use it.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Singleton identifier, always `web_search`.",
				MarkdownDescription: "Singleton identifier, always `web_search`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},

			"enable_web_search":                         b("Whether web search is enabled. Server default: `false`."),
			"enable_web_search_confirmation":            b("Whether users must confirm before a query is sent to the web search provider. Server default: `false`." + noteV0100),
			"web_search_confirmation_content":           str("Text of the confirmation prompt shown when `enable_web_search_confirmation` is on. Server default: `Your query will be sent to the configured web search provider.`" + noteV0100),
			"web_search_engine":                         str("Web search engine: one of `searxng`, `yacy`, `google_pse`, `brave`, `brave_llm_context`, `kagi`, `mojeek`, `bocha`, `serpstack`, `serper`, `serphouse`, `serply`, `duckduckgo`, `tavily`, `searchapi`, `serpapi`, `jina`, `bing`, `exa`, `perplexity`, `perplexity_search`, `microsoft_web_iq`, `sougou`, `firecrawl`, `external`, `yandex`, `youcom`, `linkup`, `ollama_cloud` or `openserp` (Open WebUI v0.11.0; older releases support fewer). `azure` exists too, but its settings are environment variables only. Open WebUI does not validate the value. Server default: `\"\"`."),
			"web_search_trust_env":                      b("Whether web search and web loader requests honour the proxy environment variables (`http_proxy`, `https_proxy`, ...). Server default: `true`."),
			"web_search_result_count":                   i("Number of search results used per query. Server default: `3`."),
			"web_search_concurrent_requests":            i("Maximum number of search queries run in parallel; `0` means unlimited. Must be at least `0` (Open WebUI fails every search with a negative value). Server default: `0`.", int64validator.AtLeast(0)),
			"web_search_domain_filter_list":             list("Domains search results are restricted to; an entry starting with `!` blocks that domain instead. `[]` means no filter. Server default: `[]`."),
			"web_fetch_max_content_length":              i("Maximum number of characters the built-in `fetch_url` tool returns; longer pages are truncated. Null (the server default) and `0` both mean no limit; once set, use `0` to remove the limit. Must be at least `0`.", int64validator.AtLeast(0)),
			"web_loader_concurrent_requests":            i("Rate limit of the built-in web loader in requests per second; `0` disables the limit. Must be at least `0`. Server default: `10`.", int64validator.AtLeast(0)),
			"bypass_web_search_embedding_and_retrieval": b("Whether loaded web search results are passed to the model directly instead of being embedded and retrieved. Server default: `false`."),
			"bypass_web_search_web_loader":              b("Whether the search result snippets are used as they are instead of loading the result pages. Server default: `false`."),
			"ollama_cloud_web_search_api_key":           secret("Ollama Cloud API key (`ollama_cloud` engine)."),
			"searxng_query_url":                         str("SearXNG query URL, for example `http://searxng:8080/search?q=<query>`."),
			"searxng_language":                          str("SearXNG search language. Server default: `all`."),
			"openserp_base_url":                         str("OpenSERP base URL (`openserp` engine). Server default: `http://localhost:7000`." + noteV0110),
			"yacy_query_url":                            str("YaCy query URL."),
			"yacy_username":                             str("YaCy username."),
			"yacy_password":                             secret("YaCy password."),
			"google_pse_api_key":                        secret("Google Programmable Search Engine API key."),
			"google_pse_engine_id":                      str("Google Programmable Search Engine ID."),
			"brave_search_api_key":                      secret("Brave Search API key (`brave` and `brave_llm_context` engines)."),
			"brave_search_context_tokens":               i("Maximum number of tokens the `brave_llm_context` engine retrieves (the Open WebUI admin UI offers 1024 to 32768). Server default: `8192`."),
			"kagi_search_api_key":                       secret("Kagi Search API key."),
			"mojeek_search_api_key":                     secret("Mojeek Search API key."),
			"bocha_search_api_key":                      secret("Bocha Search API key."),
			"serpstack_api_key":                         secret("Serpstack API key."),
			"serpstack_https":                           b("Whether Serpstack is called over HTTPS. Server default: `true`."),
			"serper_api_key":                            secret("Serper API key."),
			"serphouse_api_key":                         secret("Serphouse API key." + noteV0100),
			"serphouse_domain":                          str("Search engine domain Serphouse queries. Server default: `google.com`." + noteV0100),
			"serply_api_key":                            secret("Serply API key."),
			"ddgs_backend":                              str("Backend of the `duckduckgo` (DDGS) engine. Server default: `auto`."),
			"tavily_api_key":                            secret("Tavily API key, used by the `tavily` search engine and the `tavily` web loader."),
			"searchapi_api_key":                         secret("SearchApi API key."),
			"searchapi_engine":                          str("SearchApi engine, for example `google`. Empty means the SearchApi default."),
			"serpapi_api_key":                           secret("SerpApi API key."),
			"serpapi_engine":                            str("SerpApi engine, for example `google`. Empty means the SerpApi default."),
			"jina_api_key":                              secret("Jina API key."),
			"jina_api_base_url":                         str("Jina search API base URL. Empty means the hosted service."),
			"bing_search_v7_endpoint":                   str("Bing Search v7 endpoint. Server default: `https://api.bing.microsoft.com/v7.0/search`."),
			"bing_search_v7_subscription_key":           secret("Bing Search v7 subscription key."),
			"exa_api_key":                               secret("Exa API key."),
			"perplexity_api_key":                        secret("Perplexity API key (`perplexity` and `perplexity_search` engines)."),
			"perplexity_model":                          str("Perplexity model (`perplexity` engine). Server default: `sonar`."),
			"perplexity_search_context_usage":           str("Perplexity search context size: `low`, `medium` or `high`. Server default: `medium`."),
			"perplexity_search_api_url":                 str("Perplexity Search API URL (`perplexity_search` engine). Server default: `https://api.perplexity.ai/search`."),
			"microsoft_web_iq_api_base_url":             str("Microsoft Web IQ API base URL (search engine and web loader). Server default: `https://api.microsoft.ai/v3`." + noteV0100),
			"microsoft_web_iq_api_key":                  secret("Microsoft Web IQ API key." + noteV0100),
			"microsoft_web_iq_language":                 str("Microsoft Web IQ language. Server default: `en`." + noteV0100),
			"sougou_api_sid":                            secret("Sougou (Tencent Cloud) SecretId."),
			"sougou_api_sk":                             secret("Sougou (Tencent Cloud) SecretKey."),
			"web_loader_engine":                         str("Web page loader: `\"\"` or `safe_web` (built-in), `playwright`, `firecrawl`, `tavily`, `external` or `microsoft_web_iq`. Open WebUI does not validate the value when saving, but every page load fails with any other value. Server default: `\"\"`."),
			"web_loader_timeout":                        str("Timeout of the built-in and `microsoft_web_iq` loaders in seconds, as a string; `\"\"` means the default. Must be empty or a plain decimal number such as `10` or `2.5` (no sign, exponent or unit; Open WebUI silently ignores anything it cannot convert). Server default: `\"\"`.", webLoaderTimeoutValidator),
			"enable_web_loader_ssl_verification":        b("Whether the web loader verifies TLS certificates. Server default: `true`."),
			"playwright_ws_url":                         secret("WebSocket URL of a remote Playwright browser; empty launches a local one. Sensitive because hosted browsers (such as browserless) take an access token in this URL."),
			"playwright_timeout":                        i("Playwright page load timeout in milliseconds. Server default: `10000`."),
			"firecrawl_api_key":                         secret("Firecrawl API key (search engine and web loader)."),
			"firecrawl_api_base_url":                    str("Firecrawl API base URL. Server default: `https://api.firecrawl.dev`."),
			"firecrawl_timeout":                         str("Firecrawl web loader timeout in whole seconds, as a string; `\"\"` means the default. Must be empty or a plain whole number such as `30` (no sign, fraction, exponent or unit; Open WebUI silently ignores anything it cannot convert). Server default: `\"\"`.", firecrawlTimeoutValidator),
			"tavily_extract_depth":                      str("Tavily web loader extract depth: `basic` or `advanced`. Server default: `basic`."),
			"external_web_search_url":                   str("URL of the external web search service (`external` engine)."),
			"external_web_search_api_key":               secret("API key of the external web search service."),
			"external_web_loader_url":                   str("URL of the external web loader (`external` loader)."),
			"external_web_loader_api_key":               secret("API key of the external web loader."),
			"youtube_loader_language":                   list("Preferred YouTube transcript languages, in order. Open WebUI stores null here when another client saves the web search settings without it; this resource leaves such a null untouched unless you set the attribute. Server default: `[\"en\"]`."),
			"youtube_loader_proxy_url":                  secret("Proxy URL for YouTube transcript requests. Sensitive because proxy URLs commonly embed credentials."),
			"yandex_web_search_url":                     str("Yandex Search API URL. Empty means the public endpoint."),
			"yandex_web_search_api_key":                 secret("Yandex Search API key."),
			"yandex_web_search_config":                  str("Yandex search request template as JSON object text, used as the base of every request body; `\"\"` means none. Stored verbatim as a string, so it is compared exactly. Must be empty or a JSON object.", emptyOrJSONObjectValidator),
			"youcom_api_key":                            secret("You.com API key."),
			"linkup_api_key":                            secret("Linkup API key."),
			"linkup_search_params_json":                 str("JSON object of extra Linkup search parameters, for example `jsonencode({ depth = \"deep\" })`. Server default: `{}`."+retrievalJSONTip+jsonNumberNote, jsonObjectValidator()),
		},
	}
}

// Configure assigns the API client.
func (r *webSearchConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	if c, ok := req.ProviderData.(*client.Client); ok {
		r.client = c
	}
}

// Create applies the configured web search settings.
func (r *webSearchConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing web search config.")
		return
	}
	var config, plan webSearchConfigModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	state, diags := applyWebSearchConfig(ctx, r.client, config, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Read refreshes the web search settings.
func (r *webSearchConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing web search config.")
		return
	}
	var prior webSearchConfigModel
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}
	current, err := r.client.GetWebSearchConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Read web search config failed", err.Error())
		return
	}
	fresh, diags := webSearchConfigModelFromAPI(ctx, *current)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state := preserveWebSearchConfig(prior, fresh)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update applies the configured web search settings.
func (r *webSearchConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing web search config.")
		return
	}
	var config, plan webSearchConfigModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	state, diags := applyWebSearchConfig(ctx, r.client, config, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete removes the resource from state without changing remote configuration.
func (r *webSearchConfigResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing web search config.")
		return
	}
	// Singleton config: destroying the resource leaves the remote config untouched.
}

// ImportState maps import identifiers onto the id attribute.
func (r *webSearchConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// applyWebSearchConfig changes the attributes set in the configuration and
// re-reads the web search settings. The update is built from config, not
// plan: every attribute is Optional+Computed with UseStateForUnknown, so on
// update the plan also carries prior-state values for attributes the
// configuration leaves out, and sending those would revert changes made
// outside Terraform. client.UpdateWebSearchConfig merges the update into the
// current web object (Open WebUI replaces it wholesale), so every attribute
// left out is sent back exactly as Open WebUI currently has it.
//
// The plan is still what the written state must agree with: configured
// attributes are checked by checkWebSearchApplyResult, and unconfigured ones
// are reconciled by reconcileUnconfiguredAttributes.
func applyWebSearchConfig(ctx context.Context, apiClient *client.Client, config, plan webSearchConfigModel) (webSearchConfigModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	update, updateDiags := webSearchConfigUpdateFromConfig(ctx, config)
	diags.Append(updateDiags...)
	if diags.HasError() {
		return webSearchConfigModel{}, diags
	}

	result, err := apiClient.UpdateWebSearchConfig(ctx, update)
	if err != nil {
		diags.AddError("Update web search config failed", err.Error())
		return webSearchConfigModel{}, diags
	}
	if result.HybridSearchTurnedOff {
		diags.AddWarning("Hybrid search was turned off by Open WebUI",
			"This web search update made Open WebUI reload the reranking model (hybrid search was enabled), the reload failed, and Open WebUI turned hybrid search off (ENABLE_RAG_HYBRID_SEARCH). Check the reranking settings of openwebui_retrieval_config and the Open WebUI logs, then enable hybrid search again.")
	}

	current, err := apiClient.GetWebSearchConfig(ctx)
	if err != nil {
		diags.AddError("Read web search config after update failed", err.Error())
		return webSearchConfigModel{}, diags
	}

	fresh, freshDiags := webSearchConfigModelFromAPI(ctx, *current)
	diags.Append(freshDiags...)
	if diags.HasError() {
		return webSearchConfigModel{}, diags
	}

	diags.Append(checkWebSearchApplyResult(config, fresh)...)
	if diags.HasError() {
		return webSearchConfigModel{}, diags
	}
	state, reconcileDiags := reconcileUnconfiguredAttributes(config, plan, preserveWebSearchConfig(plan, fresh),
		"changed outside Terraform since the last refresh, or by Open WebUI during this apply, for example a null domain filter list stored as []")
	diags.Append(reconcileDiags...)
	return state, diags
}

// webSearchConfigUpdateFromConfig maps every attribute set in the
// configuration onto the update. Null (unset) and unknown attributes stay nil
// and keep their current Open WebUI value.
func webSearchConfigUpdateFromConfig(ctx context.Context, m webSearchConfigModel) (client.WebSearchConfigUpdate, diag.Diagnostics) {
	var diags diag.Diagnostics

	rawJSON := func(v types.String, attr string) json.RawMessage {
		if v.IsNull() || v.IsUnknown() {
			return nil
		}
		canonical, err := canonicalJSON([]byte(v.ValueString()))
		if err != nil {
			diags.AddAttributeError(path.Root(attr), "Invalid JSON value", err.Error())
			return nil
		}
		return json.RawMessage(canonical)
	}

	update := client.WebSearchConfigUpdate{
		EnableWebSearch:                      boolPtr(m.EnableWebSearch),
		EnableWebSearchConfirmation:          boolPtr(m.EnableWebSearchConfirmation),
		WebSearchConfirmationContent:         stringPtr(m.WebSearchConfirmationContent),
		WebSearchEngine:                      stringPtr(m.WebSearchEngine),
		WebSearchTrustEnv:                    boolPtr(m.WebSearchTrustEnv),
		WebSearchResultCount:                 int64Ptr(m.WebSearchResultCount),
		WebSearchConcurrentRequests:          int64Ptr(m.WebSearchConcurrentRequests),
		WebSearchDomainFilterList:            expandStringListPtr(ctx, m.WebSearchDomainFilterList, path.Root("web_search_domain_filter_list"), &diags),
		WebFetchMaxContentLength:             int64Ptr(m.WebFetchMaxContentLength),
		WebLoaderConcurrentRequests:          int64Ptr(m.WebLoaderConcurrentRequests),
		BypassWebSearchEmbeddingAndRetrieval: boolPtr(m.BypassWebSearchEmbeddingAndRetrieval),
		BypassWebSearchWebLoader:             boolPtr(m.BypassWebSearchWebLoader),
		OllamaCloudWebSearchAPIKey:           stringPtr(m.OllamaCloudWebSearchAPIKey),
		SearXNGQueryURL:                      stringPtr(m.SearXNGQueryURL),
		SearXNGLanguage:                      stringPtr(m.SearXNGLanguage),
		OpenSERPBaseURL:                      stringPtr(m.OpenSERPBaseURL),
		YaCyQueryURL:                         stringPtr(m.YaCyQueryURL),
		YaCyUsername:                         stringPtr(m.YaCyUsername),
		YaCyPassword:                         stringPtr(m.YaCyPassword),
		GooglePSEAPIKey:                      stringPtr(m.GooglePSEAPIKey),
		GooglePSEEngineID:                    stringPtr(m.GooglePSEEngineID),
		BraveSearchAPIKey:                    stringPtr(m.BraveSearchAPIKey),
		BraveSearchContextTokens:             int64Ptr(m.BraveSearchContextTokens),
		KagiSearchAPIKey:                     stringPtr(m.KagiSearchAPIKey),
		MojeekSearchAPIKey:                   stringPtr(m.MojeekSearchAPIKey),
		BochaSearchAPIKey:                    stringPtr(m.BochaSearchAPIKey),
		SerpstackAPIKey:                      stringPtr(m.SerpstackAPIKey),
		SerpstackHTTPS:                       boolPtr(m.SerpstackHTTPS),
		SerperAPIKey:                         stringPtr(m.SerperAPIKey),
		SerphouseAPIKey:                      stringPtr(m.SerphouseAPIKey),
		SerphouseDomain:                      stringPtr(m.SerphouseDomain),
		SerplyAPIKey:                         stringPtr(m.SerplyAPIKey),
		DDGSBackend:                          stringPtr(m.DDGSBackend),
		TavilyAPIKey:                         stringPtr(m.TavilyAPIKey),
		SearchAPIAPIKey:                      stringPtr(m.SearchAPIAPIKey),
		SearchAPIEngine:                      stringPtr(m.SearchAPIEngine),
		SerpAPIAPIKey:                        stringPtr(m.SerpAPIAPIKey),
		SerpAPIEngine:                        stringPtr(m.SerpAPIEngine),
		JinaAPIKey:                           stringPtr(m.JinaAPIKey),
		JinaAPIBaseURL:                       stringPtr(m.JinaAPIBaseURL),
		BingSearchV7Endpoint:                 stringPtr(m.BingSearchV7Endpoint),
		BingSearchV7SubscriptionKey:          stringPtr(m.BingSearchV7SubscriptionKey),
		ExaAPIKey:                            stringPtr(m.ExaAPIKey),
		PerplexityAPIKey:                     stringPtr(m.PerplexityAPIKey),
		PerplexityModel:                      stringPtr(m.PerplexityModel),
		PerplexitySearchContextUsage:         stringPtr(m.PerplexitySearchContextUsage),
		PerplexitySearchAPIURL:               stringPtr(m.PerplexitySearchAPIURL),
		MicrosoftWebIQAPIBaseURL:             stringPtr(m.MicrosoftWebIQAPIBaseURL),
		MicrosoftWebIQAPIKey:                 stringPtr(m.MicrosoftWebIQAPIKey),
		MicrosoftWebIQLanguage:               stringPtr(m.MicrosoftWebIQLanguage),
		SougouAPISID:                         stringPtr(m.SougouAPISID),
		SougouAPISK:                          stringPtr(m.SougouAPISK),
		WebLoaderEngine:                      stringPtr(m.WebLoaderEngine),
		WebLoaderTimeout:                     stringPtr(m.WebLoaderTimeout),
		EnableWebLoaderSSLVerification:       boolPtr(m.EnableWebLoaderSSLVerification),
		PlaywrightWSURL:                      stringPtr(m.PlaywrightWSURL),
		PlaywrightTimeout:                    int64Ptr(m.PlaywrightTimeout),
		FirecrawlAPIKey:                      stringPtr(m.FirecrawlAPIKey),
		FirecrawlAPIBaseURL:                  stringPtr(m.FirecrawlAPIBaseURL),
		FirecrawlTimeout:                     stringPtr(m.FirecrawlTimeout),
		TavilyExtractDepth:                   stringPtr(m.TavilyExtractDepth),
		ExternalWebSearchURL:                 stringPtr(m.ExternalWebSearchURL),
		ExternalWebSearchAPIKey:              stringPtr(m.ExternalWebSearchAPIKey),
		ExternalWebLoaderURL:                 stringPtr(m.ExternalWebLoaderURL),
		ExternalWebLoaderAPIKey:              stringPtr(m.ExternalWebLoaderAPIKey),
		YouTubeLoaderLanguage:                expandStringListPtr(ctx, m.YouTubeLoaderLanguage, path.Root("youtube_loader_language"), &diags),
		YouTubeLoaderProxyURL:                stringPtr(m.YouTubeLoaderProxyURL),
		YandexWebSearchURL:                   stringPtr(m.YandexWebSearchURL),
		YandexWebSearchAPIKey:                stringPtr(m.YandexWebSearchAPIKey),
		YandexWebSearchConfig:                stringPtr(m.YandexWebSearchConfig),
		YouComAPIKey:                         stringPtr(m.YouComAPIKey),
		LinkupAPIKey:                         stringPtr(m.LinkupAPIKey),
		LinkupSearchParams:                   rawJSON(m.LinkupSearchParamsJSON, "linkup_search_params_json"),
	}
	return update, diags
}

// webSearchConfigModelFromAPI flattens the web search settings. Absent or
// null keys become null attributes; null lists stay null and [] stays an
// empty list.
func webSearchConfigModelFromAPI(ctx context.Context, c client.WebSearchConfig) (webSearchConfigModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	list := func(values []string) types.List {
		l, d := flattenNullableStringSlice(ctx, values)
		diags.Append(d...)
		return l
	}

	m := webSearchConfigModel{
		ID:                                   types.StringValue("web_search"),
		EnableWebSearch:                      boolValueOrNull(c.EnableWebSearch),
		EnableWebSearchConfirmation:          boolValueOrNull(c.EnableWebSearchConfirmation),
		WebSearchConfirmationContent:         stringValueOrNull(c.WebSearchConfirmationContent),
		WebSearchEngine:                      stringValueOrNull(c.WebSearchEngine),
		WebSearchTrustEnv:                    boolValueOrNull(c.WebSearchTrustEnv),
		WebSearchResultCount:                 int64ValueOrNull(c.WebSearchResultCount),
		WebSearchConcurrentRequests:          int64ValueOrNull(c.WebSearchConcurrentRequests),
		WebSearchDomainFilterList:            list(c.WebSearchDomainFilterList),
		WebFetchMaxContentLength:             int64ValueOrNull(c.WebFetchMaxContentLength),
		WebLoaderConcurrentRequests:          int64ValueOrNull(c.WebLoaderConcurrentRequests),
		BypassWebSearchEmbeddingAndRetrieval: boolValueOrNull(c.BypassWebSearchEmbeddingAndRetrieval),
		BypassWebSearchWebLoader:             boolValueOrNull(c.BypassWebSearchWebLoader),
		OllamaCloudWebSearchAPIKey:           stringValueOrNull(c.OllamaCloudWebSearchAPIKey),
		SearXNGQueryURL:                      stringValueOrNull(c.SearXNGQueryURL),
		SearXNGLanguage:                      stringValueOrNull(c.SearXNGLanguage),
		OpenSERPBaseURL:                      stringValueOrNull(c.OpenSERPBaseURL),
		YaCyQueryURL:                         stringValueOrNull(c.YaCyQueryURL),
		YaCyUsername:                         stringValueOrNull(c.YaCyUsername),
		YaCyPassword:                         stringValueOrNull(c.YaCyPassword),
		GooglePSEAPIKey:                      stringValueOrNull(c.GooglePSEAPIKey),
		GooglePSEEngineID:                    stringValueOrNull(c.GooglePSEEngineID),
		BraveSearchAPIKey:                    stringValueOrNull(c.BraveSearchAPIKey),
		BraveSearchContextTokens:             int64ValueOrNull(c.BraveSearchContextTokens),
		KagiSearchAPIKey:                     stringValueOrNull(c.KagiSearchAPIKey),
		MojeekSearchAPIKey:                   stringValueOrNull(c.MojeekSearchAPIKey),
		BochaSearchAPIKey:                    stringValueOrNull(c.BochaSearchAPIKey),
		SerpstackAPIKey:                      stringValueOrNull(c.SerpstackAPIKey),
		SerpstackHTTPS:                       boolValueOrNull(c.SerpstackHTTPS),
		SerperAPIKey:                         stringValueOrNull(c.SerperAPIKey),
		SerphouseAPIKey:                      stringValueOrNull(c.SerphouseAPIKey),
		SerphouseDomain:                      stringValueOrNull(c.SerphouseDomain),
		SerplyAPIKey:                         stringValueOrNull(c.SerplyAPIKey),
		DDGSBackend:                          stringValueOrNull(c.DDGSBackend),
		TavilyAPIKey:                         stringValueOrNull(c.TavilyAPIKey),
		SearchAPIAPIKey:                      stringValueOrNull(c.SearchAPIAPIKey),
		SearchAPIEngine:                      stringValueOrNull(c.SearchAPIEngine),
		SerpAPIAPIKey:                        stringValueOrNull(c.SerpAPIAPIKey),
		SerpAPIEngine:                        stringValueOrNull(c.SerpAPIEngine),
		JinaAPIKey:                           stringValueOrNull(c.JinaAPIKey),
		JinaAPIBaseURL:                       stringValueOrNull(c.JinaAPIBaseURL),
		BingSearchV7Endpoint:                 stringValueOrNull(c.BingSearchV7Endpoint),
		BingSearchV7SubscriptionKey:          stringValueOrNull(c.BingSearchV7SubscriptionKey),
		ExaAPIKey:                            stringValueOrNull(c.ExaAPIKey),
		PerplexityAPIKey:                     stringValueOrNull(c.PerplexityAPIKey),
		PerplexityModel:                      stringValueOrNull(c.PerplexityModel),
		PerplexitySearchContextUsage:         stringValueOrNull(c.PerplexitySearchContextUsage),
		PerplexitySearchAPIURL:               stringValueOrNull(c.PerplexitySearchAPIURL),
		MicrosoftWebIQAPIBaseURL:             stringValueOrNull(c.MicrosoftWebIQAPIBaseURL),
		MicrosoftWebIQAPIKey:                 stringValueOrNull(c.MicrosoftWebIQAPIKey),
		MicrosoftWebIQLanguage:               stringValueOrNull(c.MicrosoftWebIQLanguage),
		SougouAPISID:                         stringValueOrNull(c.SougouAPISID),
		SougouAPISK:                          stringValueOrNull(c.SougouAPISK),
		WebLoaderEngine:                      stringValueOrNull(c.WebLoaderEngine),
		WebLoaderTimeout:                     stringValueOrNull(c.WebLoaderTimeout),
		EnableWebLoaderSSLVerification:       boolValueOrNull(c.EnableWebLoaderSSLVerification),
		PlaywrightWSURL:                      stringValueOrNull(c.PlaywrightWSURL),
		PlaywrightTimeout:                    int64ValueOrNull(c.PlaywrightTimeout),
		FirecrawlAPIKey:                      stringValueOrNull(c.FirecrawlAPIKey),
		FirecrawlAPIBaseURL:                  stringValueOrNull(c.FirecrawlAPIBaseURL),
		FirecrawlTimeout:                     stringValueOrNull(c.FirecrawlTimeout),
		TavilyExtractDepth:                   stringValueOrNull(c.TavilyExtractDepth),
		ExternalWebSearchURL:                 stringValueOrNull(c.ExternalWebSearchURL),
		ExternalWebSearchAPIKey:              stringValueOrNull(c.ExternalWebSearchAPIKey),
		ExternalWebLoaderURL:                 stringValueOrNull(c.ExternalWebLoaderURL),
		ExternalWebLoaderAPIKey:              stringValueOrNull(c.ExternalWebLoaderAPIKey),
		YouTubeLoaderLanguage:                list(c.YouTubeLoaderLanguage),
		YouTubeLoaderProxyURL:                stringValueOrNull(c.YouTubeLoaderProxyURL),
		YandexWebSearchURL:                   stringValueOrNull(c.YandexWebSearchURL),
		YandexWebSearchAPIKey:                stringValueOrNull(c.YandexWebSearchAPIKey),
		YandexWebSearchConfig:                stringValueOrNull(c.YandexWebSearchConfig),
		YouComAPIKey:                         stringValueOrNull(c.YouComAPIKey),
		LinkupAPIKey:                         stringValueOrNull(c.LinkupAPIKey),
		LinkupSearchParamsJSON:               rawJSONToState(c.LinkupSearchParams),
	}
	return m, diags
}

// preserveWebSearchConfig keeps the preferred (plan or prior state) JSON text
// when the server value is semantically equal.
func preserveWebSearchConfig(preferred, fresh webSearchConfigModel) webSearchConfigModel {
	fresh.LinkupSearchParamsJSON = preserveEquivalentJSON(preferred.LinkupSearchParamsJSON, fresh.LinkupSearchParamsJSON)
	return fresh
}

// checkWebSearchApplyResult turns a configured setting that the server does
// not know (an older Open WebUI release ignores unknown web keys and reads
// them back as null) into a clear error, instead of Terraform's generic
// "inconsistent result after apply". Attributes the configuration leaves out
// are never checked here.
func checkWebSearchApplyResult(config, fresh webSearchConfigModel) diag.Diagnostics {
	var diags diag.Diagnostics
	known := func(v attr.Value) bool { return !v.IsNull() && !v.IsUnknown() }
	for _, c := range []struct {
		name    string
		planned bool
		missing bool
		version string
	}{
		{"enable_web_search_confirmation", known(config.EnableWebSearchConfirmation), fresh.EnableWebSearchConfirmation.IsNull(), "v0.10.0"},
		{"web_search_confirmation_content", known(config.WebSearchConfirmationContent), fresh.WebSearchConfirmationContent.IsNull(), "v0.10.0"},
		{"serphouse_api_key", known(config.SerphouseAPIKey), fresh.SerphouseAPIKey.IsNull(), "v0.10.0"},
		{"serphouse_domain", known(config.SerphouseDomain), fresh.SerphouseDomain.IsNull(), "v0.10.0"},
		{"microsoft_web_iq_api_base_url", known(config.MicrosoftWebIQAPIBaseURL), fresh.MicrosoftWebIQAPIBaseURL.IsNull(), "v0.10.0"},
		{"microsoft_web_iq_api_key", known(config.MicrosoftWebIQAPIKey), fresh.MicrosoftWebIQAPIKey.IsNull(), "v0.10.0"},
		{"microsoft_web_iq_language", known(config.MicrosoftWebIQLanguage), fresh.MicrosoftWebIQLanguage.IsNull(), "v0.10.0"},
		{"openserp_base_url", known(config.OpenSERPBaseURL), fresh.OpenSERPBaseURL.IsNull(), "v0.11.0"},
	} {
		if c.planned && c.missing {
			diags.AddAttributeError(path.Root(c.name), "Setting not supported by this Open WebUI version",
				fmt.Sprintf("%s was added in Open WebUI %s; this server ignored it. Remove it from the configuration or upgrade Open WebUI.", c.name, c.version))
		}
	}
	return diags
}
