package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

var _ datasource.DataSource = &toolDataSource{}
var _ datasource.DataSourceWithConfigure = &toolDataSource{}

// toolDataSource exposes tool details.
type toolDataSource struct {
	client *client.Client
}

// toolDataSourceModel maps data source inputs and outputs.
type toolDataSourceModel struct {
	ToolID       types.String `tfsdk:"tool_id"`
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Content      types.String `tfsdk:"content"`
	Description  types.String `tfsdk:"description"`
	ManifestJSON types.String `tfsdk:"manifest_json"`
	ReadGroups   types.List   `tfsdk:"read_groups"`
	WriteGroups  types.List   `tfsdk:"write_groups"`
	SpecsJSON    types.String `tfsdk:"specs_json"`
	UserID       types.String `tfsdk:"user_id"`
	CreatedAt    types.Int64  `tfsdk:"created_at"`
	UpdatedAt    types.Int64  `tfsdk:"updated_at"`
	WriteAccess  types.Bool   `tfsdk:"write_access"`
}

// NewToolDataSource constructs a new tool data source.
func NewToolDataSource() datasource.DataSource {
	return &toolDataSource{}
}

// Metadata sets the data source type name.
func (d *toolDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tool"
}

// Schema defines the tool data source schema.
func (d *toolDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up an existing tool by its `tool_id`.",
		Attributes: map[string]schema.Attribute{
			"tool_id": schema.StringAttribute{
				Required:            true,
				Description:         "Identifier of the tool to look up.",
				MarkdownDescription: "Identifier of the tool to look up.",
			},
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "UUID assigned by Open WebUI.",
				MarkdownDescription: "UUID assigned by Open WebUI.",
			},
			"name": schema.StringAttribute{
				Computed:            true,
				Description:         "Display name of the tool.",
				MarkdownDescription: "Display name of the tool.",
			},
			"content": schema.StringAttribute{
				Computed:            true,
				Description:         "Python source code of the tool.",
				MarkdownDescription: "Python source code of the tool.",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				Description:         "Short description of the tool.",
				MarkdownDescription: "Short description of the tool.",
			},
			"manifest_json": schema.StringAttribute{
				Computed:            true,
				Description:         "JSON manifest derived from the tool's frontmatter.",
				MarkdownDescription: "JSON manifest derived from the tool's frontmatter.",
			},
			"read_groups": schema.ListAttribute{
				ElementType:         types.StringType,
				Computed:            true,
				Description:         "Read-access group names currently applied to this tool.",
				MarkdownDescription: "Read-access group names currently applied to this tool.",
			},
			"write_groups": schema.ListAttribute{
				ElementType:         types.StringType,
				Computed:            true,
				Description:         "Write-access group names currently applied to this tool.",
				MarkdownDescription: "Write-access group names currently applied to this tool.",
			},
			"specs_json": schema.StringAttribute{
				Computed:            true,
				Description:         "JSON OpenAPI-style specification of the tool's functions.",
				MarkdownDescription: "JSON OpenAPI-style specification of the tool's functions.",
			},
			"user_id": schema.StringAttribute{
				Computed:            true,
				Description:         "Owner user identifier.",
				MarkdownDescription: "Owner user identifier.",
			},
			"created_at": schema.Int64Attribute{
				Computed:            true,
				Description:         "Unix timestamp of creation.",
				MarkdownDescription: "Unix timestamp of creation.",
			},
			"updated_at": schema.Int64Attribute{
				Computed:            true,
				Description:         "Unix timestamp of last update.",
				MarkdownDescription: "Unix timestamp of last update.",
			},
			"write_access": schema.BoolAttribute{
				Computed:            true,
				Description:         "Whether the authenticated user has write access.",
				MarkdownDescription: "Whether the authenticated user has write access.",
			},
		},
	}
}

// Configure assigns the API client.
func (d *toolDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	if client, ok := req.ProviderData.(*client.Client); ok {
		d.client = client
	}
}

// Read retrieves tool details.
func (d *toolDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before using the tool data source.")
		return
	}

	var config toolDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.ToolID.IsUnknown() || config.ToolID.IsNull() || config.ToolID.ValueString() == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("tool_id"),
			"Missing tool identifier",
			"The tool_id argument must be supplied to query an existing tool.",
		)
		return
	}

	access, err := d.client.GetTool(ctx, config.ToolID.ValueString())
	if err != nil {
		if err == client.ErrNotFound {
			resp.Diagnostics.AddAttributeError(
				path.Root("tool_id"),
				"Tool not found",
				"No Open WebUI tool was found with the supplied tool_id.",
			)
			return
		}
		resp.Diagnostics.AddError("Read tool failed", err.Error())
		return
	}

	content, specs, fetchDiags := fetchToolContent(ctx, d.client, access.ID)
	resp.Diagnostics.Append(fetchDiags...)

	state, diags := toolResponseToModel(ctx, d.client, access, content, specs, types.StringNull())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
