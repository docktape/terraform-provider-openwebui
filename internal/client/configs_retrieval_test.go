package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

// stockRetrievalResponse mirrors a stock Open WebUI v0.11.0 GET
// /retrieval/config response (RAG_TEMPLATE shortened, web shortened). Note
// MINERU_API_TIMEOUT: the v0.11.0 env default is the string "300", so a
// never-written instance serves a JSON string for an int-typed field.
const stockRetrievalResponse = `{
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
	"DOCLING_PARAMS": {}, "DOCUMENT_INTELLIGENCE_ENDPOINT": "", "DOCUMENT_INTELLIGENCE_KEY": "",
	"DOCUMENT_INTELLIGENCE_MODEL": "prebuilt-layout", "MISTRAL_OCR_API_BASE_URL": "https://api.mistral.ai/v1",
	"MISTRAL_OCR_API_KEY": "", "MISTRAL_OCR_USE_BASE64": false,
	"PADDLEOCR_VL_BASE_URL": "http://localhost:8080", "PADDLEOCR_VL_TOKEN": "",
	"MINERU_API_MODE": "local", "MINERU_API_URL": "http://localhost:8000", "MINERU_API_KEY": "",
	"MINERU_API_TIMEOUT": "300", "MINERU_PARAMS": {}, "MINERU_FILE_EXTENSIONS": ["pdf"],
	"RAG_RERANKING_MODEL": "", "RAG_RERANKING_ENGINE": "", "RAG_RERANKING_BATCH_SIZE": 32,
	"RAG_EXTERNAL_RERANKER_URL": "", "RAG_EXTERNAL_RERANKER_API_KEY": "", "RAG_EXTERNAL_RERANKER_TIMEOUT": "",
	"TEXT_SPLITTER": "", "RAG_TOKENIZER_MODEL": "", "ENABLE_MARKDOWN_HEADER_TEXT_SPLITTER": true,
	"CHUNK_SIZE": 1000, "CHUNK_MIN_SIZE_TARGET": 0, "CHUNK_OVERLAP": 100,
	"FILE_MAX_SIZE": null, "FILE_MAX_COUNT": null,
	"FILE_IMAGE_COMPRESSION_WIDTH": null, "FILE_IMAGE_COMPRESSION_HEIGHT": null,
	"ALLOWED_FILE_EXTENSIONS": [],
	"ENABLE_GOOGLE_DRIVE_INTEGRATION": false, "ENABLE_ONEDRIVE_INTEGRATION": false,
	"web": {"ENABLE_WEB_SEARCH": false, "WEB_SEARCH_ENGINE": ""}
}`

