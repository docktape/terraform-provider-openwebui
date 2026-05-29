package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

var _ datasource.DataSource = &fileDataSource{}
var _ datasource.DataSourceWithConfigure = &fileDataSource{}

// fileDataSource exposes file metadata.
type fileDataSource struct {
	client *client.Client
}

// fileDataSourceModel maps data source inputs and outputs.
type fileDataSourceModel struct {
	FileID    types.String `tfsdk:"file_id"`
	ID        types.String `tfsdk:"id"`
	Filename  types.String `tfsdk:"filename"`
	Hash      types.String `tfsdk:"hash"`
	UserID    types.String `tfsdk:"user_id"`
	DataJSON  types.String `tfsdk:"data_json"`
	MetaJSON  types.String `tfsdk:"meta_json"`
	CreatedAt types.Int64  `tfsdk:"created_at"`
	UpdatedAt types.Int64  `tfsdk:"updated_at"`
}

// NewFileDataSource constructs a new file data source.
func NewFileDataSource() datasource.DataSource {
	return &fileDataSource{}
}

// Metadata sets the data source type name.
func (d *fileDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_file"
}

// Schema defines the file data source schema.
func (d *fileDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up an uploaded file by its UUID.",
		Attributes: map[string]schema.Attribute{
			"file_id": schema.StringAttribute{
				Required:    true,
				Description: "UUID of the file to look up.",
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "UUID of the file. Set after lookup.",
			},
			"filename": schema.StringAttribute{
				Computed:    true,
				Description: "Original filename as stored by Open WebUI.",
			},
			"hash": schema.StringAttribute{
				Computed:    true,
				Description: "SHA-256 hash of the file content.",
			},
			"user_id": schema.StringAttribute{
				Computed:    true,
				Description: "Owner user identifier.",
			},
			"data_json": schema.StringAttribute{
				Computed:    true,
				Description: "JSON data blob associated with the file as returned by Open WebUI.",
			},
			"meta_json": schema.StringAttribute{
				Computed:    true,
				Description: "JSON metadata blob associated with the file as returned by Open WebUI.",
			},
			"created_at": schema.Int64Attribute{
				Computed:    true,
				Description: "Unix timestamp of when the file was uploaded.",
			},
			"updated_at": schema.Int64Attribute{
				Computed:    true,
				Description: "Unix timestamp of when the file record was last updated.",
			},
		},
	}
}

// Configure assigns the API client.
func (d *fileDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	if client, ok := req.ProviderData.(*client.Client); ok {
		d.client = client
	}
}

// Read retrieves file metadata.
func (d *fileDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before using the file data source.")
		return
	}

	var config fileDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.FileID.IsUnknown() || config.FileID.IsNull() || config.FileID.ValueString() == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("file_id"),
			"Missing file identifier",
			"The file_id argument must be supplied to query an existing file.",
		)
		return
	}

	state, diags := fileStateFromAPI(ctx, d.client, config.FileID.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if state.ID.IsNull() || state.ID.IsUnknown() || state.ID.ValueString() == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("file_id"),
			"File not found",
			"No Open WebUI file was found with the supplied identifier.",
		)
		return
	}

	result := fileDataSourceModel{
		FileID:    types.StringValue(state.ID.ValueString()),
		ID:        state.ID,
		Filename:  state.Filename,
		Hash:      state.Hash,
		UserID:    state.UserID,
		DataJSON:  state.DataJSON,
		MetaJSON:  state.MetaJSON,
		CreatedAt: state.CreatedAt,
		UpdatedAt: state.UpdatedAt,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &result)...)
}
