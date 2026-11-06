package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

// RetrievalConfig is the GET /retrieval/config response (backend/open_webui/
// routers/retrieval.py's get_rag_config). Every field is a pointer, nil slice
// or raw JSON so a key absent on an older server (or served as null) reads as
// nil. The `status` key is a constant `true` and is ignored. The nested `web`
// object is kept as raw JSON so it can be handed to a separate web-search
// model without this type knowing its fields.
//
// Lists keep the null/[] distinction: JSON null decodes to a nil slice, []
// to an empty non-nil slice. The integer fields are decoded leniently (see
// UnmarshalJSON).
type RetrievalConfig struct {
	RAGTemplate                 *string `json:"RAG_TEMPLATE"`
	TopK                        *int64  `json:"-"`
	BypassEmbeddingAndRetrieval *bool   `json:"BYPASS_EMBEDDING_AND_RETRIEVAL"`
	RAGFullContext              *bool   `json:"RAG_FULL_CONTEXT"`

	EnableRAGHybridSearch              *bool    `json:"ENABLE_RAG_HYBRID_SEARCH"`
	EnableRAGHybridSearchEnrichedTexts *bool    `json:"ENABLE_RAG_HYBRID_SEARCH_ENRICHED_TEXTS"`
	TopKReranker                       *int64   `json:"-"`
	RelevanceThreshold                 *float64 `json:"RELEVANCE_THRESHOLD"`
	HybridBM25Weight                   *float64 `json:"HYBRID_BM25_WEIGHT"`

	ContentExtractionEngine                  *string  `json:"CONTENT_EXTRACTION_ENGINE"`
	ContentExtractionSupportedMediaMimeTypes []string `json:"CONTENT_EXTRACTION_SUPPORTED_MEDIA_MIME_TYPES"`
	PDFExtractImages                         *bool    `json:"PDF_EXTRACT_IMAGES"`
	PDFLoaderMode                            *string  `json:"PDF_LOADER_MODE"`

	DatalabMarkerAPIKey                 *string `json:"DATALAB_MARKER_API_KEY"`
	DatalabMarkerAPIBaseURL             *string `json:"DATALAB_MARKER_API_BASE_URL"`
	DatalabMarkerAdditionalConfig       *string `json:"DATALAB_MARKER_ADDITIONAL_CONFIG"`
	DatalabMarkerSkipCache              *bool   `json:"DATALAB_MARKER_SKIP_CACHE"`
	DatalabMarkerForceOCR               *bool   `json:"DATALAB_MARKER_FORCE_OCR"`
	DatalabMarkerPaginate               *bool   `json:"DATALAB_MARKER_PAGINATE"`
	DatalabMarkerStripExistingOCR       *bool   `json:"DATALAB_MARKER_STRIP_EXISTING_OCR"`
	DatalabMarkerDisableImageExtraction *bool   `json:"DATALAB_MARKER_DISABLE_IMAGE_EXTRACTION"`
	DatalabMarkerFormatLines            *bool   `json:"DATALAB_MARKER_FORMAT_LINES"`
	DatalabMarkerUseLLM                 *bool   `json:"DATALAB_MARKER_USE_LLM"`
	DatalabMarkerOutputFormat           *string `json:"DATALAB_MARKER_OUTPUT_FORMAT"`

	ExternalDocumentLoaderURL     *string         `json:"EXTERNAL_DOCUMENT_LOADER_URL"`
	ExternalDocumentLoaderAPIKey  *string         `json:"EXTERNAL_DOCUMENT_LOADER_API_KEY"`
	ExternalDocumentLoaderHeaders json.RawMessage `json:"EXTERNAL_DOCUMENT_LOADER_HEADERS"`

	TikaServerURL                *string         `json:"TIKA_SERVER_URL"`
	DoclingServerURL             *string         `json:"DOCLING_SERVER_URL"`
	DoclingAPIKey                *string         `json:"DOCLING_API_KEY"`
	DoclingParams                json.RawMessage `json:"DOCLING_PARAMS"`
	DocumentIntelligenceEndpoint *string         `json:"DOCUMENT_INTELLIGENCE_ENDPOINT"`
	DocumentIntelligenceKey      *string         `json:"DOCUMENT_INTELLIGENCE_KEY"`
	DocumentIntelligenceModel    *string         `json:"DOCUMENT_INTELLIGENCE_MODEL"`
	MistralOCRAPIBaseURL         *string         `json:"MISTRAL_OCR_API_BASE_URL"`
	MistralOCRAPIKey             *string         `json:"MISTRAL_OCR_API_KEY"`
	MistralOCRUseBase64          *bool           `json:"MISTRAL_OCR_USE_BASE64"`
	PaddleOCRVLBaseURL           *string         `json:"PADDLEOCR_VL_BASE_URL"`
	PaddleOCRVLToken             *string         `json:"PADDLEOCR_VL_TOKEN"`

	MinerUAPIMode        *string         `json:"MINERU_API_MODE"`
	MinerUAPIURL         *string         `json:"MINERU_API_URL"`
	MinerUAPIKey         *string         `json:"MINERU_API_KEY"`
	MinerUAPITimeout     *int64          `json:"-"`
	MinerUParams         json.RawMessage `json:"MINERU_PARAMS"`
	MinerUFileExtensions []string        `json:"MINERU_FILE_EXTENSIONS"`

	RAGRerankingModel          *string `json:"RAG_RERANKING_MODEL"`
	RAGRerankingEngine         *string `json:"RAG_RERANKING_ENGINE"`
	RAGRerankingBatchSize      *int64  `json:"-"`
	RAGExternalRerankerURL     *string `json:"RAG_EXTERNAL_RERANKER_URL"`
	RAGExternalRerankerAPIKey  *string `json:"RAG_EXTERNAL_RERANKER_API_KEY"`
	RAGExternalRerankerTimeout *string `json:"RAG_EXTERNAL_RERANKER_TIMEOUT"`

	TextSplitter                     *string `json:"TEXT_SPLITTER"`
	RAGTokenizerModel                *string `json:"RAG_TOKENIZER_MODEL"`
	EnableMarkdownHeaderTextSplitter *bool   `json:"ENABLE_MARKDOWN_HEADER_TEXT_SPLITTER"`
	ChunkSize                        *int64  `json:"-"`
	ChunkMinSizeTarget               *int64  `json:"-"`
	ChunkOverlap                     *int64  `json:"-"`

	FileMaxSize                *int64   `json:"-"`
	FileMaxCount               *int64   `json:"-"`
	FileImageCompressionWidth  *int64   `json:"-"`
	FileImageCompressionHeight *int64   `json:"-"`
	AllowedFileExtensions      []string `json:"ALLOWED_FILE_EXTENSIONS"`

	EnableGoogleDriveIntegration *bool `json:"ENABLE_GOOGLE_DRIVE_INTEGRATION"`
	EnableOneDriveIntegration    *bool `json:"ENABLE_ONEDRIVE_INTEGRATION"`

	// Web is the raw nested `web` (WebConfig) object.
	Web json.RawMessage `json:"web"`
}

