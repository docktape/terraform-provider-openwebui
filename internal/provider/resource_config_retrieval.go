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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

var _ resource.Resource = &retrievalConfigResource{}
var _ resource.ResourceWithConfigure = &retrievalConfigResource{}
var _ resource.ResourceWithImportState = &retrievalConfigResource{}

// noEdgeWhitespaceValidator rejects leading or trailing whitespace: the
// backend stores RAG_TOKENIZER_MODEL with .strip(), so such a value would not
// round-trip.
var noEdgeWhitespaceValidator = stringvalidator.RegexMatches(
	regexp.MustCompile(`^$|^\S$|^\S[\s\S]*\S$`),
	"must not start or end with whitespace (Open WebUI strips it)",
)

// rerankerTimeoutValidator mirrors get_rf's int(RAG_EXTERNAL_RERANKER_TIMEOUT)
// conversion: anything but "" or a whole number makes the reranker load fail,
// which silently turns hybrid search off.
var rerankerTimeoutValidator = stringvalidator.RegexMatches(
	regexp.MustCompile(`^[0-9]*$`),
	"must be empty or a whole number of seconds (Open WebUI converts it with int() when loading the external reranker)",
)

// retrievalLimitValidator allows -1 ("no limit"/"no compression", see the attribute
// descriptions) and any value Open WebUI stores verbatim.
var retrievalLimitValidator = int64validator.AtLeast(-1)

type retrievalConfigResource struct {
	client *client.Client
}

type retrievalConfigModel struct {
	ID                                       types.String  `tfsdk:"id"`
	RAGTemplate                              types.String  `tfsdk:"rag_template"`
	TopK                                     types.Int64   `tfsdk:"top_k"`
	BypassEmbeddingAndRetrieval              types.Bool    `tfsdk:"bypass_embedding_and_retrieval"`
	RAGFullContext                           types.Bool    `tfsdk:"rag_full_context"`
	EnableRAGHybridSearch                    types.Bool    `tfsdk:"enable_rag_hybrid_search"`
	EnableRAGHybridSearchEnrichedTexts       types.Bool    `tfsdk:"enable_rag_hybrid_search_enriched_texts"`
	TopKReranker                             types.Int64   `tfsdk:"top_k_reranker"`
	RelevanceThreshold                       types.Float64 `tfsdk:"relevance_threshold"`
	HybridBM25Weight                         types.Float64 `tfsdk:"hybrid_bm25_weight"`
	ContentExtractionEngine                  types.String  `tfsdk:"content_extraction_engine"`
	ContentExtractionSupportedMediaMimeTypes types.List    `tfsdk:"content_extraction_supported_media_mime_types"`
	PDFExtractImages                         types.Bool    `tfsdk:"pdf_extract_images"`
	PDFLoaderMode                            types.String  `tfsdk:"pdf_loader_mode"`
	DatalabMarkerAPIKey                      types.String  `tfsdk:"datalab_marker_api_key"`
	DatalabMarkerAPIBaseURL                  types.String  `tfsdk:"datalab_marker_api_base_url"`
	DatalabMarkerAdditionalConfig            types.String  `tfsdk:"datalab_marker_additional_config"`
	DatalabMarkerSkipCache                   types.Bool    `tfsdk:"datalab_marker_skip_cache"`
	DatalabMarkerForceOCR                    types.Bool    `tfsdk:"datalab_marker_force_ocr"`
	DatalabMarkerPaginate                    types.Bool    `tfsdk:"datalab_marker_paginate"`
	DatalabMarkerStripExistingOCR            types.Bool    `tfsdk:"datalab_marker_strip_existing_ocr"`
	DatalabMarkerDisableImageExtraction      types.Bool    `tfsdk:"datalab_marker_disable_image_extraction"`
	DatalabMarkerFormatLines                 types.Bool    `tfsdk:"datalab_marker_format_lines"`
	DatalabMarkerUseLLM                      types.Bool    `tfsdk:"datalab_marker_use_llm"`
	DatalabMarkerOutputFormat                types.String  `tfsdk:"datalab_marker_output_format"`
	ExternalDocumentLoaderURL                types.String  `tfsdk:"external_document_loader_url"`
	ExternalDocumentLoaderAPIKey             types.String  `tfsdk:"external_document_loader_api_key"`
	ExternalDocumentLoaderHeadersJSON        types.String  `tfsdk:"external_document_loader_headers_json"`
	TikaServerURL                            types.String  `tfsdk:"tika_server_url"`
	DoclingServerURL                         types.String  `tfsdk:"docling_server_url"`
	DoclingAPIKey                            types.String  `tfsdk:"docling_api_key"`
	DoclingParamsJSON                        types.String  `tfsdk:"docling_params_json"`
	DocumentIntelligenceEndpoint             types.String  `tfsdk:"document_intelligence_endpoint"`
	DocumentIntelligenceKey                  types.String  `tfsdk:"document_intelligence_key"`
	DocumentIntelligenceModel                types.String  `tfsdk:"document_intelligence_model"`
	MistralOCRAPIBaseURL                     types.String  `tfsdk:"mistral_ocr_api_base_url"`
	MistralOCRAPIKey                         types.String  `tfsdk:"mistral_ocr_api_key"`
	MistralOCRUseBase64                      types.Bool    `tfsdk:"mistral_ocr_use_base64"`
	PaddleOCRVLBaseURL                       types.String  `tfsdk:"paddleocr_vl_base_url"`
	PaddleOCRVLToken                         types.String  `tfsdk:"paddleocr_vl_token"`
	MinerUAPIMode                            types.String  `tfsdk:"mineru_api_mode"`
	MinerUAPIURL                             types.String  `tfsdk:"mineru_api_url"`
	MinerUAPIKey                             types.String  `tfsdk:"mineru_api_key"`
	MinerUAPITimeout                         types.Int64   `tfsdk:"mineru_api_timeout"`
	MinerUParamsJSON                         types.String  `tfsdk:"mineru_params_json"`
	MinerUFileExtensions                     types.List    `tfsdk:"mineru_file_extensions"`
	RAGRerankingModel                        types.String  `tfsdk:"rag_reranking_model"`
	RAGRerankingEngine                       types.String  `tfsdk:"rag_reranking_engine"`
	RAGRerankingBatchSize                    types.Int64   `tfsdk:"rag_reranking_batch_size"`
	RAGExternalRerankerURL                   types.String  `tfsdk:"rag_external_reranker_url"`
	RAGExternalRerankerAPIKey                types.String  `tfsdk:"rag_external_reranker_api_key"`
	RAGExternalRerankerTimeout               types.String  `tfsdk:"rag_external_reranker_timeout"`
	TextSplitter                             types.String  `tfsdk:"text_splitter"`
	RAGTokenizerModel                        types.String  `tfsdk:"rag_tokenizer_model"`
	EnableMarkdownHeaderTextSplitter         types.Bool    `tfsdk:"enable_markdown_header_text_splitter"`
	ChunkSize                                types.Int64   `tfsdk:"chunk_size"`
	ChunkMinSizeTarget                       types.Int64   `tfsdk:"chunk_min_size_target"`
	ChunkOverlap                             types.Int64   `tfsdk:"chunk_overlap"`
	FileMaxSize                              types.Int64   `tfsdk:"file_max_size"`
	FileMaxCount                             types.Int64   `tfsdk:"file_max_count"`
	FileImageCompressionWidth                types.Int64   `tfsdk:"file_image_compression_width"`
	FileImageCompressionHeight               types.Int64   `tfsdk:"file_image_compression_height"`
	AllowedFileExtensions                    types.List    `tfsdk:"allowed_file_extensions"`
	EnableGoogleDriveIntegration             types.Bool    `tfsdk:"enable_google_drive_integration"`
	EnableOneDriveIntegration                types.Bool    `tfsdk:"enable_onedrive_integration"`
}

