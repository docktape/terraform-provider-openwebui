package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

var _ resource.Resource = &retrievalEmbeddingConfigResource{}
var _ resource.ResourceWithConfigure = &retrievalEmbeddingConfigResource{}
var _ resource.ResourceWithImportState = &retrievalEmbeddingConfigResource{}

// embeddingConnectionAttrTypes is the {url, key} shape of openai_config and
// ollama_config.
var embeddingConnectionAttrTypes = map[string]attr.Type{
	"url": types.StringType,
	"key": types.StringType,
}

// embeddingAzureAttrTypes is the {url, key, version} shape of
// azure_openai_config.
var embeddingAzureAttrTypes = map[string]attr.Type{
	"url":     types.StringType,
	"key":     types.StringType,
	"version": types.StringType,
}

type retrievalEmbeddingConfigResource struct {
	client *client.Client
}

type retrievalEmbeddingConfigModel struct {
	ID                             types.String `tfsdk:"id"`
	RAGEmbeddingEngine             types.String `tfsdk:"rag_embedding_engine"`
	RAGEmbeddingModel              types.String `tfsdk:"rag_embedding_model"`
	RAGEmbeddingBatchSize          types.Int64  `tfsdk:"rag_embedding_batch_size"`
	EnableAsyncEmbedding           types.Bool   `tfsdk:"enable_async_embedding"`
	RAGEmbeddingConcurrentRequests types.Int64  `tfsdk:"rag_embedding_concurrent_requests"`
	OpenAIConfig                   types.Object `tfsdk:"openai_config"`
	OllamaConfig                   types.Object `tfsdk:"ollama_config"`
	AzureOpenAIConfig              types.Object `tfsdk:"azure_openai_config"`
}

// embeddingConnectionModel is the model of openai_config and ollama_config.
type embeddingConnectionModel struct {
	URL types.String `tfsdk:"url"`
	Key types.String `tfsdk:"key"`
}

// embeddingAzureModel is the model of azure_openai_config.
type embeddingAzureModel struct {
	URL     types.String `tfsdk:"url"`
	Key     types.String `tfsdk:"key"`
	Version types.String `tfsdk:"version"`
}

// NewRetrievalEmbeddingConfigResource constructs the embedding config resource.
func NewRetrievalEmbeddingConfigResource() resource.Resource {
	return &retrievalEmbeddingConfigResource{}
}

// Metadata sets the resource type name.
func (r *retrievalEmbeddingConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_retrieval_embedding_config"
}

