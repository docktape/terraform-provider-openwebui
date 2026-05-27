package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

var _ datasource.DataSource = &configExportDataSource{}
var _ datasource.DataSourceWithConfigure = &configExportDataSource{}

// configExportDataSource exports full Open WebUI configuration.
type configExportDataSource struct {
	client *client.Client
}

type configExportDataSourceModel struct {
	ConfigJSON types.String `tfsdk:"config_json"`
}

// NewConfigExportDataSource constructs a new config export data source.
func NewConfigExportDataSource() datasource.DataSource {
	return &configExportDataSource{}
}

// Metadata sets the data source type name.
func (d *configExportDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_config_export"
}

// Schema defines the config export schema.
func (d *configExportDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"config_json": schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "Full configuration export payload as JSON.",
			},
		},
	}
}

// Configure assigns the API client.
func (d *configExportDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	if client, ok := req.ProviderData.(*client.Client); ok {
		d.client = client
	}
}

// Read exports the configuration.
func (d *configExportDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before using the config export data source.")
		return
	}

	config, err := d.client.ExportConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Export config failed", err.Error())
		return
	}

	configJSON, err := encodeOptionalJSON(config)
	if err != nil {
		resp.Diagnostics.AddError("Serialize config export", err.Error())
		return
	}

	state := configExportDataSourceModel{
		ConfigJSON: configJSON,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