func TestGetRetrievalConfig(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/retrieval/config" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(stockRetrievalResponse))
	})
	c, server := newConfigTestClient(handler)
	defer server.Close()

	got, err := c.GetRetrievalConfig(context.Background())
	if err != nil {
		t.Fatalf("GetRetrievalConfig failed: %v", err)
	}
	if got.TopK == nil || *got.TopK != 3 || got.ChunkSize == nil || *got.ChunkSize != 1000 {
		t.Errorf("unexpected ints: TOP_K=%v CHUNK_SIZE=%v", got.TopK, got.ChunkSize)
	}
	if got.MinerUAPITimeout == nil || *got.MinerUAPITimeout != 300 {
		t.Errorf("expected MINERU_API_TIMEOUT \"300\" decoded as 300, got %v", got.MinerUAPITimeout)
	}
	if got.FileMaxSize != nil || got.FileMaxCount != nil || got.FileImageCompressionWidth != nil || got.FileImageCompressionHeight != nil {
		t.Errorf("expected null file limits to decode as nil")
	}
	if got.HybridBM25Weight == nil || *got.HybridBM25Weight != 0.5 || got.RelevanceThreshold == nil || *got.RelevanceThreshold != 0 {
		t.Errorf("unexpected floats: %v / %v", got.HybridBM25Weight, got.RelevanceThreshold)
	}
	if got.ContentExtractionSupportedMediaMimeTypes != nil {
		t.Errorf("expected null mime types to decode as nil, got %#v", got.ContentExtractionSupportedMediaMimeTypes)
	}
	if got.AllowedFileExtensions == nil || len(got.AllowedFileExtensions) != 0 {
		t.Errorf("expected [] allowed extensions to decode as empty non-nil slice, got %#v", got.AllowedFileExtensions)
	}
	if len(got.MinerUFileExtensions) != 1 || got.MinerUFileExtensions[0] != "pdf" {
		t.Errorf("unexpected MINERU_FILE_EXTENSIONS: %v", got.MinerUFileExtensions)
	}
	if string(got.DoclingParams) != "{}" || string(got.ExternalDocumentLoaderHeaders) != "{}" || string(got.MinerUParams) != "{}" {
		t.Errorf("unexpected raw JSON blobs: %s / %s / %s", got.DoclingParams, got.ExternalDocumentLoaderHeaders, got.MinerUParams)
	}
	if got.EnableMarkdownHeaderTextSplitter == nil || !*got.EnableMarkdownHeaderTextSplitter {
		t.Errorf("expected ENABLE_MARKDOWN_HEADER_TEXT_SPLITTER true")
	}
	if got.TikaServerURL == nil || *got.TikaServerURL != "http://tika:9998" {
		t.Errorf("unexpected TIKA_SERVER_URL: %v", got.TikaServerURL)
	}
	var web map[string]any
	if err := json.Unmarshal(got.Web, &web); err != nil || web["WEB_SEARCH_ENGINE"] != "" {
		t.Errorf("expected raw web subtree to be preserved, got %s (%v)", got.Web, err)
	}
}

// TestGetRetrievalConfigLenientInts covers the int|str shapes the server can
// return: MINERU_API_TIMEOUT as an int (written by v0.10+), the file limits as
// numeric strings (Union[int, str] in ConfigForm) or "".
func TestGetRetrievalConfigLenientInts(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"MINERU_API_TIMEOUT": 120, "FILE_MAX_SIZE": "25", "FILE_MAX_COUNT": "",
			"FILE_IMAGE_COMPRESSION_WIDTH": 800, "TOP_K": "5", "CHUNK_OVERLAP": ""}`))
	})
	c, server := newConfigTestClient(handler)
	defer server.Close()

	got, err := c.GetRetrievalConfig(context.Background())
	if err != nil {
		t.Fatalf("GetRetrievalConfig failed: %v", err)
	}
	if got.MinerUAPITimeout == nil || *got.MinerUAPITimeout != 120 {
		t.Errorf("MINERU_API_TIMEOUT: %v", got.MinerUAPITimeout)
	}
	if got.FileMaxSize == nil || *got.FileMaxSize != 25 {
		t.Errorf("FILE_MAX_SIZE: %v", got.FileMaxSize)
	}
	if got.FileMaxCount != nil {
		t.Errorf("FILE_MAX_COUNT \"\" should be nil, got %v", *got.FileMaxCount)
	}
	if got.FileImageCompressionWidth == nil || *got.FileImageCompressionWidth != 800 {
		t.Errorf("FILE_IMAGE_COMPRESSION_WIDTH: %v", got.FileImageCompressionWidth)
	}
	if got.TopK == nil || *got.TopK != 5 || got.ChunkOverlap != nil {
		t.Errorf("TOP_K/CHUNK_OVERLAP: %v / %v", got.TopK, got.ChunkOverlap)
	}
	if got.RAGTemplate != nil || got.Web != nil {
		t.Errorf("absent keys should decode as nil (older servers)")
	}
}

func TestGetRetrievalConfigRejectsNonNumericInt(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"FILE_MAX_SIZE": "lots"}`))
	})
	c, server := newConfigTestClient(handler)
	defer server.Close()

	if _, err := c.GetRetrievalConfig(context.Background()); err == nil {
		t.Fatal("expected an error for a non-numeric FILE_MAX_SIZE")
	}
}

