package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

var _ datasource.DataSource = &openAIConnectionVerifyDataSource{}
var _ datasource.DataSourceWithConfigure = &openAIConnectionVerifyDataSource{}

type openAIConnectionVerifyDataSource struct {
	client *client.Client
}

type openAIConnectionVerifyModel struct {
	URL         types.String `tfsdk:"url"`
	Key         types.String `tfsdk:"key"`
	AuthType    types.String `tfsdk:"auth_type"`
	Provider    types.String `tfsdk:"provider"`
	APIVersion  types.String `tfsdk:"api_version"`
	HeadersJSON types.String `tfsdk:"headers_json"`
	Verified    types.Bool   `tfsdk:"verified"`
}

// NewOpenAIConnectionVerifyDataSource constructs a new openai_connection_verify data source.
func NewOpenAIConnectionVerifyDataSource() datasource.DataSource {
	return &openAIConnectionVerifyDataSource{}
}

func (d *openAIConnectionVerifyDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_openai_connection_verify"
}

func (d *openAIConnectionVerifyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Verifies that an OpenAI-compatible endpoint is reachable and accepts the provided credentials.",
		Attributes: map[string]schema.Attribute{
			"url": schema.StringAttribute{
				Required:    true,
				Description: "Base URL to verify, e.g. `https://api.openai.com/v1`.",
			},
			"key": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "API key or bearer token. Sensitive.",
			},
			"auth_type": schema.StringAttribute{
				Optional:    true,
				Description: "Authentication method, e.g. `bearer`. Passed to Open WebUI's verification logic.",
			},
			"provider": schema.StringAttribute{
				Optional:    true,
				Description: "Provider hint, e.g. `azure`. Passed to Open WebUI's verification logic.",
			},
			"api_version": schema.StringAttribute{
				Optional:    true,
				Description: "Azure API version, e.g. `2024-02-01`. Used when `provider = \"azure\"`.",
			},
			"headers_json": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "JSON object of custom headers to include in the verification request.",
			},
			"verified": schema.BoolAttribute{
				Computed:    true,
				Description: "`true` if Open WebUI successfully connected to the endpoint.",
			},
		},
	}
}

func (d *openAIConnectionVerifyDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	if c, ok := req.ProviderData.(*client.Client); ok {
		d.client = c
	}
}

func (d *openAIConnectionVerifyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before using the openai_connection_verify data source.")
		return
	}

	var config openAIConnectionVerifyModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	headers := decodeOptionalJSON(config.HeadersJSON, path.Root("headers_json"), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	key := ""
	if !config.Key.IsNull() && !config.Key.IsUnknown() {
		key = config.Key.ValueString()
	}

	cfg := &client.OpenAIConnectionConfig{
		AuthType:   config.AuthType.ValueString(),
		Provider:   config.Provider.ValueString(),
		APIVersion: config.APIVersion.ValueString(),
		Headers:    headers,
	}

	if err := d.client.VerifyOpenAIConnection(ctx, config.URL.ValueString(), key, cfg); err != nil {
		resp.Diagnostics.AddError("OpenAI connection verification failed", err.Error())
		return
	}

	state := config
	state.Verified = types.BoolValue(true)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
