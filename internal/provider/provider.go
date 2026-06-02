package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

var _ provider.Provider = &openWebUIProvider{}

const defaultEndpoint = "http://localhost:3000/api/v1"

// openWebUIProvider defines the provider implementation.
type openWebUIProvider struct {
	version string
}

// providerModel maps provider schema data to Go type.
type providerModel struct {
	Endpoint           types.String `tfsdk:"endpoint"`
	Token              types.String `tfsdk:"token"`
	InsecureSkipVerify types.Bool   `tfsdk:"insecure_skip_verify"`
}

// New instantiates a new provider.
func New() provider.Provider {
	return &openWebUIProvider{
		version: Version,
	}
}

// Metadata satisfies the provider.Provider interface.
func (p *openWebUIProvider) Metadata(_ context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "openwebui"
	resp.Version = p.version
}

// Schema defines the provider-level schema.
func (p *openWebUIProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The **openwebui** provider manages resources in an [Open WebUI](https://openwebui.com) deployment via its REST API.\n\n" +
			"## Resources\n\n" +
			"- `openwebui_knowledge` — knowledge base entries\n" +
			"- `openwebui_model` — custom model definitions\n" +
			"- `openwebui_prompt` — reusable prompt commands\n" +
			"- `openwebui_group` — user groups with permissions\n" +
			"- `openwebui_tool` / `openwebui_tool_valves` — Python tools and their settings\n" +
			"- `openwebui_pipeline` / `openwebui_pipeline_valves` — pipeline registrations and settings\n" +
			"- `openwebui_file` / `openwebui_knowledge_file` — uploaded files and knowledge attachments\n" +
			"- `openwebui_function` / `openwebui_function_valves` — Python functions and their settings\n" +
			"- `openwebui_config_import` — bulk configuration restore\n" +
			"- `openwebui_connections_config` — direct connection settings\n" +
			"- `openwebui_tool_servers_config` — external tool server registrations\n" +
			"- `openwebui_code_execution_config` — code execution and interpreter settings\n" +
			"- `openwebui_models_config` — default model ordering settings\n" +
			"- `openwebui_suggestions_config` — default prompt suggestions\n" +
			"- `openwebui_banners_config` — UI announcement banners\n" +
			"- `openwebui_oauth_client` — OAuth client registrations\n\n" +
			"## Data Sources\n\n" +
			"- `openwebui_model`, `openwebui_knowledge`, `openwebui_prompt`, `openwebui_group`, `openwebui_tool`, `openwebui_pipeline` — look up existing objects by name or ID\n" +
			"- `openwebui_file`, `openwebui_files` — look up uploaded files\n" +
			"- `openwebui_config_export` — export the current Open WebUI configuration\n" +
			"- `openwebui_user` — look up a user by email or ID\n" +
			"- `openwebui_tool_server_verify` — verify connectivity to a tool server\n",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Optional:    true,
				Description: "Base URL for the Open WebUI API, e.g. `https://openwebui.example.com/api/v1`. Defaults to `http://localhost:3000/api/v1`. Can also be set via the `OPENWEBUI_ENDPOINT` environment variable.",
			},
			"token": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Bearer token used to authenticate requests to the Open WebUI API. Can also be set via the `OPENWEBUI_TOKEN` environment variable.",
			},
			"insecure_skip_verify": schema.BoolAttribute{
				Optional:    true,
				Description: "Disable TLS certificate verification. **Not recommended for production use.** Can also be set via the `OPENWEBUI_INSECURE` environment variable.",
			},
		},
	}
}

// Configure prepares the Open WebUI API client for data sources and resources.
func (p *openWebUIProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data providerModel

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := defaultEndpoint
	if !data.Endpoint.IsNull() && !data.Endpoint.IsUnknown() {
		endpoint = data.Endpoint.ValueString()
	} else if envEndpoint := os.Getenv("OPENWEBUI_ENDPOINT"); envEndpoint != "" {
		endpoint = envEndpoint
	}

	token := ""
	if !data.Token.IsNull() && !data.Token.IsUnknown() {
		token = data.Token.ValueString()
	} else if envToken := os.Getenv("OPENWEBUI_TOKEN"); envToken != "" {
		token = envToken
	}

	insecure := false
	if !data.InsecureSkipVerify.IsNull() && !data.InsecureSkipVerify.IsUnknown() {
		insecure = data.InsecureSkipVerify.ValueBool()
	} else if os.Getenv("OPENWEBUI_INSECURE") != "" {
		insecure = true
	}

	if token == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("token"),
			"Missing Open WebUI API token",
			"A valid API token must be supplied via the provider configuration or the OPENWEBUI_TOKEN environment variable.",
		)
		return
	}

	apiClient, err := client.NewClient(endpoint, token, insecure)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create Open WebUI API client",
			err.Error(),
		)
		return
	}

	tflog.Debug(ctx, "Configured Open WebUI provider", map[string]any{
		"endpoint": endpoint,
	})

	resp.ResourceData = apiClient
	resp.DataSourceData = apiClient
}

// Resources defines provider-supported resources.
func (p *openWebUIProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewKnowledgeResource,
		NewModelResource,
		NewPromptResource,
		NewGroupResource,
		NewToolResource,
		NewToolValvesResource,
		NewPipelineResource,
		NewPipelineValvesResource,
		NewFileResource,
		NewKnowledgeFileResource,
		NewConfigImportResource,
		NewConnectionsConfigResource,
		NewToolServersConfigResource,
		NewCodeExecutionConfigResource,
		NewModelsConfigResource,
		NewSuggestionsConfigResource,
		NewBannersConfigResource,
		NewOAuthClientResource,
		NewFunctionResource,
		NewFunctionValvesResource,
		NewOpenAIConnectionsResource,
		NewOllamaConnectionsResource,
	}
}

// DataSources defines provider-supported data sources.
func (p *openWebUIProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewModelDataSource,
		NewKnowledgeDataSource,
		NewGroupDataSource,
		NewPromptDataSource,
		NewToolDataSource,
		NewPipelineDataSource,
		NewFileDataSource,
		NewFilesDataSource,
		NewConfigExportDataSource,
		NewUserDataSource,
		NewToolServerVerifyDataSource,
		NewOpenAIConnectionVerifyDataSource,
		NewOllamaConnectionVerifyDataSource,
	}
}
