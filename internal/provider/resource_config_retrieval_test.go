package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

func retrievalHCL(attrs string) string {
	return `
resource "openwebui_retrieval_config" "test" {
` + attrs + `
}
`
}

// stockRetrievalJSON is a stock Open WebUI v0.11.0 GET /retrieval/config body
// (RAG_TEMPLATE shortened, web shortened, one secret populated).
const stockRetrievalJSON = `{
	"status": true,
	"RAG_TEMPLATE": "### Task:", "TOP_K": 3, "BYPASS_EMBEDDING_AND_RETRIEVAL": false, "RAG_FULL_CONTEXT": false,
	"ENABLE_RAG_HYBRID_SEARCH": false, "ENABLE_RAG_HYBRID_SEARCH_ENRICHED_TEXTS": false,
	"TOP_K_RERANKER": 3, "RELEVANCE_THRESHOLD": 0.0, "HYBRID_BM25_WEIGHT": 0.5,
	"CONTENT_EXTRACTION_ENGINE": "", "CONTENT_EXTRACTION_SUPPORTED_MEDIA_MIME_TYPES": null,
	"PDF_EXTRACT_IMAGES": false, "PDF_LOADER_MODE": "page",
	"DATALAB_MARKER_API_KEY": "", "DATALAB_MARKER_API_BASE_URL": "", "DATALAB_MARKER_ADDITIONAL_CONFIG": "",
	"DATALAB_MARKER_SKIP_CACHE": false, "DATALAB_MARKER_FORCE_OCR": false, "DATALAB_MARKER_PAGINATE": false,
	"DATALAB_MARKER_STRIP_EXISTING_OCR": false, "DATALAB_MARKER_DISABLE_IMAGE_EXTRACTION": false,
	"DATALAB_MARKER_FORMAT_LINES": false, "DATALAB_MARKER_USE_LLM": false, "DATALAB_MARKER_OUTPUT_FORMAT": "markdown",
	"EXTERNAL_DOCUMENT_LOADER_URL": "", "EXTERNAL_DOCUMENT_LOADER_API_KEY": "", "EXTERNAL_DOCUMENT_LOADER_HEADERS": {},
	"TIKA_SERVER_URL": "http://tika:9998", "DOCLING_SERVER_URL": "http://docling:5001", "DOCLING_API_KEY": "",
	"DOCLING_PARAMS": {"do_ocr": true, "lang": ["en"]}, "DOCUMENT_INTELLIGENCE_ENDPOINT": "", "DOCUMENT_INTELLIGENCE_KEY": "",
	"DOCUMENT_INTELLIGENCE_MODEL": "prebuilt-layout", "MISTRAL_OCR_API_BASE_URL": "https://api.mistral.ai/v1",
	"MISTRAL_OCR_API_KEY": "existing-secret", "MISTRAL_OCR_USE_BASE64": false,
	"PADDLEOCR_VL_BASE_URL": "http://localhost:8080", "PADDLEOCR_VL_TOKEN": "",
	"MINERU_API_MODE": "local", "MINERU_API_URL": "http://localhost:8000", "MINERU_API_KEY": "",
	"MINERU_API_TIMEOUT": "300", "MINERU_PARAMS": {}, "MINERU_FILE_EXTENSIONS": ["pdf"],
	"RAG_RERANKING_MODEL": "", "RAG_RERANKING_ENGINE": "", "RAG_RERANKING_BATCH_SIZE": 32,
	"RAG_EXTERNAL_RERANKER_URL": "", "RAG_EXTERNAL_RERANKER_API_KEY": "", "RAG_EXTERNAL_RERANKER_TIMEOUT": "",
	"TEXT_SPLITTER": "", "RAG_TOKENIZER_MODEL": "", "ENABLE_MARKDOWN_HEADER_TEXT_SPLITTER": true,
	"CHUNK_SIZE": 1000, "CHUNK_MIN_SIZE_TARGET": 0, "CHUNK_OVERLAP": 100,
	"FILE_MAX_SIZE": null, "FILE_MAX_COUNT": 5,
	"FILE_IMAGE_COMPRESSION_WIDTH": null, "FILE_IMAGE_COMPRESSION_HEIGHT": null,
	"ALLOWED_FILE_EXTENSIONS": [],
	"ENABLE_GOOGLE_DRIVE_INTEGRATION": false, "ENABLE_ONEDRIVE_INTEGRATION": false,
	"web": {"ENABLE_WEB_SEARCH": true, "WEB_SEARCH_ENGINE": "searxng"}
}`