// NewRetrievalConfigResource constructs the retrieval (RAG) config resource.
func NewRetrievalConfigResource() resource.Resource {
	return &retrievalConfigResource{}
}

// Metadata sets the resource type name.
func (r *retrievalConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_retrieval_config"
}

const (
	noteV0100        = " Added in Open WebUI v0.10.0; on older servers the value is ignored and reads back as null, so setting it fails the apply."
	noteV0110        = " Added in Open WebUI v0.11.0; on older servers the value is ignored and reads back as null, so setting it fails the apply."
	retrievalJSONTip = " Compared semantically: key order and whitespace differences are not drift. Use `{}` for none (Open WebUI cannot store null here once set)."
	retrievalLimit   = " Set `-1` for no limit: Open WebUI stores it as null, and this resource keeps showing `-1` while the server value stays null. `0` is stored as 0 and is not \"no limit\"."
	retrievalNoComp  = " Set `-1` for no compression: Open WebUI stores it as null, and this resource keeps showing `-1` while the server value stays null."
)

// Schema defines the retrieval config schema.
func (r *retrievalConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
	f := func(desc string) schema.Float64Attribute {
		return schema.Float64Attribute{Optional: true, Computed: true, Description: desc, MarkdownDescription: desc, PlanModifiers: []planmodifier.Float64{float64planmodifier.UseStateForUnknown()}}
	}
	list := func(desc string) schema.ListAttribute {
		return schema.ListAttribute{ElementType: types.StringType, Optional: true, Computed: true, Description: desc, MarkdownDescription: desc, PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()}}
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages Open WebUI's document retrieval (RAG) settings: retrieval and hybrid search, content extraction engines, reranking, chunking, file upload limits and the cloud-drive integration toggles. The nested web search settings of the same endpoint are not managed (and never sent) by this resource.\n\n" +
			"~> Open WebUI's update endpoint merges per field: a setting left out of the request keeps its stored value. This resource only sends the attributes set in your configuration; any attribute you leave out is not sent, so Open WebUI keeps it, and its state shows the current server value. Removing an attribute from your configuration therefore does not reset it; set it to the value you want instead (`\"\"` for strings, `[]` for lists, `{}` for JSON objects, `-1` for the file limits).\n\n" +
			"~> The API keys, tokens, `external_document_loader_headers_json` and `docling_params_json` are stored in Terraform state because Open WebUI returns them in plain text. Protect your state accordingly.\n\n" +
			"~> If Open WebUI reports a different value during an apply for an attribute you do not set (because it was changed outside Terraform after the last refresh, or by Open WebUI itself), the attribute is still not sent; the apply keeps the previous state value with a warning, and the next refresh shows the server value.\n\n" +
			"-> While hybrid search is enabled (and `bypass_embedding_and_retrieval` is false), every update makes Open WebUI (re)load the reranking model inside the update request, which can download it. If loading fails, Open WebUI turns hybrid search off; when your configuration sets `enable_rag_hybrid_search = true` the apply then fails with an error, otherwise it warns. An apply with no attribute configured sends nothing.\n\n" +
			"~> The provider gives up on a request after 30 seconds, while Open WebUI finishes the change on its own: when the model is not cached yet, the apply can fail with `Client.Timeout exceeded` although Open WebUI completes the download and saves the change, and because Open WebUI handles no other request meanwhile, other Open WebUI resources in the same apply can time out too. Pre-download the model (for example by selecting it once in the Open WebUI admin page), or re-run `terraform plan`/`terraform apply`, which converges once the model is loaded.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Singleton identifier, always `retrieval`.",
				MarkdownDescription: "Singleton identifier, always `retrieval`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},

			"rag_template":                   str("Prompt template used to inject retrieved context. Server default: Open WebUI's built-in RAG template."),
			"top_k":                          i("Number of chunks retrieved per query. Server default: `3`."),
			"bypass_embedding_and_retrieval": b("Whether to skip embedding and retrieval and pass whole documents to the model. Server default: `false`."),
			"rag_full_context":               b("Whether to inject the full content of attached documents instead of retrieved chunks. Server default: `false`."),

			"enable_rag_hybrid_search":                b("Whether hybrid (BM25 + vector) search with reranking is enabled. See the note on reranking model loading. Server default: `false`."),
			"enable_rag_hybrid_search_enriched_texts": b("Whether BM25 indexes enriched texts (file name, title, headings) in hybrid search. Server default: `false`."),
			"top_k_reranker":                          i("Number of chunks kept after reranking. Server default: `3`."),
			"relevance_threshold":                     f("Minimum reranker relevance score for a chunk to be kept (0 keeps all). Server default: `0`."),
			"hybrid_bm25_weight":                      f("Weight of BM25 in hybrid search; the vector search weight is `1 - hybrid_bm25_weight`. Server default: `0.5`."),

			"content_extraction_engine":                     str("Document content extraction engine: `\"\"` (built-in loaders), `external`, `tika`, `docling`, `datalab_marker`, `document_intelligence`, `mistral_ocr`, `mineru` or `paddleocr_vl`. Server default: `\"\"`."),
			"content_extraction_supported_media_mime_types": list("MIME type patterns of image/video uploads sent to the content extraction engine. Null (the server default) means only the `external` engine receives media; `[]` means none does. Once set it cannot be returned to null through the API." + noteV0110),
			"pdf_extract_images":                            b("Whether images in PDFs are extracted and OCR'd by the built-in loader. Server default: `false`."),
			"pdf_loader_mode":                               str("Built-in PDF loader mode: `page` or `single`. Server default: `page`."),

			"datalab_marker_api_key":                  secret("Datalab Marker API key."),
			"datalab_marker_api_base_url":             str("Datalab Marker API base URL. Empty means the hosted service."),
			"datalab_marker_additional_config":        str("Additional Datalab Marker options as JSON text. Stored verbatim as a string, so it is compared exactly."),
			"datalab_marker_skip_cache":               b("Datalab Marker: skip the server-side cache. Server default: `false`."),
			"datalab_marker_force_ocr":                b("Datalab Marker: force OCR on every page. Server default: `false`."),
			"datalab_marker_paginate":                 b("Datalab Marker: paginate the output. Server default: `false`."),
			"datalab_marker_strip_existing_ocr":       b("Datalab Marker: strip existing OCR text. Server default: `false`."),
			"datalab_marker_disable_image_extraction": b("Datalab Marker: do not extract images. Server default: `false`."),
			"datalab_marker_format_lines":             b("Datalab Marker: reformat lines. Server default: `false`."),
			"datalab_marker_use_llm":                  b("Datalab Marker: use an LLM to improve accuracy. Server default: `false`."),
			"datalab_marker_output_format":            str("Datalab Marker output format, e.g. `markdown`, `json` or `html`. Server default: `markdown`."),

			"external_document_loader_url":          str("URL of the external document loader."),
			"external_document_loader_api_key":      secret("API key for the external document loader."),
			"external_document_loader_headers_json": secret("JSON object of extra HTTP headers sent to the external document loader, for example `jsonencode({ \"X-Tenant\" = \"a\" })`. Sensitive because headers commonly carry credentials. Server default: `{}`."+retrievalJSONTip+jsonNumberNote+noteV0100, jsonObjectValidator()),

			"tika_server_url":                str("Apache Tika server URL. Server default: `http://tika:9998`."),
			"docling_server_url":             str("Docling server URL. Server default: `http://docling:5001`."),
			"docling_api_key":                secret("Docling API key."),
			"docling_params_json":            secret("JSON object of Docling conversion parameters, for example `jsonencode({ do_ocr = true })`. Sensitive because docling-serve options (such as `picture_description_api` headers) can carry credentials. Server default: `{}`."+retrievalJSONTip+jsonNumberNote, jsonObjectValidator()),
			"document_intelligence_endpoint": str("Azure Document Intelligence endpoint."),
			"document_intelligence_key":      secret("Azure Document Intelligence key."),
			"document_intelligence_model":    str("Azure Document Intelligence model. Server default: `prebuilt-layout`."),
			"mistral_ocr_api_base_url":       str("Mistral OCR API base URL. Server default: `https://api.mistral.ai/v1`."),
			"mistral_ocr_api_key":            secret("Mistral OCR API key."),
			"mistral_ocr_use_base64":         b("Whether documents are sent to Mistral OCR inline as base64 instead of being uploaded. Server default: `false`." + noteV0100),
			"paddleocr_vl_base_url":          str("PaddleOCR-VL server base URL. Server default: `http://localhost:8080`."),
			"paddleocr_vl_token":             secret("PaddleOCR-VL access token."),

			"mineru_api_mode":        str("MinerU API mode: `local` or `cloud`. Server default: `local`."),
			"mineru_api_url":         str("MinerU API URL. Server default: `http://localhost:8000`."),
			"mineru_api_key":         secret("MinerU API key (cloud mode)."),
			"mineru_api_timeout":     i("MinerU request timeout in seconds. Open WebUI v0.9.6 stores it as a string and v0.10.0+ as an integer; both are read as a number. Server default: `300`."),
			"mineru_params_json":     str("JSON object of MinerU parameters. Server default: `{}`."+retrievalJSONTip+jsonNumberNote, jsonObjectValidator()),
			"mineru_file_extensions": list("File extensions routed to MinerU. Server default: `[\"pdf\"]`."),

			"rag_reranking_model":           str("Reranking model used by hybrid search. Empty means none."),
			"rag_reranking_engine":          str("Reranking engine: `\"\"` (local sentence-transformers) or `external`. Server default: `\"\"`."),
			"rag_reranking_batch_size":      i("Batch size for reranking. Server default: `32`."),
			"rag_external_reranker_url":     str("URL of the external reranker."),
			"rag_external_reranker_api_key": secret("API key for the external reranker."),
			"rag_external_reranker_timeout": str("External reranker timeout in whole seconds as a string; `\"\"` means the system default. Server default: `\"\"`.", rerankerTimeoutValidator),

			"text_splitter":                        str("Text splitter: `\"\"`/`character` or `token`. Server default: `\"\"`."),
			"rag_tokenizer_model":                  str("Hugging Face tokenizer used by the `token` splitter. Empty means tiktoken. Must not start or end with whitespace (Open WebUI strips it)."+noteV0100, noEdgeWhitespaceValidator),
			"enable_markdown_header_text_splitter": b("Whether Markdown documents are split on headers first. Server default: `true`."),
			"chunk_size":                           i("Chunk size (characters or tokens, depending on `text_splitter`). Server default: `1000`."),
			"chunk_min_size_target":                i("Chunks smaller than this are merged with neighbours; `0` disables merging. Server default: `0`."),
			"chunk_overlap":                        i("Overlap between consecutive chunks. Server default: `100`."),

			"file_max_size":                 i("Maximum upload size in MB."+retrievalLimit+" Server default: null (no limit).", retrievalLimitValidator),
			"file_max_count":                i("Maximum number of files per upload."+retrievalLimit+" Server default: null (no limit).", retrievalLimitValidator),
			"file_image_compression_width":  i("Width in pixels uploaded images are compressed to."+retrievalNoComp+" Server default: null (no compression).", retrievalLimitValidator),
			"file_image_compression_height": i("Height in pixels uploaded images are compressed to."+retrievalNoComp+" Server default: null (no compression).", retrievalLimitValidator),
			"allowed_file_extensions":       list("Allowed upload file extensions (without dots). `[]` allows all. Server default: `[]`."),

			"enable_google_drive_integration": b("Whether the Google Drive file picker is enabled. Server default: `false`."),
			"enable_onedrive_integration":     b("Whether the OneDrive file picker is enabled. Server default: `false`."),
		},
	}
}