// Schema defines the embedding config schema.
func (r *retrievalEmbeddingConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	str := func(desc string, sensitive bool, validators ...validator.String) schema.StringAttribute {
		return schema.StringAttribute{Optional: true, Computed: true, Sensitive: sensitive, Description: desc, MarkdownDescription: desc, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}, Validators: validators}
	}
	i := func(desc string, validators ...validator.Int64) schema.Int64Attribute {
		return schema.Int64Attribute{Optional: true, Computed: true, Description: desc, MarkdownDescription: desc, PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}, Validators: validators}
	}
	nested := func(desc string, attrs map[string]schema.Attribute) schema.SingleNestedAttribute {
		return schema.SingleNestedAttribute{Optional: true, Computed: true, Description: desc, MarkdownDescription: desc, PlanModifiers: []planmodifier.Object{objectplanmodifier.UseStateForUnknown()}, Attributes: attrs}
	}
	const keepNote = " Attributes left out of a configured block keep their current Open WebUI value; a block left out is not sent at all."
	const storedNote = " Open WebUI stores a change to this block only while `rag_embedding_engine` is `ollama`, `openai` or `azure_openai`; with the local engine a change is refused before anything is sent."
	const keyNote = " Stored in Terraform state (Open WebUI returns it in plain text). Use `\"\"` for none."
	const urlNote = " Shown in plain text: do not embed credentials in it, use `key` instead."

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages Open WebUI's embedding model settings (Admin Settings > Documents > Embedding): the embedding engine and model, batching and concurrency, and the connection settings of the Ollama, OpenAI and Azure OpenAI engines. They come from their own endpoint and are not part of `openwebui_retrieval_config`.\n\n" +
			"~> Open WebUI requires the engine and the model in every update and resets the batch size, the async toggle and the concurrency to their defaults when they are left out. This resource therefore reads the current settings, changes only the attributes set in your configuration, and sends the others back as read. Any attribute you leave out keeps its current Open WebUI value, and its state shows that value; removing an attribute from your configuration does not reset it. A connection block (`openai_config`, `ollama_config`, `azure_openai_config`) you leave out is not sent; inside a configured block, an attribute you leave out keeps its current value. The read and the write are separate requests, so an embedding change made by someone else in between is overwritten.\n\n" +
			"~> Open WebUI stores the connection blocks only while `rag_embedding_engine` is `ollama`, `openai` or `azure_openai` (all three blocks, for any of these engines) and silently ignores them for the local engine (`\"\"`). A change to a block while the engine (after the apply) is the local one therefore fails before anything is sent. A configured block that equals the current values is accepted with any engine.\n\n" +
			"~> The connection keys are stored in Terraform state because Open WebUI returns them in plain text. Protect your state accordingly. The `url` attributes are not marked sensitive: do not embed credentials in them.\n\n" +
			"~> Every update makes Open WebUI unload and reload the embedding model. With the local engine (`\"\"`) Open WebUI loads the sentence-transformers model into its own process: a model that is not in its cache yet is downloaded from Hugging Face inside the update request, and unless `OFFLINE_MODE` is set or `RAG_EMBEDDING_MODEL_AUTO_UPDATE` is false Open WebUI checks Hugging Face for a newer revision on every update. If the model cannot be loaded, Open WebUI still reports success, and document embedding fails until a valid model is set. The update request waits up to 10 minutes for Open WebUI to load the embedding model (unless a proxy in front of Open WebUI has a shorter timeout). If a download takes longer, the apply fails, but Open WebUI still completes the change, so re-running `terraform plan`/`terraform apply` converges. Open WebUI handles no other request while it loads a model, so other Open WebUI resources in the same apply (whose requests keep the provider's 30-second timeout) can time out meanwhile; re-running converges as well. Pre-downloading the model (for example by selecting it once in the Open WebUI admin page) avoids the wait. Changing the engine or the model does not re-embed existing documents; reindex the knowledge bases afterwards (Admin Settings > Documents > Reindex Knowledge Base Vectors). An apply that changes nothing sends nothing.\n\n" +
			"~> If Open WebUI reports a different value during an apply for an attribute you do not set (because it was changed outside Terraform after the last refresh), the apply leaves it unchanged and keeps the previous state value with a warning; the next refresh shows the server value.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Singleton identifier, always `retrieval_embedding`.",
				MarkdownDescription: "Singleton identifier, always `retrieval_embedding`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"rag_embedding_engine": str("Embedding engine: `\"\"` for the local sentence-transformers model (the server default), `ollama`, `openai` or `azure_openai`. Open WebUI fails to reload the model with any other value.", false,
				stringvalidator.OneOf("", "ollama", "openai", "azure_openai")),
			"rag_embedding_model": str("Embedding model: a Hugging Face sentence-transformers model name such as `sentence-transformers/all-MiniLM-L6-v2` (the server default) for the local engine, or the remote engine's model name, for example `text-embedding-3-small`. With the local engine a model that is not cached yet is downloaded during the apply, and `\"\"` loads no model. Must not start or end with whitespace (Open WebUI strips it).", false,
				noEdgeWhitespaceValidator),
			"rag_embedding_batch_size": i("Number of texts per embedding batch, at least 1. Server default: `1`.",
				int64validator.AtLeast(1)),
			"enable_async_embedding": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Whether batches are sent to the remote engine concurrently. Used by the `ollama`, `openai` and `azure_openai` engines only. Server default: `true`.",
				MarkdownDescription: "Whether batches are sent to the remote engine concurrently. Used by the `ollama`, `openai` and `azure_openai` engines only. Server default: `true`.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"rag_embedding_concurrent_requests": i("Maximum number of concurrent embedding requests to the remote engine while `enable_async_embedding` is on; `0` means unlimited. Used by the `ollama`, `openai` and `azure_openai` engines only. Server default: `0`.",
				int64validator.AtLeast(0)),
			"openai_config": nested("Connection of the `openai` engine (Open WebUI's `RAG_OPENAI_API_BASE_URL` and `RAG_OPENAI_API_KEY`)."+keepNote+storedNote, map[string]schema.Attribute{
				"url": str("Base URL of the OpenAI-compatible API, for example `https://api.openai.com/v1`."+urlNote, false),
				"key": str("API key of the OpenAI-compatible API."+keyNote, true),
			}),
			"ollama_config": nested("Connection of the `ollama` engine (Open WebUI's `RAG_OLLAMA_BASE_URL` and `RAG_OLLAMA_API_KEY`)."+keepNote+storedNote, map[string]schema.Attribute{
				"url": str("Base URL of the Ollama server, for example `http://ollama:11434`."+urlNote, false),
				"key": str("API key sent to the Ollama server as a bearer token."+keyNote, true),
			}),
			"azure_openai_config": nested("Connection of the `azure_openai` engine (Open WebUI's `RAG_AZURE_OPENAI_BASE_URL`, `RAG_AZURE_OPENAI_API_KEY` and `RAG_AZURE_OPENAI_API_VERSION`)."+keepNote+storedNote, map[string]schema.Attribute{
				"url":     str("Azure OpenAI endpoint, for example `https://my-resource.openai.azure.com`."+urlNote, false),
				"key":     str("Azure OpenAI API key."+keyNote, true),
				"version": str("Azure OpenAI API version, for example `2024-02-01`.", false),
			}),
		},
	}
}