func decodeStockRetrieval(t *testing.T, body string) client.RetrievalConfig {
	t.Helper()
	var c client.RetrievalConfig
	if err := json.Unmarshal([]byte(body), &c); err != nil {
		t.Fatalf("decode stock config: %v", err)
	}
	return c
}

// unknownRetrievalModel returns a model whose every attribute is unknown, as
// in a create plan for a configuration that sets nothing.
func unknownRetrievalModel() retrievalConfigModel {
	var m retrievalConfigModel
	v := reflect.ValueOf(&m).Elem()
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		switch f.Interface().(type) {
		case types.String:
			f.Set(reflect.ValueOf(types.StringUnknown()))
		case types.Bool:
			f.Set(reflect.ValueOf(types.BoolUnknown()))
		case types.Int64:
			f.Set(reflect.ValueOf(types.Int64Unknown()))
		case types.Float64:
			f.Set(reflect.ValueOf(types.Float64Unknown()))
		case types.List:
			f.Set(reflect.ValueOf(types.ListUnknown(types.StringType)))
		}
	}
	return m
}

func stringList(t *testing.T, values ...string) types.List {
	t.Helper()
	elems := make([]attr.Value, len(values))
	for i, v := range values {
		elems[i] = types.StringValue(v)
	}
	return types.ListValueMust(types.StringType, elems)
}

func marshalForm(t *testing.T, form client.RetrievalConfigUpdateForm) map[string]any {
	t.Helper()
	raw, err := json.Marshal(form)
	if err != nil {
		t.Fatalf("marshal form: %v", err)
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("decode form: %v", err)
	}
	return body
}

func TestRetrievalFormFromAllUnknownPlanIsEmpty(t *testing.T) {
	form, diags := retrievalConfigFormFromPlan(context.Background(), unknownRetrievalModel())
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if body := marshalForm(t, form); len(body) != 0 {
		t.Errorf("an all-unknown plan must send nothing (partial merge keeps everything), got %v", body)
	}
}

func TestRetrievalFormFromNullPlanIsEmpty(t *testing.T) {
	// A zero model has every attribute null (e.g. the user removed them).
	form, diags := retrievalConfigFormFromPlan(context.Background(), retrievalConfigModel{})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if body := marshalForm(t, form); len(body) != 0 {
		t.Errorf("an all-null plan must send nothing, got %v", body)
	}
}