// Configure assigns the API client.
func (r *retrievalConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	if c, ok := req.ProviderData.(*client.Client); ok {
		r.client = c
	}
}

// Create applies the configured retrieval settings.
func (r *retrievalConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing retrieval config.")
		return
	}
	var config, plan retrievalConfigModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	state, diags := applyRetrievalConfig(ctx, r.client, config, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Read refreshes the retrieval settings.
func (r *retrievalConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing retrieval config.")
		return
	}
	var prior retrievalConfigModel
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}
	current, err := r.client.GetRetrievalConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Read retrieval config failed", err.Error())
		return
	}
	fresh, diags := retrievalConfigModelFromAPI(ctx, *current)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state := preserveRetrievalConfig(prior, fresh)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update applies the configured retrieval settings.
func (r *retrievalConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing retrieval config.")
		return
	}
	var config, plan retrievalConfigModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	state, diags := applyRetrievalConfig(ctx, r.client, config, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete removes the resource from state without changing remote configuration.
func (r *retrievalConfigResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing retrieval config.")
		return
	}
	// Singleton config: destroying the resource leaves the remote config untouched.
}

// ImportState maps import identifiers onto the id attribute.
func (r *retrievalConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// applyRetrievalConfig sends the attributes set in the configuration as a
// partial update, then re-reads the configuration (the update response omits
// several keys). It builds the request from config, not plan: every attribute
// is Optional+Computed with UseStateForUnknown, so on update the plan also
// carries prior-state values for attributes the configuration leaves out, and
// sending those would revert changes made outside Terraform. Open WebUI keeps
// every field it is not sent, and `web` is never sent.
//
// The plan is still what the written state must agree with: configured
// attributes are checked by checkRetrievalApplyResult, and unconfigured ones
// are reconciled by reconcileUnconfiguredRetrieval.
func applyRetrievalConfig(ctx context.Context, apiClient *client.Client, config, plan retrievalConfigModel) (retrievalConfigModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	form, formDiags := retrievalConfigFormFromConfig(ctx, config)
	diags.Append(formDiags...)
	if diags.HasError() {
		return retrievalConfigModel{}, diags
	}

	if err := apiClient.UpdateRetrievalConfig(ctx, form); err != nil {
		diags.AddError("Update retrieval config failed", err.Error())
		return retrievalConfigModel{}, diags
	}

	current, err := apiClient.GetRetrievalConfig(ctx)
	if err != nil {
		diags.AddError("Read retrieval config after update failed", err.Error())
		return retrievalConfigModel{}, diags
	}

	fresh, freshDiags := retrievalConfigModelFromAPI(ctx, *current)
	diags.Append(freshDiags...)
	if diags.HasError() {
		return retrievalConfigModel{}, diags
	}

	diags.Append(checkRetrievalApplyResult(config, fresh)...)
	if diags.HasError() {
		return retrievalConfigModel{}, diags
	}
	state, reconcileDiags := reconcileUnconfiguredRetrieval(config, plan, preserveRetrievalConfig(plan, fresh))
	diags.Append(reconcileDiags...)
	return state, diags
}

// reconcileUnconfiguredRetrieval handles attributes the configuration leaves
// out (null in config) whose planned value is known, i.e. copied from the
// prior state by UseStateForUnknown. Terraform requires the applied state to
// match such a planned value, but the server value can legitimately differ:
// it was changed outside Terraform after the last refresh (or with
// -refresh=false), or Open WebUI changed it while applying (it turns
// ENABLE_RAG_HYBRID_SEARCH off when the reranking model fails to load). This
// resource never sent those attributes, so it keeps the planned value in
// state and warns; the next refresh shows the server value.
func reconcileUnconfiguredRetrieval(config, plan, fresh retrievalConfigModel) (retrievalConfigModel, diag.Diagnostics) {
	return reconcileUnconfiguredAttributes(config, plan, fresh,
		"changed outside Terraform since the last refresh, or by Open WebUI during this apply, for example hybrid search turned off after a reranking model load failure")
}

// retrievalConfigFormFromConfig maps every attribute set in the configuration
// onto the update form. Null (unset) and unknown attributes stay nil and are
// omitted from the request, so Open WebUI keeps their stored value. A file
// limit of -1 is sent as "" (stored as null).
func retrievalConfigFormFromConfig(ctx context.Context, m retrievalConfigModel) (client.RetrievalConfigUpdateForm, diag.Diagnostics) {
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

	form := client.RetrievalConfigUpdateForm{
		RAGTemplate:                              stringPtr(m.RAGTemplate),
		TopK:                                     int64Ptr(m.TopK),
		BypassEmbeddingAndRetrieval:              boolPtr(m.BypassEmbeddingAndRetrieval),
		RAGFullContext:                           boolPtr(m.RAGFullContext),
		EnableRAGHybridSearch:                    boolPtr(m.EnableRAGHybridSearch),
		EnableRAGHybridSearchEnrichedTexts:       boolPtr(m.EnableRAGHybridSearchEnrichedTexts),
		TopKReranker:                             int64Ptr(m.TopKReranker),
		RelevanceThreshold:                       float64Ptr(m.RelevanceThreshold),
		HybridBM25Weight:                         float64Ptr(m.HybridBM25Weight),
		ContentExtractionEngine:                  stringPtr(m.ContentExtractionEngine),
		ContentExtractionSupportedMediaMimeTypes: expandStringListPtr(ctx, m.ContentExtractionSupportedMediaMimeTypes, path.Root("content_extraction_supported_media_mime_types"), &diags),
		PDFExtractImages:                         boolPtr(m.PDFExtractImages),
		PDFLoaderMode:                            stringPtr(m.PDFLoaderMode),
		DatalabMarkerAPIKey:                      stringPtr(m.DatalabMarkerAPIKey),
		DatalabMarkerAPIBaseURL:                  stringPtr(m.DatalabMarkerAPIBaseURL),
		DatalabMarkerAdditionalConfig:            stringPtr(m.DatalabMarkerAdditionalConfig),
		DatalabMarkerSkipCache:                   boolPtr(m.DatalabMarkerSkipCache),
		DatalabMarkerForceOCR:                    boolPtr(m.DatalabMarkerForceOCR),
		DatalabMarkerPaginate:                    boolPtr(m.DatalabMarkerPaginate),
		DatalabMarkerStripExistingOCR:            boolPtr(m.DatalabMarkerStripExistingOCR),
		DatalabMarkerDisableImageExtraction:      boolPtr(m.DatalabMarkerDisableImageExtraction),
		DatalabMarkerFormatLines:                 boolPtr(m.DatalabMarkerFormatLines),
		DatalabMarkerUseLLM:                      boolPtr(m.DatalabMarkerUseLLM),
		DatalabMarkerOutputFormat:                stringPtr(m.DatalabMarkerOutputFormat),
		ExternalDocumentLoaderURL:                stringPtr(m.ExternalDocumentLoaderURL),
		ExternalDocumentLoaderAPIKey:             stringPtr(m.ExternalDocumentLoaderAPIKey),
		ExternalDocumentLoaderHeaders:            rawJSON(m.ExternalDocumentLoaderHeadersJSON, "external_document_loader_headers_json"),
		TikaServerURL:                            stringPtr(m.TikaServerURL),
		DoclingServerURL:                         stringPtr(m.DoclingServerURL),
		DoclingAPIKey:                            stringPtr(m.DoclingAPIKey),
		DoclingParams:                            rawJSON(m.DoclingParamsJSON, "docling_params_json"),
		DocumentIntelligenceEndpoint:             stringPtr(m.DocumentIntelligenceEndpoint),
		DocumentIntelligenceKey:                  stringPtr(m.DocumentIntelligenceKey),
		DocumentIntelligenceModel:                stringPtr(m.DocumentIntelligenceModel),
		MistralOCRAPIBaseURL:                     stringPtr(m.MistralOCRAPIBaseURL),
		MistralOCRAPIKey:                         stringPtr(m.MistralOCRAPIKey),
		MistralOCRUseBase64:                      boolPtr(m.MistralOCRUseBase64),
		PaddleOCRVLBaseURL:                       stringPtr(m.PaddleOCRVLBaseURL),
		PaddleOCRVLToken:                         stringPtr(m.PaddleOCRVLToken),
		MinerUAPIMode:                            stringPtr(m.MinerUAPIMode),
		MinerUAPIURL:                             stringPtr(m.MinerUAPIURL),
		MinerUAPIKey:                             stringPtr(m.MinerUAPIKey),
		MinerUAPITimeout:                         int64Ptr(m.MinerUAPITimeout),
		MinerUParams:                             rawJSON(m.MinerUParamsJSON, "mineru_params_json"),
		MinerUFileExtensions:                     expandStringListPtr(ctx, m.MinerUFileExtensions, path.Root("mineru_file_extensions"), &diags),
		RAGRerankingModel:                        stringPtr(m.RAGRerankingModel),
		RAGRerankingEngine:                       stringPtr(m.RAGRerankingEngine),
		RAGRerankingBatchSize:                    int64Ptr(m.RAGRerankingBatchSize),
		RAGExternalRerankerURL:                   stringPtr(m.RAGExternalRerankerURL),
		RAGExternalRerankerAPIKey:                stringPtr(m.RAGExternalRerankerAPIKey),
		RAGExternalRerankerTimeout:               stringPtr(m.RAGExternalRerankerTimeout),
		TextSplitter:                             stringPtr(m.TextSplitter),
		RAGTokenizerModel:                        stringPtr(m.RAGTokenizerModel),
		EnableMarkdownHeaderTextSplitter:         boolPtr(m.EnableMarkdownHeaderTextSplitter),
		ChunkSize:                                int64Ptr(m.ChunkSize),
		ChunkMinSizeTarget:                       int64Ptr(m.ChunkMinSizeTarget),
		ChunkOverlap:                             int64Ptr(m.ChunkOverlap),
		FileMaxSize:                              int64Ptr(m.FileMaxSize),
		FileMaxCount:                             int64Ptr(m.FileMaxCount),
		FileImageCompressionWidth:                int64Ptr(m.FileImageCompressionWidth),
		FileImageCompressionHeight:               int64Ptr(m.FileImageCompressionHeight),
		AllowedFileExtensions:                    expandStringListPtr(ctx, m.AllowedFileExtensions, path.Root("allowed_file_extensions"), &diags),
		EnableGoogleDriveIntegration:             boolPtr(m.EnableGoogleDriveIntegration),
		EnableOneDriveIntegration:                boolPtr(m.EnableOneDriveIntegration),
	}
	return form, diags
}

// retrievalConfigModelFromAPI flattens a GET /retrieval/config response.
// Absent or null keys become null attributes; null lists stay null and []
// stays an empty list.
func retrievalConfigModelFromAPI(ctx context.Context, c client.RetrievalConfig) (retrievalConfigModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	list := func(values []string) types.List {
		l, d := flattenNullableStringSlice(ctx, values)
		diags.Append(d...)
		return l
	}

	m := retrievalConfigModel{
		ID:                                       types.StringValue("retrieval"),
		RAGTemplate:                              stringValueOrNull(c.RAGTemplate),
		TopK:                                     int64ValueOrNull(c.TopK),
		BypassEmbeddingAndRetrieval:              boolValueOrNull(c.BypassEmbeddingAndRetrieval),
		RAGFullContext:                           boolValueOrNull(c.RAGFullContext),
		EnableRAGHybridSearch:                    boolValueOrNull(c.EnableRAGHybridSearch),
		EnableRAGHybridSearchEnrichedTexts:       boolValueOrNull(c.EnableRAGHybridSearchEnrichedTexts),
		TopKReranker:                             int64ValueOrNull(c.TopKReranker),
		RelevanceThreshold:                       float64ValueOrNull(c.RelevanceThreshold),
		HybridBM25Weight:                         float64ValueOrNull(c.HybridBM25Weight),
		ContentExtractionEngine:                  stringValueOrNull(c.ContentExtractionEngine),
		ContentExtractionSupportedMediaMimeTypes: list(c.ContentExtractionSupportedMediaMimeTypes),
		PDFExtractImages:                         boolValueOrNull(c.PDFExtractImages),
		PDFLoaderMode:                            stringValueOrNull(c.PDFLoaderMode),
		DatalabMarkerAPIKey:                      stringValueOrNull(c.DatalabMarkerAPIKey),
		DatalabMarkerAPIBaseURL:                  stringValueOrNull(c.DatalabMarkerAPIBaseURL),
		DatalabMarkerAdditionalConfig:            stringValueOrNull(c.DatalabMarkerAdditionalConfig),
		DatalabMarkerSkipCache:                   boolValueOrNull(c.DatalabMarkerSkipCache),
		DatalabMarkerForceOCR:                    boolValueOrNull(c.DatalabMarkerForceOCR),
		DatalabMarkerPaginate:                    boolValueOrNull(c.DatalabMarkerPaginate),
		DatalabMarkerStripExistingOCR:            boolValueOrNull(c.DatalabMarkerStripExistingOCR),
		DatalabMarkerDisableImageExtraction:      boolValueOrNull(c.DatalabMarkerDisableImageExtraction),
		DatalabMarkerFormatLines:                 boolValueOrNull(c.DatalabMarkerFormatLines),
		DatalabMarkerUseLLM:                      boolValueOrNull(c.DatalabMarkerUseLLM),
		DatalabMarkerOutputFormat:                stringValueOrNull(c.DatalabMarkerOutputFormat),
		ExternalDocumentLoaderURL:                stringValueOrNull(c.ExternalDocumentLoaderURL),
		ExternalDocumentLoaderAPIKey:             stringValueOrNull(c.ExternalDocumentLoaderAPIKey),
		ExternalDocumentLoaderHeadersJSON:        rawJSONToState(c.ExternalDocumentLoaderHeaders),
		TikaServerURL:                            stringValueOrNull(c.TikaServerURL),
		DoclingServerURL:                         stringValueOrNull(c.DoclingServerURL),
		DoclingAPIKey:                            stringValueOrNull(c.DoclingAPIKey),
		DoclingParamsJSON:                        rawJSONToState(c.DoclingParams),
		DocumentIntelligenceEndpoint:             stringValueOrNull(c.DocumentIntelligenceEndpoint),
		DocumentIntelligenceKey:                  stringValueOrNull(c.DocumentIntelligenceKey),
		DocumentIntelligenceModel:                stringValueOrNull(c.DocumentIntelligenceModel),
		MistralOCRAPIBaseURL:                     stringValueOrNull(c.MistralOCRAPIBaseURL),
		MistralOCRAPIKey:                         stringValueOrNull(c.MistralOCRAPIKey),
		MistralOCRUseBase64:                      boolValueOrNull(c.MistralOCRUseBase64),
		PaddleOCRVLBaseURL:                       stringValueOrNull(c.PaddleOCRVLBaseURL),
		PaddleOCRVLToken:                         stringValueOrNull(c.PaddleOCRVLToken),
		MinerUAPIMode:                            stringValueOrNull(c.MinerUAPIMode),
		MinerUAPIURL:                             stringValueOrNull(c.MinerUAPIURL),
		MinerUAPIKey:                             stringValueOrNull(c.MinerUAPIKey),
		MinerUAPITimeout:                         int64ValueOrNull(c.MinerUAPITimeout),
		MinerUParamsJSON:                         rawJSONToState(c.MinerUParams),
		MinerUFileExtensions:                     list(c.MinerUFileExtensions),
		RAGRerankingModel:                        stringValueOrNull(c.RAGRerankingModel),
		RAGRerankingEngine:                       stringValueOrNull(c.RAGRerankingEngine),
		RAGRerankingBatchSize:                    int64ValueOrNull(c.RAGRerankingBatchSize),
		RAGExternalRerankerURL:                   stringValueOrNull(c.RAGExternalRerankerURL),
		RAGExternalRerankerAPIKey:                stringValueOrNull(c.RAGExternalRerankerAPIKey),
		RAGExternalRerankerTimeout:               stringValueOrNull(c.RAGExternalRerankerTimeout),
		TextSplitter:                             stringValueOrNull(c.TextSplitter),
		RAGTokenizerModel:                        stringValueOrNull(c.RAGTokenizerModel),
		EnableMarkdownHeaderTextSplitter:         boolValueOrNull(c.EnableMarkdownHeaderTextSplitter),
		ChunkSize:                                int64ValueOrNull(c.ChunkSize),
		ChunkMinSizeTarget:                       int64ValueOrNull(c.ChunkMinSizeTarget),
		ChunkOverlap:                             int64ValueOrNull(c.ChunkOverlap),
		FileMaxSize:                              int64ValueOrNull(c.FileMaxSize),
		FileMaxCount:                             int64ValueOrNull(c.FileMaxCount),
		FileImageCompressionWidth:                int64ValueOrNull(c.FileImageCompressionWidth),
		FileImageCompressionHeight:               int64ValueOrNull(c.FileImageCompressionHeight),
		AllowedFileExtensions:                    list(c.AllowedFileExtensions),
		EnableGoogleDriveIntegration:             boolValueOrNull(c.EnableGoogleDriveIntegration),
		EnableOneDriveIntegration:                boolValueOrNull(c.EnableOneDriveIntegration),
	}
	return m, diags
}

// preserveRetrievalConfig reconciles a freshly read model with the preferred
// (plan or prior state) one: semantically equal JSON keeps the preferred
// text, and a file limit that is null on the server keeps a preferred -1
// ("no limit", which the server stores as null).
func preserveRetrievalConfig(preferred, fresh retrievalConfigModel) retrievalConfigModel {
	fresh.ExternalDocumentLoaderHeadersJSON = preserveEquivalentJSON(preferred.ExternalDocumentLoaderHeadersJSON, fresh.ExternalDocumentLoaderHeadersJSON)
	fresh.DoclingParamsJSON = preserveEquivalentJSON(preferred.DoclingParamsJSON, fresh.DoclingParamsJSON)
	fresh.MinerUParamsJSON = preserveEquivalentJSON(preferred.MinerUParamsJSON, fresh.MinerUParamsJSON)

	keepCleared := func(p, f types.Int64) types.Int64 {
		if f.IsNull() && !p.IsNull() && !p.IsUnknown() && p.ValueInt64() < 0 {
			return p
		}
		return f
	}
	fresh.FileMaxSize = keepCleared(preferred.FileMaxSize, fresh.FileMaxSize)
	fresh.FileMaxCount = keepCleared(preferred.FileMaxCount, fresh.FileMaxCount)
	fresh.FileImageCompressionWidth = keepCleared(preferred.FileImageCompressionWidth, fresh.FileImageCompressionWidth)
	fresh.FileImageCompressionHeight = keepCleared(preferred.FileImageCompressionHeight, fresh.FileImageCompressionHeight)
	return fresh
}

// checkRetrievalApplyResult turns the two known ways the server can disagree
// with a configured (sent) value into clear errors, instead of Terraform's
// generic "inconsistent result after apply". Attributes the configuration
// leaves out are never checked here (see reconcileUnconfiguredRetrieval).
func checkRetrievalApplyResult(config, fresh retrievalConfigModel) diag.Diagnostics {
	var diags diag.Diagnostics

	if !config.EnableRAGHybridSearch.IsNull() && !config.EnableRAGHybridSearch.IsUnknown() && config.EnableRAGHybridSearch.ValueBool() &&
		!fresh.EnableRAGHybridSearch.IsNull() && !fresh.EnableRAGHybridSearch.ValueBool() {
		diags.AddAttributeError(path.Root("enable_rag_hybrid_search"), "Hybrid search was turned off by Open WebUI",
			"Open WebUI could not load the reranking model (rag_reranking_engine / rag_reranking_model / rag_external_reranker_*) and disabled hybrid search. Check the Open WebUI server log, fix the reranking settings, and apply again.")
	}

	type compat struct {
		name    string
		planned bool
		missing bool
		version string
	}
	known := func(v attr.Value) bool { return !v.IsNull() && !v.IsUnknown() }
	for _, c := range []compat{
		{"external_document_loader_headers_json", known(config.ExternalDocumentLoaderHeadersJSON), fresh.ExternalDocumentLoaderHeadersJSON.IsNull(), "v0.10.0"},
		{"rag_tokenizer_model", known(config.RAGTokenizerModel), fresh.RAGTokenizerModel.IsNull(), "v0.10.0"},
		{"mistral_ocr_use_base64", known(config.MistralOCRUseBase64), fresh.MistralOCRUseBase64.IsNull(), "v0.10.0"},
		{"content_extraction_supported_media_mime_types", known(config.ContentExtractionSupportedMediaMimeTypes), fresh.ContentExtractionSupportedMediaMimeTypes.IsNull(), "v0.11.0"},
	} {
		if c.planned && c.missing {
			diags.AddAttributeError(path.Root(c.name), "Setting not supported by this Open WebUI version",
				fmt.Sprintf("%s was added in Open WebUI %s; this server ignored it. Remove it from the configuration or upgrade Open WebUI.", c.name, c.version))
		}
	}
	return diags
}
