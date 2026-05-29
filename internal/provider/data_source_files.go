package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

var _ datasource.DataSource = &filesDataSource{}
var _ datasource.DataSourceWithConfigure = &filesDataSource{}

// filesDataSource lists files.
type filesDataSource struct {
	client *client.Client
}

type fileSummaryModel struct {
	ID        types.String `tfsdk:"id"`
	Filename  types.String `tfsdk:"filename"`
	Hash      types.String `tfsdk:"hash"`
	UserID    types.String `tfsdk:"user_id"`
	DataJSON  types.String `tfsdk:"data_json"`
	MetaJSON  types.String `tfsdk:"meta_json"`
	CreatedAt types.Int64  `tfsdk:"created_at"`
	UpdatedAt types.Int64  `tfsdk:"updated_at"`
}

type filesDataSourceModel struct {
	Filename types.String       `tfsdk:"filename"`
	Content  types.Bool         `tfsdk:"content"`
	Skip     types.Int64        `tfsdk:"skip"`
	Limit    types.Int64        `tfsdk:"limit"`
	Files    []fileSummaryModel `tfsdk:"files"`
}

// NewFilesDataSource constructs a new files data source.
func NewFilesDataSource() datasource.DataSource {
	return &filesDataSource{}
}

// Metadata sets the data source type name.
func (d *filesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_files"
}

// Schema defines the files data source schema.
func (d *filesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Returns a filtered list of files uploaded to Open WebUI.",
		Attributes: map[string]schema.Attribute{
			"filename": schema.StringAttribute{
				Optional:    true,
				Description: "Filename substring to filter by. Returns all files if omitted.",
			},
			"content": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to include file content metadata in the response.",
			},
			"skip": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Number of results to skip for pagination.",
			},
			"limit": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Maximum number of results to return.",
			},
			"files": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of files matching the filter criteria.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":         schema.StringAttribute{Computed: true, Description: "UUID of the file."},
						"filename":   schema.StringAttribute{Computed: true, Description: "Original filename as stored by Open WebUI."},
						"hash":       schema.StringAttribute{Computed: true, Description: "SHA-256 hash of the file content."},
						"user_id":    schema.StringAttribute{Computed: true, Description: "Owner user identifier."},
						"data_json":  schema.StringAttribute{Computed: true, Description: "JSON data blob associated with the file."},
						"meta_json":  schema.StringAttribute{Computed: true, Description: "JSON metadata blob associated with the file."},
						"created_at": schema.Int64Attribute{Computed: true, Description: "Unix timestamp of when the file was uploaded."},
						"updated_at": schema.Int64Attribute{Computed: true, Description: "Unix timestamp of when the file record was last updated."},
					},
				},
			},
		},
	}
}

// Configure assigns the API client.
func (d *filesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	if client, ok := req.ProviderData.(*client.Client); ok {
		d.client = client
	}
}

// Read lists files.
func (d *filesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before using the files data source.")
		return
	}

	var config filesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	filename := strings.TrimSpace(config.Filename.ValueString())
	includeContent := true
	if !config.Content.IsNull() && !config.Content.IsUnknown() {
		includeContent = config.Content.ValueBool()
	}

	skip := int(config.Skip.ValueInt64())
	if config.Skip.IsNull() || config.Skip.IsUnknown() {
		skip = 0
	}
	limit := int(config.Limit.ValueInt64())
	if config.Limit.IsNull() || config.Limit.IsUnknown() {
		limit = 0
	}

	var files []client.FileModelResponse
	var err error
	if filename != "" {
		files, err = d.client.SearchFiles(ctx, filename, includeContent, skip, limit)
	} else {
		files, err = d.client.ListFiles(ctx, includeContent)
	}
	if err != nil {
		resp.Diagnostics.AddError("List files failed", err.Error())
		return
	}

	items := make([]fileSummaryModel, 0, len(files))
	for _, file := range files {
		dataJSON, err := encodeOptionalJSON(file.Data)
		if err != nil {
			resp.Diagnostics.AddError("Serialize file data", err.Error())
			return
		}
		metaJSON, err := encodeOptionalJSON(fileMetaToMap(file.Meta))
		if err != nil {
			resp.Diagnostics.AddError("Serialize file metadata", err.Error())
			return
		}

		hash := types.StringNull()
		if file.Hash != nil {
			hash = types.StringValue(*file.Hash)
		}

		items = append(items, fileSummaryModel{
			ID:        types.StringValue(file.ID),
			Filename:  types.StringValue(file.Filename),
			Hash:      hash,
			UserID:    types.StringValue(file.UserID),
			DataJSON:  dataJSON,
			MetaJSON:  metaJSON,
			CreatedAt: types.Int64Value(file.CreatedAt),
			UpdatedAt: types.Int64Value(file.UpdatedAt),
		})
	}

	state := filesDataSourceModel{
		Filename: config.Filename,
		Content:  types.BoolValue(includeContent),
		Skip:     types.Int64Value(int64(skip)),
		Limit:    types.Int64Value(int64(limit)),
		Files:    items,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func fileMetaToMap(meta client.FileMeta) map[string]any {
	result := map[string]any{}
	if meta.Name != nil {
		result["name"] = *meta.Name
	}
	if meta.ContentType != nil {
		result["content_type"] = *meta.ContentType
	}
	if meta.Size != nil {
		result["size"] = *meta.Size
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
