package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

var _ datasource.DataSource = &terminalServerVerifyDataSource{}
var _ datasource.DataSourceWithConfigure = &terminalServerVerifyDataSource{}

// terminalServerVerifyDataSource verifies terminal server connectivity.
type terminalServerVerifyDataSource struct {
	client *client.Client
}

type terminalServerVerifyModel struct {
	URL        types.String `tfsdk:"url"`
	Key        types.String `tfsdk:"key"`
	AuthType   types.String `tfsdk:"auth_type"`
	Verified   types.Bool   `tfsdk:"verified"`
	ServerType types.String `tfsdk:"server_type"`
}

// NewTerminalServerVerifyDataSource constructs a new terminal server verify data source.
func NewTerminalServerVerifyDataSource() datasource.DataSource {
	return &terminalServerVerifyDataSource{}
}

func (d *terminalServerVerifyDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_terminal_server_verify"
}

func (d *terminalServerVerifyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	const urlDesc = "Base URL of the terminal server to verify."
	const keyDesc = "Bearer token sent when `auth_type` is `bearer`."
	const authDesc = "Authentication type: `bearer` (default, sends `key`), `none`, `session` or `system_oauth`. Open WebUI only sends credentials for `bearer` when verifying."
	const verifiedDesc = "`true` if the server answered. A failed verification is an error instead."
	const typeDesc = "Detected server type: `orchestrator` when `<url>/api/v1/policies` answers, otherwise `terminal` when `<url>/api/config` answers."

	resp.Schema = schema.Schema{
		MarkdownDescription: "Asks Open WebUI to probe a terminal server and detect whether it is an orchestrator or a plain Open Terminal. The request is made by the Open WebUI server, so the URL must be reachable from there. Fails when neither probe endpoint answers.",
		Attributes: map[string]schema.Attribute{
			"url": schema.StringAttribute{
				Required: true, Description: urlDesc, MarkdownDescription: urlDesc,
				Validators: []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"key": schema.StringAttribute{
				Optional: true, Sensitive: true, Description: keyDesc, MarkdownDescription: keyDesc,
			},
			"auth_type": schema.StringAttribute{
				Optional: true, Description: authDesc, MarkdownDescription: authDesc,
				Validators: []validator.String{stringvalidator.OneOf("bearer", "none", "session", "system_oauth")},
			},
			"verified": schema.BoolAttribute{
				Computed: true, Description: verifiedDesc, MarkdownDescription: verifiedDesc,
			},
			"server_type": schema.StringAttribute{
				Computed: true, Description: typeDesc, MarkdownDescription: typeDesc,
			},
		},
	}
}

func (d *terminalServerVerifyDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	if c, ok := req.ProviderData.(*client.Client); ok {
		d.client = c
	}
}

func (d *terminalServerVerifyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before using the terminal server verify data source.")
		return
	}
	var config terminalServerVerifyModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The client always sends auth_type; an explicit null would defeat the
	// backend's `bearer` default, so apply it here.
	authType := "bearer"
	if !config.AuthType.IsNull() && !config.AuthType.IsUnknown() {
		authType = config.AuthType.ValueString()
	}
	result, err := d.client.VerifyTerminalServer(ctx, client.TerminalServerConnection{
		URL:      config.URL.ValueString(),
		Key:      stringPtr(config.Key),
		AuthType: &authType,
	})
	if err != nil {
		resp.Diagnostics.AddError("Verify terminal server failed", err.Error())
		return
	}

	config.Verified = types.BoolValue(result.Status)
	config.ServerType = types.StringValue(result.Type)
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
