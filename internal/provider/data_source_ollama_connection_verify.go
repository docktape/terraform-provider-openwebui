package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

var _ datasource.DataSource = &ollamaConnectionVerifyDataSource{}
var _ datasource.DataSourceWithConfigure = &ollamaConnectionVerifyDataSource{}

type ollamaConnectionVerifyDataSource struct {
	client *client.Client
}

type ollamaConnectionVerifyModel struct {
	URL      types.String `tfsdk:"url"`
	Key      types.String `tfsdk:"key"`
	Verified types.Bool   `tfsdk:"verified"`
}

// NewOllamaConnectionVerifyDataSource constructs a new ollama_connection_verify data source.
func NewOllamaConnectionVerifyDataSource() datasource.DataSource {
	return &ollamaConnectionVerifyDataSource{}
}

func (d *ollamaConnectionVerifyDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ollama_connection_verify"
}

func (d *ollamaConnectionVerifyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Verifies that an Ollama backend is reachable from Open WebUI.",
		Attributes: map[string]schema.Attribute{
			"url": schema.StringAttribute{
				Required:    true,
				Description: "Base URL of the Ollama backend, e.g. `http://localhost:11434`.",
			},
			"key": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Optional API key for authenticated Ollama instances. Sensitive.",
			},
			"verified": schema.BoolAttribute{
				Computed:    true,
				Description: "`true` if Open WebUI successfully connected to the Ollama backend.",
			},
		},
	}
}

func (d *ollamaConnectionVerifyDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	if c, ok := req.ProviderData.(*client.Client); ok {
		d.client = c
	}
}

func (d *ollamaConnectionVerifyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before using the ollama_connection_verify data source.")
		return
	}

	var config ollamaConnectionVerifyModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var key *string
	if !config.Key.IsNull() && !config.Key.IsUnknown() {
		v := config.Key.ValueString()
		key = &v
	}

	if err := d.client.VerifyOllamaConnection(ctx, config.URL.ValueString(), key); err != nil {
		resp.Diagnostics.AddError("Ollama connection verification failed", err.Error())
		return
	}

	state := config
	state.Verified = types.BoolValue(true)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
