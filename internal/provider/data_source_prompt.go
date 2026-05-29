package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

var _ datasource.DataSource = &promptDataSource{}
var _ datasource.DataSourceWithConfigure = &promptDataSource{}

// promptDataSource exposes prompt definitions.
type promptDataSource struct {
	client *client.Client
}

// promptDataSourceModel reuses the resource representation.
type promptDataSourceModel = promptResourceModel

// NewPromptDataSource constructs a new prompt data source.
func NewPromptDataSource() datasource.DataSource {
	return &promptDataSource{}
}

// Metadata sets the data source identifier.
func (d *promptDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_prompt"
}

// Schema describes the prompt data source schema.
func (d *promptDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up an existing prompt by its slash-command.",
		Attributes: map[string]schema.Attribute{
			"command": schema.StringAttribute{
				Required:    true,
				Description: "Slash-command to look up, e.g. `summarize` or `/summarize`.",
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Server-assigned UUID for the prompt.",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "Display name of the prompt.",
			},
			"content": schema.StringAttribute{
				Computed:    true,
				Description: "Prompt template text.",
			},
			"read_groups": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "Read-access group names currently applied to this prompt.",
			},
			"write_groups": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "Write-access group names currently applied to this prompt.",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "Creation date in `YYYY-MM-DD` format.",
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "Last-updated date in `YYYY-MM-DD` format.",
			},
			"user_id": schema.StringAttribute{
				Computed:    true,
				Description: "Owner user identifier.",
			},
		},
	}
}

// Configure attaches the API client.
func (d *promptDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	if client, ok := req.ProviderData.(*client.Client); ok {
		d.client = client
	}
}

// Read retrieves the prompt definition by matching its command in the prompt list.
func (d *promptDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before using the prompt data source.")
		return
	}

	var config promptDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	command := config.Command.ValueString()
	if config.Command.IsUnknown() || config.Command.IsNull() || command == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("command"),
			"Missing prompt command",
			"The command argument must be supplied to query an existing prompt.",
		)
		return
	}

	prompts, err := d.client.ListPrompts(ctx)
	if err != nil {
		resp.Diagnostics.AddError("List prompts failed", err.Error())
		return
	}

	var match *client.PromptModel
	for i := range prompts {
		if prompts[i].Command == command {
			match = &prompts[i]
			break
		}
	}

	if match == nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("command"),
			"Prompt not found",
			"No Open WebUI prompt was found with the supplied command.",
		)
		return
	}

	state, diags := promptResponseToModel(ctx, d.client, match)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