func TestRetrievalFormFromKnownValues(t *testing.T) {
	ctx := context.Background()
	m := unknownRetrievalModel()
	m.RAGTemplate = types.StringValue("")
	m.TopK = types.Int64Value(0)
	m.RAGFullContext = types.BoolValue(false)
	m.HybridBM25Weight = types.Float64Value(0.25)
	m.RelevanceThreshold = types.Float64Value(0)
	m.ContentExtractionSupportedMediaMimeTypes = stringList(t)
	m.AllowedFileExtensions = stringList(t, "pdf", "md")
	m.MinerUFileExtensions = types.ListNull(types.StringType)
	m.DoclingParamsJSON = types.StringValue(`{ "lang": ["en"],  "do_ocr": true }`)
	m.ExternalDocumentLoaderHeadersJSON = types.StringValue(`{}`)
	m.MistralOCRAPIKey = types.StringValue("secret")
	m.MinerUAPITimeout = types.Int64Value(120)
	m.FileMaxSize = types.Int64Value(-1)
	m.FileMaxCount = types.Int64Value(10)
	m.RAGExternalRerankerTimeout = types.StringValue("30")

	form, diags := retrievalConfigFormFromPlan(ctx, m)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	body := marshalForm(t, form)
	want := map[string]string{
		"RAG_TEMPLATE":        `""`,
		"TOP_K":               `0`,
		"RAG_FULL_CONTEXT":    `false`,
		"HYBRID_BM25_WEIGHT":  `0.25`,
		"RELEVANCE_THRESHOLD": `0`,
		"CONTENT_EXTRACTION_SUPPORTED_MEDIA_MIME_TYPES": `[]`,
		"ALLOWED_FILE_EXTENSIONS":                       `["pdf","md"]`,
		"DOCLING_PARAMS":                                `{"do_ocr":true,"lang":["en"]}`,
		"EXTERNAL_DOCUMENT_LOADER_HEADERS":              `{}`,
		"MISTRAL_OCR_API_KEY":                           `"secret"`,
		"MINERU_API_TIMEOUT":                            `"120"`,
		"FILE_MAX_SIZE":                                 `""`,
		"FILE_MAX_COUNT":                                `10`,
		"RAG_EXTERNAL_RERANKER_TIMEOUT":                 `"30"`,
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
	if _, ok := body["web"]; ok {
		t.Error("web must never be sent by this resource")
	}
}

func TestRetrievalModelFromAPIShapes(t *testing.T) {
	ctx := context.Background()
	m, diags := retrievalConfigModelFromAPI(ctx, decodeStockRetrieval(t, stockRetrievalJSON))
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if m.ID.ValueString() != "retrieval" {
		t.Errorf("id: %v", m.ID)
	}
	if m.TopK.ValueInt64() != 3 || m.MinerUAPITimeout.ValueInt64() != 300 || m.FileMaxCount.ValueInt64() != 5 {
		t.Errorf("ints: %v %v %v", m.TopK, m.MinerUAPITimeout, m.FileMaxCount)
	}
	if !m.FileMaxSize.IsNull() || !m.FileImageCompressionWidth.IsNull() {
		t.Errorf("null limits must be null: %v %v", m.FileMaxSize, m.FileImageCompressionWidth)
	}
	if m.HybridBM25Weight.ValueFloat64() != 0.5 || m.RelevanceThreshold.ValueFloat64() != 0 {
		t.Errorf("floats: %v %v", m.HybridBM25Weight, m.RelevanceThreshold)
	}
	// null list stays null, [] stays [], populated stays populated.
	if !m.ContentExtractionSupportedMediaMimeTypes.IsNull() {
		t.Errorf("null mime types must be null, got %v", m.ContentExtractionSupportedMediaMimeTypes)
	}
	if m.AllowedFileExtensions.IsNull() || len(m.AllowedFileExtensions.Elements()) != 0 {
		t.Errorf("[] extensions must be an empty list, got %v", m.AllowedFileExtensions)
	}
	if !m.MinerUFileExtensions.Equal(stringList(t, "pdf")) {
		t.Errorf("mineru extensions: %v", m.MinerUFileExtensions)
	}
	if m.DoclingParamsJSON.ValueString() != `{"do_ocr":true,"lang":["en"]}` || m.ExternalDocumentLoaderHeadersJSON.ValueString() != `{}` {
		t.Errorf("json: %v %v", m.DoclingParamsJSON, m.ExternalDocumentLoaderHeadersJSON)
	}
	if m.MistralOCRAPIKey.ValueString() != "existing-secret" || m.TikaServerURL.ValueString() != "http://tika:9998" {
		t.Errorf("strings: %v %v", m.MistralOCRAPIKey, m.TikaServerURL)
	}
	if !m.EnableMarkdownHeaderTextSplitter.ValueBool() {
		t.Error("ENABLE_MARKDOWN_HEADER_TEXT_SPLITTER")
	}
}

// TestRetrievalModelFromOlderServer covers keys missing on Open WebUI v0.9.6
// (no CONTENT_EXTRACTION_SUPPORTED_MEDIA_MIME_TYPES, EXTERNAL_DOCUMENT_LOADER_HEADERS,
// RAG_TOKENIZER_MODEL, MISTRAL_OCR_USE_BASE64) and MINERU_API_TIMEOUT stored as
// a string: absent keys read as null.
func TestRetrievalModelFromOlderServer(t *testing.T) {
	m, diags := retrievalConfigModelFromAPI(context.Background(), decodeStockRetrieval(t, `{"TOP_K": 3, "MINERU_API_TIMEOUT": "60"}`))
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if !m.ExternalDocumentLoaderHeadersJSON.IsNull() || !m.RAGTokenizerModel.IsNull() || !m.MistralOCRUseBase64.IsNull() || !m.ContentExtractionSupportedMediaMimeTypes.IsNull() {
		t.Error("keys absent on older servers must be null")
	}
	if m.MinerUAPITimeout.ValueInt64() != 60 {
		t.Errorf("MINERU_API_TIMEOUT: %v", m.MinerUAPITimeout)
	}
}

func TestRetrievalPreserveJSONAndClearedLimits(t *testing.T) {
	fresh, _ := retrievalConfigModelFromAPI(context.Background(), decodeStockRetrieval(t, stockRetrievalJSON))

	preferred := unknownRetrievalModel()
	preferred.DoclingParamsJSON = types.StringValue(`{ "lang": ["en"], "do_ocr": true }`)
	preferred.FileMaxSize = types.Int64Value(-1)
	preferred.FileImageCompressionWidth = types.Int64Value(-1)
	preferred.FileMaxCount = types.Int64Value(-1) // server has 5: not cleared, so 5 wins

	got := preserveRetrievalConfig(preferred, fresh)
	if got.DoclingParamsJSON.ValueString() != `{ "lang": ["en"], "do_ocr": true }` {
		t.Errorf("equivalent JSON should keep the user's text, got %v", got.DoclingParamsJSON)
	}
	if got.FileMaxSize.ValueInt64() != -1 || got.FileImageCompressionWidth.ValueInt64() != -1 {
		t.Errorf("a cleared (null) limit should keep -1, got %v / %v", got.FileMaxSize, got.FileImageCompressionWidth)
	}
	if got.FileMaxCount.ValueInt64() != 5 {
		t.Errorf("a set limit must show the server value, got %v", got.FileMaxCount)
	}
	if !got.FileImageCompressionHeight.IsNull() {
		t.Errorf("unknown preferred value must leave null, got %v", got.FileImageCompressionHeight)
	}

	// Different JSON: the server value wins.
	preferred.DoclingParamsJSON = types.StringValue(`{"do_ocr": false}`)
	got = preserveRetrievalConfig(preferred, fresh)
	if got.DoclingParamsJSON.ValueString() != `{"do_ocr":true,"lang":["en"]}` {
		t.Errorf("non-equivalent JSON should show the server value, got %v", got.DoclingParamsJSON)
	}
}

func TestRetrievalApplyResultChecks(t *testing.T) {
	fresh, _ := retrievalConfigModelFromAPI(context.Background(), decodeStockRetrieval(t, stockRetrievalJSON))

	plan := unknownRetrievalModel()
	if diags := checkRetrievalApplyResult(plan, fresh); diags.HasError() {
		t.Fatalf("unknown plan must pass: %v", diags)
	}

	plan.EnableRAGHybridSearch = types.BoolValue(true)
	diags := checkRetrievalApplyResult(plan, fresh)
	if !diags.HasError() || !strings.Contains(diags[0].Detail(), "reranking model") {
		t.Errorf("expected a reranker-load error, got %v", diags)
	}

	older, _ := retrievalConfigModelFromAPI(context.Background(), decodeStockRetrieval(t, `{"TOP_K": 3}`))
	plan = unknownRetrievalModel()
	plan.RAGTokenizerModel = types.StringValue("")
	diags = checkRetrievalApplyResult(plan, older)
	if !diags.HasError() || !strings.Contains(diags[0].Detail(), "v0.10.0") {
		t.Errorf("expected a version error, got %v", diags)
	}
}

// TestApplyRetrievalConfigAgainstFakeServer runs the full apply against a fake
// backend: the POST body must hold only the planned keys (never `web`), and
// the resulting state must come from the follow-up GET.
func TestApplyRetrievalConfigAgainstFakeServer(t *testing.T) {
	var posted map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/retrieval/config/update":
			raw, _ := io.ReadAll(r.Body)
			if err := json.Unmarshal(raw, &posted); err != nil {
				t.Errorf("decode POST: %v", err)
			}
			// The real handler's response omits keys; the provider must not rely on it.
			_, _ = w.Write([]byte(`{"status": true}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/retrieval/config":
			_, _ = w.Write([]byte(strings.Replace(stockRetrievalJSON, `"TOP_K": 3,`, `"TOP_K": 8,`, 1)))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	apiClient, err := client.NewClient(server.URL, "test-token", false)
	if err != nil {
		t.Fatal(err)
	}

	plan := unknownRetrievalModel()
	plan.TopK = types.Int64Value(8)
	plan.DoclingParamsJSON = types.StringValue(`{"lang": ["en"], "do_ocr": true}`)

	state, diags := applyRetrievalConfig(context.Background(), apiClient, plan)
	if diags.HasError() {
		t.Fatalf("apply failed: %v", diags)
	}
	if len(posted) != 2 || posted["TOP_K"] != float64(8) || posted["DOCLING_PARAMS"] == nil {
		t.Errorf("unexpected POST body: %v", posted)
	}
	if state.TopK.ValueInt64() != 8 || state.RAGRerankingBatchSize.ValueInt64() != 32 {
		t.Errorf("state must come from the follow-up GET: %v %v", state.TopK, state.RAGRerankingBatchSize)
	}
	if state.DoclingParamsJSON.ValueString() != `{"lang": ["en"], "do_ocr": true}` {
		t.Errorf("planned JSON text should be kept: %v", state.DoclingParamsJSON)
	}
}

func TestRetrievalConfigValidators(t *testing.T) {
	planOnlyValid(t, retrievalHCL(`  file_max_size = -1`))
	planOnlyValid(t, retrievalHCL(`  file_max_count = 0`))
	for _, attr := range []string{"file_max_size", "file_max_count", "file_image_compression_width", "file_image_compression_height"} {
		planOnlyRejected(t, retrievalHCL(`  `+attr+` = -2`), `(?s)`+attr+`.*at\s+least\s+-1`)
	}

	for _, ok := range []string{"", "30"} {
		planOnlyValid(t, retrievalHCL(`  rag_external_reranker_timeout = "`+ok+`"`))
	}
	for _, bad := range []string{"30s", " 30", "-1"} {
		planOnlyRejected(t, retrievalHCL(`  rag_external_reranker_timeout = "`+bad+`"`), `(?s)rag_external_reranker_timeout.*whole\s+number`)
	}

	for _, ok := range []string{"", "gpt2", "org/model name"} {
		planOnlyValid(t, retrievalHCL(`  rag_tokenizer_model = "`+ok+`"`))
	}
	for _, bad := range []string{" gpt2", "gpt2 ", " "} {
		planOnlyRejected(t, retrievalHCL(`  rag_tokenizer_model = "`+bad+`"`), `(?s)rag_tokenizer_model.*whitespace`)
	}

	for _, attr := range []string{"docling_params_json", "mineru_params_json", "external_document_loader_headers_json"} {
		planOnlyValid(t, retrievalHCL(`  `+attr+` = "{}"`))
		for _, bad := range []string{`null`, `[]`, `{`} {
			planOnlyRejected(t, retrievalHCL(`  `+attr+` = "`+bad+`"`), `(?s)`+attr+`.*JSON\s+object`)
		}
	}
}

func TestRetrievalConfigSchemaShape(t *testing.T) {
	var resp resource.SchemaResponse
	NewRetrievalConfigResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if len(resp.Schema.Attributes) != 65 {
		t.Fatalf("expected id plus 64 attributes, got %d", len(resp.Schema.Attributes))
	}
	sensitive := map[string]bool{
		"datalab_marker_api_key": true, "external_document_loader_api_key": true,
		"external_document_loader_headers_json": true, "docling_api_key": true,
		"document_intelligence_key": true, "mistral_ocr_api_key": true, "paddleocr_vl_token": true,
		"mineru_api_key": true, "rag_external_reranker_api_key": true,
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