// Configure assigns the API client.
func (r *retrievalEmbeddingConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	if c, ok := req.ProviderData.(*client.Client); ok {
		r.client = c
	}
}

// Create applies the configured embedding settings.
func (r *retrievalEmbeddingConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing embedding config.")
		return
	}
	var config, plan retrievalEmbeddingConfigModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	state, diags := applyRetrievalEmbeddingConfig(ctx, r.client, config, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Read refreshes the embedding settings.
func (r *retrievalEmbeddingConfigResource) Read(ctx context.Context, _ resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing embedding config.")
		return
	}
	current, err := r.client.GetRetrievalEmbeddingConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Read embedding config failed", err.Error())
		return
	}
	state, diags := retrievalEmbeddingConfigModelFromAPI(ctx, *current)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update applies the configured embedding settings.
func (r *retrievalEmbeddingConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing embedding config.")
		return
	}
	var config, plan retrievalEmbeddingConfigModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	state, diags := applyRetrievalEmbeddingConfig(ctx, r.client, config, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete removes the resource from state without changing remote configuration.
func (r *retrievalEmbeddingConfigResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing embedding config.")
		return
	}
	// Singleton config: destroying the resource leaves the remote config untouched.
}

// ImportState maps import identifiers onto the id attribute.
func (r *retrievalEmbeddingConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// applyRetrievalEmbeddingConfig changes the attributes set in the
// configuration and re-reads the embedding settings. The update is built from
// config, not plan: every attribute is Optional+Computed with
// UseStateForUnknown, so on update the plan also carries prior-state values
// for attributes the configuration leaves out, and sending those would revert
// changes made outside Terraform. client.UpdateRetrievalEmbeddingConfig
// completes the update with the current settings (the endpoint requires the
// engine and model and resets the other scalars when they are left out) and
// sends nothing when nothing changes.
//
// The plan is still what the written state must agree with: configured
// attributes are checked by checkRetrievalEmbeddingApplyResult, and
// unconfigured ones are reconciled by reconcileRetrievalEmbedding.
func applyRetrievalEmbeddingConfig(ctx context.Context, apiClient *client.Client, config, plan retrievalEmbeddingConfigModel) (retrievalEmbeddingConfigModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	update, updateDiags := retrievalEmbeddingConfigUpdateFromConfig(ctx, config)
	diags.Append(updateDiags...)
	if diags.HasError() {
		return retrievalEmbeddingConfigModel{}, diags
	}

	if err := apiClient.UpdateRetrievalEmbeddingConfig(ctx, update); err != nil {
		diags.AddError("Update embedding config failed", err.Error())
		return retrievalEmbeddingConfigModel{}, diags
	}

	current, err := apiClient.GetRetrievalEmbeddingConfig(ctx)
	if err != nil {
		diags.AddError("Read embedding config after update failed", err.Error())
		return retrievalEmbeddingConfigModel{}, diags
	}
	fresh, freshDiags := retrievalEmbeddingConfigModelFromAPI(ctx, *current)
	diags.Append(freshDiags...)
	if diags.HasError() {
		return retrievalEmbeddingConfigModel{}, diags
	}

	diags.Append(checkRetrievalEmbeddingApplyResult(config, fresh)...)
	if diags.HasError() {
		return retrievalEmbeddingConfigModel{}, diags
	}
	state, reconcileDiags := reconcileRetrievalEmbedding(ctx, config, plan, fresh)
	diags.Append(reconcileDiags...)
	return state, diags
}

// retrievalEmbeddingCause completes the reconcile warning.
const retrievalEmbeddingCause = "changed outside Terraform since the last refresh"

// reconcileRetrievalEmbedding keeps the planned value, with a warning, of
// every unconfigured attribute that Open WebUI reports differently (see
// reconcileUnconfiguredAttributes): first the unconfigured attributes inside
// each configured connection block, then the top-level attributes, whole
// unconfigured blocks included.
func reconcileRetrievalEmbedding(ctx context.Context, config, plan, fresh retrievalEmbeddingConfigModel) (retrievalEmbeddingConfigModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	fresh.OpenAIConfig = reconcileEmbeddingBlock[embeddingConnectionModel](ctx, "openai_config", embeddingConnectionAttrTypes, config.OpenAIConfig, plan.OpenAIConfig, fresh.OpenAIConfig, &diags)
	fresh.OllamaConfig = reconcileEmbeddingBlock[embeddingConnectionModel](ctx, "ollama_config", embeddingConnectionAttrTypes, config.OllamaConfig, plan.OllamaConfig, fresh.OllamaConfig, &diags)
	fresh.AzureOpenAIConfig = reconcileEmbeddingBlock[embeddingAzureModel](ctx, "azure_openai_config", embeddingAzureAttrTypes, config.AzureOpenAIConfig, plan.AzureOpenAIConfig, fresh.AzureOpenAIConfig, &diags)
	if diags.HasError() {
		return retrievalEmbeddingConfigModel{}, diags
	}
	state, topDiags := reconcileUnconfiguredAttributes(config, plan, fresh, retrievalEmbeddingCause)
	diags.Append(topDiags...)
	return state, diags
}

// reconcileEmbeddingBlock reconciles the inner attributes of one configured
// connection block; an unconfigured block is left to the top-level pass.
func reconcileEmbeddingBlock[T any](ctx context.Context, name string, attrTypes map[string]attr.Type, config, plan, fresh types.Object, diags *diag.Diagnostics) types.Object {
	if !knownObject(config) || !knownObject(plan) || !knownObject(fresh) {
		return fresh
	}
	var c, p, f T
	diags.Append(config.As(ctx, &c, objectAsOptions)...)
	diags.Append(plan.As(ctx, &p, objectAsOptions)...)
	diags.Append(fresh.As(ctx, &f, objectAsOptions)...)
	if diags.HasError() {
		return fresh
	}
	merged, d := reconcileUnconfiguredAttributesAt(path.Root(name), c, p, f, retrievalEmbeddingCause)
	diags.Append(d...)
	obj, d := types.ObjectValueFrom(ctx, attrTypes, merged)
	diags.Append(d...)
	return obj
}

// retrievalEmbeddingConfigUpdateFromConfig maps every attribute set in the
// configuration onto the update. Null (unset) and unknown attributes stay nil
// and keep their current Open WebUI value; inside a configured connection
// block likewise.
func retrievalEmbeddingConfigUpdateFromConfig(ctx context.Context, m retrievalEmbeddingConfigModel) (client.RetrievalEmbeddingConfigUpdate, diag.Diagnostics) {
	var diags diag.Diagnostics
	update := client.RetrievalEmbeddingConfigUpdate{
		RAGEmbeddingEngine:             stringPtr(m.RAGEmbeddingEngine),
		RAGEmbeddingModel:              stringPtr(m.RAGEmbeddingModel),
		RAGEmbeddingBatchSize:          int64Ptr(m.RAGEmbeddingBatchSize),
		EnableAsyncEmbedding:           boolPtr(m.EnableAsyncEmbedding),
		RAGEmbeddingConcurrentRequests: int64Ptr(m.RAGEmbeddingConcurrentRequests),
	}
	connection := func(obj types.Object) *client.EmbeddingConnectionConfig {
		if !knownObject(obj) {
			return nil
		}
		var c embeddingConnectionModel
		diags.Append(obj.As(ctx, &c, objectAsOptions)...)
		out := &client.EmbeddingConnectionConfig{URL: stringPtr(c.URL), Key: stringPtr(c.Key)}
		if out.URL == nil && out.Key == nil {
			return nil
		}
		return out
	}
	update.OpenAIConfig = connection(m.OpenAIConfig)
	update.OllamaConfig = connection(m.OllamaConfig)
	if knownObject(m.AzureOpenAIConfig) {
		var c embeddingAzureModel
		diags.Append(m.AzureOpenAIConfig.As(ctx, &c, objectAsOptions)...)
		azure := &client.EmbeddingAzureOpenAIConfig{URL: stringPtr(c.URL), Key: stringPtr(c.Key), Version: stringPtr(c.Version)}
		if azure.URL != nil || azure.Key != nil || azure.Version != nil {
			update.AzureOpenAIConfig = azure
		}
	}
	return update, diags
}

// retrievalEmbeddingConfigModelFromAPI flattens the embedding settings. Absent
// or null keys become null attributes; an absent or null connection object
// becomes a null block, and a null field inside one a null attribute.
func retrievalEmbeddingConfigModelFromAPI(ctx context.Context, c client.RetrievalEmbeddingConfig) (retrievalEmbeddingConfigModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	connection := func(v *client.EmbeddingConnectionConfig) types.Object {
		if v == nil {
			return types.ObjectNull(embeddingConnectionAttrTypes)
		}
		obj, d := types.ObjectValueFrom(ctx, embeddingConnectionAttrTypes, embeddingConnectionModel{URL: stringValueOrNull(v.URL), Key: stringValueOrNull(v.Key)})
		diags.Append(d...)
		return obj
	}
	azure := types.ObjectNull(embeddingAzureAttrTypes)
	if v := c.AzureOpenAIConfig; v != nil {
		obj, d := types.ObjectValueFrom(ctx, embeddingAzureAttrTypes, embeddingAzureModel{URL: stringValueOrNull(v.URL), Key: stringValueOrNull(v.Key), Version: stringValueOrNull(v.Version)})
		diags.Append(d...)
		azure = obj
	}
	return retrievalEmbeddingConfigModel{
		ID:                             types.StringValue("retrieval_embedding"),
		RAGEmbeddingEngine:             stringValueOrNull(c.RAGEmbeddingEngine),
		RAGEmbeddingModel:              stringValueOrNull(c.RAGEmbeddingModel),
		RAGEmbeddingBatchSize:          int64ValueOrNull(c.RAGEmbeddingBatchSize),
		EnableAsyncEmbedding:           boolValueOrNull(c.EnableAsyncEmbedding),
		RAGEmbeddingConcurrentRequests: int64ValueOrNull(c.RAGEmbeddingConcurrentRequests),
		OpenAIConfig:                   connection(c.OpenAIConfig),
		OllamaConfig:                   connection(c.OllamaConfig),
		AzureOpenAIConfig:              azure,
	}, diags
}

// checkRetrievalEmbeddingApplyResult turns a configured value that Open WebUI
// did not store (for example a connection block it dropped) into a clear
// attribute error instead of Terraform's generic "inconsistent result after
// apply". Attributes the configuration leaves out are never checked here, and
// values are never printed (keys are secrets).
func checkRetrievalEmbeddingApplyResult(config, fresh retrievalEmbeddingConfigModel) diag.Diagnostics {
	var diags diag.Diagnostics
	check := func(p path.Path, configured, got attr.Value, hint string) {
		if configured.IsNull() || configured.IsUnknown() || configured.Equal(got) {
			return
		}
		diags.AddAttributeError(p, "Open WebUI did not store the configured value",
			fmt.Sprintf("%s is set in the configuration, but Open WebUI reports a different value after the update.%s", p, hint))
	}
	check(path.Root("rag_embedding_engine"), config.RAGEmbeddingEngine, fresh.RAGEmbeddingEngine, "")
	check(path.Root("rag_embedding_model"), config.RAGEmbeddingModel, fresh.RAGEmbeddingModel, "")
	check(path.Root("rag_embedding_batch_size"), config.RAGEmbeddingBatchSize, fresh.RAGEmbeddingBatchSize, "")
	check(path.Root("enable_async_embedding"), config.EnableAsyncEmbedding, fresh.EnableAsyncEmbedding, "")
	check(path.Root("rag_embedding_concurrent_requests"), config.RAGEmbeddingConcurrentRequests, fresh.RAGEmbeddingConcurrentRequests, "")

	const blockHint = " Open WebUI stores the connection blocks only while rag_embedding_engine is ollama, openai or azure_openai."
	for _, b := range []struct {
		name            string
		fields          []string
		configured, got types.Object
	}{
		{"openai_config", []string{"url", "key"}, config.OpenAIConfig, fresh.OpenAIConfig},
		{"ollama_config", []string{"url", "key"}, config.OllamaConfig, fresh.OllamaConfig},
		{"azure_openai_config", []string{"url", "key", "version"}, config.AzureOpenAIConfig, fresh.AzureOpenAIConfig},
	} {
		if !knownObject(b.configured) {
			continue
		}
		gotAttrs := map[string]attr.Value{}
		if knownObject(b.got) {
			gotAttrs = b.got.Attributes()
		}
		for _, field := range b.fields {
			got, ok := gotAttrs[field]
			if !ok {
				got = types.StringNull()
			}
			check(path.Root(b.name).AtName(field), b.configured.Attributes()[field], got, blockHint)
		}
	}
	return diags
}