// TestUpdateRetrievalConfigSendsOnlySetFields checks the partial-merge wire
// shape: unset fields and `web` are omitted (the handler keeps the stored
// value for any field that is None), [] and false and "" are sent,
// MINERU_API_TIMEOUT is sent as a numeric string, and a negative file limit is
// sent as "" (Open WebUI's "clear to no limit").
func TestUpdateRetrievalConfigSendsOnlySetFields(t *testing.T) {
	var body map[string]any
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/retrieval/config/update" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		raw, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		_, _ = w.Write([]byte(stockRetrievalResponse))
	})
	c, server := newConfigTestClient(handler)
	defer server.Close()

	topK := int64(7)
	timeout := int64(120)
	clearLimit := int64(-1)
	maxCount := int64(4)
	falseValue := false
	empty := ""
	weight := 0.25
	none := []string{}
	exts := []string{"pdf", "docx"}
	form := RetrievalConfigUpdateForm{
		TopK:                                     &topK,
		RAGFullContext:                           &falseValue,
		HybridBM25Weight:                         &weight,
		ContentExtractionEngine:                  &empty,
		ContentExtractionSupportedMediaMimeTypes: &none,
		AllowedFileExtensions:                    &exts,
		DoclingParams:                            json.RawMessage(`{"do_ocr":true}`),
		MinerUAPITimeout:                         &timeout,
		FileMaxSize:                              &clearLimit,
		FileMaxCount:                             &maxCount,
	}
	if err := c.UpdateRetrievalConfig(context.Background(), form); err != nil {
		t.Fatalf("UpdateRetrievalConfig failed: %v", err)
	}

	want := map[string]any{
		"TOP_K":                     float64(7),
		"RAG_FULL_CONTEXT":          false,
		"HYBRID_BM25_WEIGHT":        0.25,
		"CONTENT_EXTRACTION_ENGINE": "",
		"CONTENT_EXTRACTION_SUPPORTED_MEDIA_MIME_TYPES": []any{},
		"ALLOWED_FILE_EXTENSIONS":                       []any{"pdf", "docx"},
		"DOCLING_PARAMS":                                map[string]any{"do_ocr": true},
		"MINERU_API_TIMEOUT":                            "120",
		"FILE_MAX_SIZE":                                 "",
		"FILE_MAX_COUNT":                                float64(4),
	}
	if len(body) != len(want) {
		t.Errorf("expected exactly %d keys, got %d: %v", len(want), len(body), body)
	}
	for k, v := range want {
		gotJSON, _ := json.Marshal(body[k])
		wantJSON, _ := json.Marshal(v)
		if string(gotJSON) != string(wantJSON) {
			t.Errorf("%s: got %s, want %s", k, gotJSON, wantJSON)
		}
	}
	if _, ok := body["web"]; ok {
		t.Errorf("web must not be sent when unset, got %v", body["web"])
	}
}

// TestUpdateRetrievalConfigEmptyFormSendsNothing: a form with nothing set
// makes no request at all (even an empty POST makes Open WebUI re-save its
// settings and, with hybrid search on, reload the reranking model).
func TestUpdateRetrievalConfigEmptyFormSendsNothing(t *testing.T) {
	handler := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
	})
	c, server := newConfigTestClient(handler)
	defer server.Close()

	if err := c.UpdateRetrievalConfig(context.Background(), RetrievalConfigUpdateForm{}); err != nil {
		t.Fatalf("UpdateRetrievalConfig failed: %v", err)
	}
}

// TestUpdateRetrievalConfigWebOnly checks the 13b-facing contract: a form
// carrying only Web sends only the web subtree.
func TestUpdateRetrievalConfigWebOnly(t *testing.T) {
	var body map[string]any
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		_, _ = w.Write([]byte(stockRetrievalResponse))
	})
	c, server := newConfigTestClient(handler)
	defer server.Close()

	form := RetrievalConfigUpdateForm{Web: json.RawMessage(`{"ENABLE_WEB_SEARCH":true}`)}
	if err := c.UpdateRetrievalConfig(context.Background(), form); err != nil {
		t.Fatalf("UpdateRetrievalConfig failed: %v", err)
	}
	if len(body) != 1 {
		t.Errorf("expected only web, got %v", body)
	}
	if web, ok := body["web"].(map[string]any); !ok || web["ENABLE_WEB_SEARCH"] != true {
		t.Errorf("unexpected web: %v", body["web"])
	}
}
