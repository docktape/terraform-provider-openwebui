package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

var _ datasource.DataSource = &modelDataSource{}
var _ datasource.DataSourceWithConfigure = &modelDataSource{}

// modelDataSource exposes read-only model information.
type modelDataSource struct {
	client *client.Client
}

// modelDataSourceModel reuses the resource model structure for mapping purposes.
type modelDataSourceModel = modelResourceModel

// NewModelDataSource constructs a new model data source instance.
func NewModelDataSource() datasource.DataSource {
	return &modelDataSource{}
}

// Metadata sets the data source identifier.
func (d *modelDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_model"
}

// Schema describes the model data source schema.
func (d *modelDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up an existing model by its `model_id`.",
		Attributes: map[string]schema.Attribute{
			"model_id": schema.StringAttribute{
				Required:    true,
				Description: "Identifier of the model to look up.",
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Composite identifier mirroring `model_id`.",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "Display name of the model.",
			},
			"base_model_id": schema.StringAttribute{
				Computed:    true,
				Description: "Identifier of the underlying base model.",
			},
			"is_active": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the model is visible and available to users.",
			},
			"user_id": schema.StringAttribute{
				Computed:    true,
				Description: "Owner user identifier.",
			},
			"created_at": schema.Int64Attribute{
				Computed:    true,
				Description: "Unix timestamp of when the model was created.",
			},
			"updated_at": schema.Int64Attribute{
				Computed:    true,
				Description: "Unix timestamp of when the model was last updated.",
			},
			"meta_additional_json": schema.StringAttribute{
				Computed:    true,
				Description: "Additional metadata JSON as returned by Open WebUI.",
			},
			"params_additional_json": schema.StringAttribute{
				Computed:    true,
				Description: "Additional parameters JSON as returned by Open WebUI.",
			},
			"read_groups": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "Read-access group names.",
			},
			"write_groups": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "Write-access group names.",
			},
			"params": schema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Model parameter values returned by Open WebUI.",
				Attributes: map[string]schema.Attribute{
					"system":                  schema.StringAttribute{Computed: true, Description: "System prompt prepended to every conversation."},
					"stream_response":         schema.BoolAttribute{Computed: true, Description: "Whether to stream the response. Defaults to the base model setting."},
					"stream_delta_chunk_size": schema.Int64Attribute{Computed: true, Description: "Chunk size in tokens for streaming responses."},
					"function_calling":        schema.StringAttribute{Computed: true, Description: "Function calling mode, e.g. `\"auto\"`."},
					"reasoning_tags": schema.ListAttribute{
						ElementType: types.StringType,
						Computed:    true,
						Description: "List of XML tags used to delimit reasoning tokens.",
					},
					"seed":              schema.Int64Attribute{Computed: true, Description: "Random seed for reproducible outputs."},
					"temperature":       schema.Float64Attribute{Computed: true, Description: "Sampling temperature (0–2). Lower values are more deterministic."},
					"keep_alive":        schema.StringAttribute{Computed: true, Description: "How long to keep the model loaded in memory, e.g. `\"5m\"`."},
					"num_gpu":           schema.Int64Attribute{Computed: true, Description: "Number of GPU layers to use."},
					"num_thread":        schema.Int64Attribute{Computed: true, Description: "Number of CPU threads to use for inference."},
					"num_batch":         schema.Int64Attribute{Computed: true, Description: "Batch size for prompt processing."},
					"num_ctx":           schema.Int64Attribute{Computed: true, Description: "Context window size in tokens."},
					"num_keep":          schema.Int64Attribute{Computed: true, Description: "Number of tokens to retain from the previous context."},
					"format":            schema.StringAttribute{Computed: true, Description: "Output format, e.g. `json`."},
					"think":             schema.BoolAttribute{Computed: true, Description: "Whether to enable chain-of-thought reasoning."},
					"use_mlock":         schema.BoolAttribute{Computed: true, Description: "Whether to lock model weights in RAM."},
					"use_mmap":          schema.BoolAttribute{Computed: true, Description: "Whether to use memory-mapped files for model weights."},
					"repeat_penalty":    schema.Float64Attribute{Computed: true, Description: "Penalty applied to repeated tokens."},
					"tfs_z":             schema.Float64Attribute{Computed: true, Description: "Tail free sampling z parameter."},
					"repeat_last_n":     schema.Int64Attribute{Computed: true, Description: "Number of tokens to look back for repeat penalty."},
					"mirostat_tau":      schema.Float64Attribute{Computed: true, Description: "Mirostat target entropy."},
					"mirostat_eta":      schema.Float64Attribute{Computed: true, Description: "Mirostat learning rate."},
					"mirostat":          schema.Int64Attribute{Computed: true, Description: "Mirostat sampling mode: 0 = disabled, 1 = Mirostat, 2 = Mirostat 2.0."},
					"presence_penalty":  schema.Float64Attribute{Computed: true, Description: "Penalty for tokens that have appeared at all."},
					"frequency_penalty": schema.Float64Attribute{Computed: true, Description: "Penalty scaled by token frequency."},
					"min_p":             schema.Float64Attribute{Computed: true, Description: "Minimum probability threshold for token sampling."},
					"top_p":             schema.Float64Attribute{Computed: true, Description: "Nucleus sampling probability threshold."},
					"top_k":             schema.Int64Attribute{Computed: true, Description: "Top-k sampling: number of highest-probability tokens to consider."},
					"max_tokens":        schema.Int64Attribute{Computed: true, Description: "Maximum number of tokens to generate."},
					"reasoning_effort":  schema.StringAttribute{Computed: true, Description: "Reasoning effort level for models that support it, e.g. `\"high\"`."},
					"custom_params": schema.MapAttribute{
						ElementType: types.StringType,
						Computed:    true,
						Description: "Additional key/value parameters returned by Open WebUI.",
					},
				},
			},
			"capabilities": schema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Feature capability flags returned by Open WebUI.",
				Attributes: map[string]schema.Attribute{
					"vision":           schema.BoolAttribute{Computed: true, Description: "Whether the model accepts image inputs."},
					"file_upload":      schema.BoolAttribute{Computed: true, Description: "Whether file uploads are allowed in chat."},
					"web_search":       schema.BoolAttribute{Computed: true, Description: "Whether web search is available."},
					"image_generation": schema.BoolAttribute{Computed: true, Description: "Whether image generation is available."},
					"code_interpreter": schema.BoolAttribute{Computed: true, Description: "Whether code interpreter is available."},
					"citations":        schema.BoolAttribute{Computed: true, Description: "Whether the model returns citations."},
					"status_updates":   schema.BoolAttribute{Computed: true, Description: "Whether the model emits status update messages."},
					"usage":            schema.BoolAttribute{Computed: true, Description: "Whether token usage statistics are returned."},
				},
			},
		},
	}
}

// Configure assigns the shared API client.
func (d *modelDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	if client, ok := req.ProviderData.(*client.Client); ok {
		d.client = client
	}
}

// Read fetches the model details from Open WebUI.
func (d *modelDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before using the model data source.")
		return
	}

	var config modelDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.ModelID.IsUnknown() || config.ModelID.IsNull() || config.ModelID.ValueString() == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("model_id"),
			"Missing model identifier",
			"The model_id argument must be supplied to query an existing model.",
		)
		return
	}

	current, err := d.client.GetModel(ctx, config.ModelID.ValueString())
	if err != nil {
		if err == client.ErrNotFound {
			resp.Diagnostics.AddAttributeError(
				path.Root("model_id"),
				"Model not found",
				"No Open WebUI model was found with the supplied model_id.",
			)
			return
		}

		resp.Diagnostics.AddError("Read model failed", err.Error())
		return
	}

	state, diags := modelResponseToModel(ctx, d.client, current, config.ModelID.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
