package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

var _ datasource.DataSource = &pipelineDataSource{}
var _ datasource.DataSourceWithConfigure = &pipelineDataSource{}

// pipelineDataSource exposes pipeline details.
type pipelineDataSource struct {
	client *client.Client
}

// pipelineDataSourceModel maps data source inputs and outputs.
type pipelineDataSourceModel struct {
	PipelineID  types.String `tfsdk:"pipeline_id"`
	URL         types.String `tfsdk:"url"`
	URLIdx      types.Int64  `tfsdk:"url_idx"`
	ID          types.String `tfsdk:"id"`
	DetailsJSON types.String `tfsdk:"details_json"`
}

// NewPipelineDataSource constructs a new pipeline data source.
func NewPipelineDataSource() datasource.DataSource {
	return &pipelineDataSource{}
}

// Metadata sets the data source type name.
func (d *pipelineDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pipeline"
}

// Schema defines the pipeline data source schema.
func (d *pipelineDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"pipeline_id": schema.StringAttribute{
				Optional:    true,
				Description: "Pipeline identifier to look up.",
			},
			"url": schema.StringAttribute{
				Optional:    true,
				Description: "Pipeline URL to look up when pipeline_id is not supplied.",
			},
			"url_idx": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Pipeline URL index (defaults to 0).",
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier assigned by Open WebUI.",
			},
			"details_json": schema.StringAttribute{
				Computed:    true,
				Description: "Raw JSON describing the pipeline returned by the API.",
			},
		},
	}
}

// Configure assigns the API client.
func (d *pipelineDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	if client, ok := req.ProviderData.(*client.Client); ok {
		d.client = client
	}
}

// Read retrieves pipeline details.
func (d *pipelineDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before using the pipeline data source.")
		return
	}

	var config pipelineDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	pipelineID := ""
	if !config.PipelineID.IsNull() && !config.PipelineID.IsUnknown() {
		pipelineID = config.PipelineID.ValueString()
	}

	urlValue := ""
	if !config.URL.IsNull() && !config.URL.IsUnknown() {
		urlValue = config.URL.ValueString()
	}

	if pipelineID == "" && urlValue == "" {
		resp.Diagnostics.AddError(
			"Missing pipeline lookup value",
			"Either pipeline_id or url must be provided to query a pipeline.",
		)
		return
	}

	urlIdx := int(config.URLIdx.ValueInt64())
	if config.URLIdx.IsNull() || config.URLIdx.IsUnknown() {
		urlIdx = 0
	}

	item, details, err := findPipeline(ctx, d.client, pipelineID, urlValue, urlIdx)
	if err != nil {
		resp.Diagnostics.AddError("Read pipeline failed", err.Error())
		return
	}
	if item == nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("pipeline_id"),
			"Pipeline not found",
			"No Open WebUI pipeline was found with the supplied identifier.",
		)
		return
	}

	resolvedID := pipelineString(item, "id", "pipeline_id", "pipelineId", "name")
	state := pipelineDataSourceModel{
		PipelineID:  types.StringValue(resolvedID),
		URL:         config.URL,
		URLIdx:      types.Int64Value(int64(urlIdx)),
		ID:          types.StringValue(resolvedID),
		DetailsJSON: details,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