// UnmarshalJSON decodes every integer field leniently via
// decodeIntOrEmptyString (a JSON number, a numeric string, or ""/null as
// nil). This is required, not defensive: on Open WebUI v0.11.0 the
// MINERU_API_TIMEOUT env default is the string "300" (it is int-typed only in
// the update form since v0.10.0, and str-typed on v0.9.6), and the four file
// limits are Union[int, str | None] in the update form, so a string sent by
// another client is stored and served verbatim.
func (f *RetrievalConfig) UnmarshalJSON(data []byte) error {
	type alias RetrievalConfig
	aux := struct {
		*alias
		TopK                       json.RawMessage `json:"TOP_K"`
		TopKReranker               json.RawMessage `json:"TOP_K_RERANKER"`
		MinerUAPITimeout           json.RawMessage `json:"MINERU_API_TIMEOUT"`
		RAGRerankingBatchSize      json.RawMessage `json:"RAG_RERANKING_BATCH_SIZE"`
		ChunkSize                  json.RawMessage `json:"CHUNK_SIZE"`
		ChunkMinSizeTarget         json.RawMessage `json:"CHUNK_MIN_SIZE_TARGET"`
		ChunkOverlap               json.RawMessage `json:"CHUNK_OVERLAP"`
		FileMaxSize                json.RawMessage `json:"FILE_MAX_SIZE"`
		FileMaxCount               json.RawMessage `json:"FILE_MAX_COUNT"`
		FileImageCompressionWidth  json.RawMessage `json:"FILE_IMAGE_COMPRESSION_WIDTH"`
		FileImageCompressionHeight json.RawMessage `json:"FILE_IMAGE_COMPRESSION_HEIGHT"`
	}{alias: (*alias)(f)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	ints := []struct {
		key string
		raw json.RawMessage
		dst **int64
	}{
		{"TOP_K", aux.TopK, &f.TopK},
		{"TOP_K_RERANKER", aux.TopKReranker, &f.TopKReranker},
		{"MINERU_API_TIMEOUT", aux.MinerUAPITimeout, &f.MinerUAPITimeout},
		{"RAG_RERANKING_BATCH_SIZE", aux.RAGRerankingBatchSize, &f.RAGRerankingBatchSize},
		{"CHUNK_SIZE", aux.ChunkSize, &f.ChunkSize},
		{"CHUNK_MIN_SIZE_TARGET", aux.ChunkMinSizeTarget, &f.ChunkMinSizeTarget},
		{"CHUNK_OVERLAP", aux.ChunkOverlap, &f.ChunkOverlap},
		{"FILE_MAX_SIZE", aux.FileMaxSize, &f.FileMaxSize},
		{"FILE_MAX_COUNT", aux.FileMaxCount, &f.FileMaxCount},
		{"FILE_IMAGE_COMPRESSION_WIDTH", aux.FileImageCompressionWidth, &f.FileImageCompressionWidth},
		{"FILE_IMAGE_COMPRESSION_HEIGHT", aux.FileImageCompressionHeight, &f.FileImageCompressionHeight},
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

// RetrievalConfigUpdateForm is the ConfigForm body of POST
// /retrieval/config/update. Every ConfigForm field is `X | None = None` and
// the handler keeps the stored value for any field that is None, so the POST
// is a per-field partial merge: nil pointers, nil slices and empty raw JSON
// are omitted and leave the server value unchanged. `web` is different: when
// present it replaces the whole web-search configuration (every WebConfig
// field it omits is stored as None), so it is omitted unless Web is set.
//
// Lists are pointers so that an explicit [] can be sent. The four file limits
// accept a negative value meaning "clear": it is sent as "", which the handler
// stores as None (no limit / no compression). MINERU_API_TIMEOUT is sent as a
// numeric string: v0.9.6 types it str (and would reject a JSON number with
// HTTP 422), while v0.10.0+ types it int and Pydantic's lax mode converts the
// string to an int.
type RetrievalConfigUpdateForm struct {
	RAGTemplate                 *string `json:"RAG_TEMPLATE,omitempty"`
	TopK                        *int64  `json:"TOP_K,omitempty"`
	BypassEmbeddingAndRetrieval *bool   `json:"BYPASS_EMBEDDING_AND_RETRIEVAL,omitempty"`
	RAGFullContext              *bool   `json:"RAG_FULL_CONTEXT,omitempty"`

	EnableRAGHybridSearch              *bool    `json:"ENABLE_RAG_HYBRID_SEARCH,omitempty"`
	EnableRAGHybridSearchEnrichedTexts *bool    `json:"ENABLE_RAG_HYBRID_SEARCH_ENRICHED_TEXTS,omitempty"`
	TopKReranker                       *int64   `json:"TOP_K_RERANKER,omitempty"`
	RelevanceThreshold                 *float64 `json:"RELEVANCE_THRESHOLD,omitempty"`
	HybridBM25Weight                   *float64 `json:"HYBRID_BM25_WEIGHT,omitempty"`

	ContentExtractionEngine                  *string   `json:"CONTENT_EXTRACTION_ENGINE,omitempty"`
	ContentExtractionSupportedMediaMimeTypes *[]string `json:"CONTENT_EXTRACTION_SUPPORTED_MEDIA_MIME_TYPES,omitempty"`
	PDFExtractImages                         *bool     `json:"PDF_EXTRACT_IMAGES,omitempty"`
	PDFLoaderMode                            *string   `json:"PDF_LOADER_MODE,omitempty"`

	DatalabMarkerAPIKey                 *string `json:"DATALAB_MARKER_API_KEY,omitempty"`
	DatalabMarkerAPIBaseURL             *string `json:"DATALAB_MARKER_API_BASE_URL,omitempty"`
	DatalabMarkerAdditionalConfig       *string `json:"DATALAB_MARKER_ADDITIONAL_CONFIG,omitempty"`
	DatalabMarkerSkipCache              *bool   `json:"DATALAB_MARKER_SKIP_CACHE,omitempty"`
	DatalabMarkerForceOCR               *bool   `json:"DATALAB_MARKER_FORCE_OCR,omitempty"`
	DatalabMarkerPaginate               *bool   `json:"DATALAB_MARKER_PAGINATE,omitempty"`
	DatalabMarkerStripExistingOCR       *bool   `json:"DATALAB_MARKER_STRIP_EXISTING_OCR,omitempty"`
	DatalabMarkerDisableImageExtraction *bool   `json:"DATALAB_MARKER_DISABLE_IMAGE_EXTRACTION,omitempty"`
	DatalabMarkerFormatLines            *bool   `json:"DATALAB_MARKER_FORMAT_LINES,omitempty"`
	DatalabMarkerUseLLM                 *bool   `json:"DATALAB_MARKER_USE_LLM,omitempty"`
	DatalabMarkerOutputFormat           *string `json:"DATALAB_MARKER_OUTPUT_FORMAT,omitempty"`

	ExternalDocumentLoaderURL     *string         `json:"EXTERNAL_DOCUMENT_LOADER_URL,omitempty"`
	ExternalDocumentLoaderAPIKey  *string         `json:"EXTERNAL_DOCUMENT_LOADER_API_KEY,omitempty"`
	ExternalDocumentLoaderHeaders json.RawMessage `json:"EXTERNAL_DOCUMENT_LOADER_HEADERS,omitempty"`

	TikaServerURL                *string         `json:"TIKA_SERVER_URL,omitempty"`
	DoclingServerURL             *string         `json:"DOCLING_SERVER_URL,omitempty"`
	DoclingAPIKey                *string         `json:"DOCLING_API_KEY,omitempty"`
	DoclingParams                json.RawMessage `json:"DOCLING_PARAMS,omitempty"`
	DocumentIntelligenceEndpoint *string         `json:"DOCUMENT_INTELLIGENCE_ENDPOINT,omitempty"`
	DocumentIntelligenceKey      *string         `json:"DOCUMENT_INTELLIGENCE_KEY,omitempty"`
	DocumentIntelligenceModel    *string         `json:"DOCUMENT_INTELLIGENCE_MODEL,omitempty"`
	MistralOCRAPIBaseURL         *string         `json:"MISTRAL_OCR_API_BASE_URL,omitempty"`
	MistralOCRAPIKey             *string         `json:"MISTRAL_OCR_API_KEY,omitempty"`
	MistralOCRUseBase64          *bool           `json:"MISTRAL_OCR_USE_BASE64,omitempty"`
	PaddleOCRVLBaseURL           *string         `json:"PADDLEOCR_VL_BASE_URL,omitempty"`
	PaddleOCRVLToken             *string         `json:"PADDLEOCR_VL_TOKEN,omitempty"`

	MinerUAPIMode        *string         `json:"MINERU_API_MODE,omitempty"`
	MinerUAPIURL         *string         `json:"MINERU_API_URL,omitempty"`
	MinerUAPIKey         *string         `json:"MINERU_API_KEY,omitempty"`
	MinerUAPITimeout     *int64          `json:"-"`
	MinerUParams         json.RawMessage `json:"MINERU_PARAMS,omitempty"`
	MinerUFileExtensions *[]string       `json:"MINERU_FILE_EXTENSIONS,omitempty"`

	RAGRerankingModel          *string `json:"RAG_RERANKING_MODEL,omitempty"`
	RAGRerankingEngine         *string `json:"RAG_RERANKING_ENGINE,omitempty"`
	RAGRerankingBatchSize      *int64  `json:"RAG_RERANKING_BATCH_SIZE,omitempty"`
	RAGExternalRerankerURL     *string `json:"RAG_EXTERNAL_RERANKER_URL,omitempty"`
	RAGExternalRerankerAPIKey  *string `json:"RAG_EXTERNAL_RERANKER_API_KEY,omitempty"`
	RAGExternalRerankerTimeout *string `json:"RAG_EXTERNAL_RERANKER_TIMEOUT,omitempty"`

	TextSplitter                     *string `json:"TEXT_SPLITTER,omitempty"`
	RAGTokenizerModel                *string `json:"RAG_TOKENIZER_MODEL,omitempty"`
	EnableMarkdownHeaderTextSplitter *bool   `json:"ENABLE_MARKDOWN_HEADER_TEXT_SPLITTER,omitempty"`
	ChunkSize                        *int64  `json:"CHUNK_SIZE,omitempty"`
	ChunkMinSizeTarget               *int64  `json:"CHUNK_MIN_SIZE_TARGET,omitempty"`
	ChunkOverlap                     *int64  `json:"CHUNK_OVERLAP,omitempty"`

	FileMaxSize                *int64    `json:"-"`
	FileMaxCount               *int64    `json:"-"`
	FileImageCompressionWidth  *int64    `json:"-"`
	FileImageCompressionHeight *int64    `json:"-"`
	AllowedFileExtensions      *[]string `json:"ALLOWED_FILE_EXTENSIONS,omitempty"`

	EnableGoogleDriveIntegration *bool `json:"ENABLE_GOOGLE_DRIVE_INTEGRATION,omitempty"`
	EnableOneDriveIntegration    *bool `json:"ENABLE_ONEDRIVE_INTEGRATION,omitempty"`

	// Web, when set, is sent as the `web` object and replaces the whole
	// web-search configuration.
	Web json.RawMessage `json:"web,omitempty"`
}

// MarshalJSON adds the specially encoded MINERU_API_TIMEOUT and file-limit
// keys; see the type comment.
func (f RetrievalConfigUpdateForm) MarshalJSON() ([]byte, error) {
	type alias RetrievalConfigUpdateForm
	var timeout *string
	if f.MinerUAPITimeout != nil {
		s := strconv.FormatInt(*f.MinerUAPITimeout, 10)
		timeout = &s
	}
	return json.Marshal(struct {
		alias
		MinerUAPITimeout           *string `json:"MINERU_API_TIMEOUT,omitempty"`
		FileMaxSize                any     `json:"FILE_MAX_SIZE,omitempty"`
		FileMaxCount               any     `json:"FILE_MAX_COUNT,omitempty"`
		FileImageCompressionWidth  any     `json:"FILE_IMAGE_COMPRESSION_WIDTH,omitempty"`
		FileImageCompressionHeight any     `json:"FILE_IMAGE_COMPRESSION_HEIGHT,omitempty"`
	}{
		alias:                      alias(f),
		MinerUAPITimeout:           timeout,
		FileMaxSize:                clearableInt(f.FileMaxSize),
		FileMaxCount:               clearableInt(f.FileMaxCount),
		FileImageCompressionWidth:  clearableInt(f.FileImageCompressionWidth),
		FileImageCompressionHeight: clearableInt(f.FileImageCompressionHeight),
	})
}

// clearableInt maps nil to nil (omitted), a negative value to "" (clear) and
// anything else to the number itself.
func clearableInt(v *int64) any {
	switch {
	case v == nil:
		return nil
	case *v < 0:
		return ""
	default:
		return *v
	}
}

// GetRetrievalConfig retrieves the document retrieval (RAG) configuration,
// including the raw web-search subtree. API keys are returned in plain text.
func (c *Client) GetRetrievalConfig(ctx context.Context) (*RetrievalConfig, error) {
	var resp RetrievalConfig
	if err := c.do(ctx, http.MethodGet, "retrieval/config", nil, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdateRetrievalConfig posts a partial update (see RetrievalConfigUpdateForm).
// The response is discarded: the v0.11.0 handler's response omits several
// keys (ENABLE_RAG_HYBRID_SEARCH_ENRICHED_TEXTS, DATALAB_MARKER_FORMAT_LINES,
// MINERU_FILE_EXTENSIONS, RAG_RERANKING_BATCH_SIZE), so callers must re-read
// with GetRetrievalConfig.
//
// Writes to this endpoint are serialized per client: Open WebUI's handler
// loads every retrieval setting, merges the request and saves every setting
// again, so two overlapping requests (openwebui_retrieval_config and
// openwebui_web_search_config applied in parallel) could write back each
// other's stale values.
//
// A form with nothing set sends nothing: even an empty request makes Open
// WebUI re-save its retrieval settings and, while hybrid search is on, reload
// the reranking model.
func (c *Client) UpdateRetrievalConfig(ctx context.Context, form RetrievalConfigUpdateForm) error {
	encoded, err := json.Marshal(form)
	if err != nil {
		return err
	}
	if bytes.Equal(bytes.TrimSpace(encoded), []byte("{}")) {
		return nil
	}
	c.retrievalMu.Lock()
	defer c.retrievalMu.Unlock()
	return c.postRetrievalConfig(ctx, form, nil)
}

// postRetrievalConfig sends the update and decodes the response into out
// (nil discards it); callers hold retrievalMu.
func (c *Client) postRetrievalConfig(ctx context.Context, form RetrievalConfigUpdateForm, out any) error {
	return c.doModelReload(ctx, http.MethodPost, "retrieval/config/update", form, out)
}
