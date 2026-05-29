package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

var _ datasource.DataSource = &toolServerVerifyDataSource{}
var _ datasource.DataSourceWithConfigure = &toolServerVerifyDataSource{}

// toolServerVerifyDataSource verifies tool server connectivity.
type toolServerVerifyDataSource struct {
	client *client.Client
}

type toolServerVerifyModel struct {
	URL         types.String `tfsdk:"url"`
	Path        types.String `tfsdk:"path"`
	Type        types.String `tfsdk:"type"`
	AuthType    types.String `tfsdk:"auth_type"`
	HeadersJSON types.String `tfsdk:"headers_json"`
	Key         types.String `tfsdk:"key"`
	ConfigJSON  types.String `tfsdk:"config_json"`
	Verified    types.Bool   `tfsdk:"verified"`
}

// NewToolServerVerifyDataSource constructs a new tool server verify data source.
func NewToolServerVerifyDataSource() datasource.DataSource {
	return &toolServerVerifyDataSource{}
}

// Metadata sets the data source type name.
func (d *toolServerVerifyDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tool_server_verify"
}

// Schema defines the tool server verify schema.
func (d *toolServerVerifyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Verifies that an external tool server is reachable and returns a valid OpenAPI spec.",
		Attributes: map[string]schema.Attribute{
			"url": schema.StringAttribute{
				Required:    true,
				Description: "Base URL of the tool server to verify.",
			},
			"path": schema.StringAttribute{
				Required:    true,
				Description: "Path to the OpenAPI spec endpoint, e.g. `/openapi.json`.",
			},
			"type": schema.StringAttribute{
				Optional:    true,
				Description: "Tool server type identifier.",
			},
			"auth_type": schema.StringAttribute{
				Optional:    true,
				Description: "Authentication type for the tool server, e.g. `bearer`.",
			},
			"headers_json": schema.StringAttribute{
				Optional:    true,
				Description: "JSON object of additional HTTP headers to send. e.g. `jsonencode({ X-Api-Version = \"2\" })`.",
			},
			"key": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "API key or bearer token. Sensitive.",
			},
			"config_json": schema.StringAttribute{
				Optional:    true,
				Description: "Additional JSON configuration for the verification request.",
			},
			"verified": schema.BoolAttribute{
				Computed:    true,
				Description: "`true` if the server responded with a valid OpenAPI spec.",
			},
		},
	}
}

// Configure assigns the API client.
func (d *toolServerVerifyDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	if client, ok := req.ProviderData.(*client.Client); ok {
		d.client = client
	}
}

// Read verifies tool server connectivity.
func (d *toolServerVerifyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before using the tool server verify data source.")
		return
	}

	var config toolServerVerifyModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	headers := decodeOptionalJSON(config.HeadersJSON, path.Root("headers_json"), &resp.Diagnostics)
	configMap := decodeOptionalJSON(config.ConfigJSON, path.Root("config_json"), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	var authType *string
	if !config.AuthType.IsNull() && !config.AuthType.IsUnknown() {
		value := config.AuthType.ValueString()
		authType = &value
	}
	var connType *string
	if !config.Type.IsNull() && !config.Type.IsUnknown() {
		value := config.Type.ValueString()
		connType = &value
	}
	var key *string
	if !config.Key.IsNull() && !config.Key.IsUnknown() {
		value := config.Key.ValueString()
		key = &value
	}

	connection := client.ToolServerConnection{
		URL:      config.URL.ValueString(),
		Path:     config.Path.ValueString(),
		Type:     connType,
		AuthType: authType,
		Headers:  headers,
		Key:      key,
		Config:   configMap,
	}

	if err := d.client.VerifyToolServer(ctx, connection); err != nil {
		resp.Diagnostics.AddError("Verify tool server failed", err.Error())
		return
	}

	state := config
	state.Verified = types.BoolValue(true)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
