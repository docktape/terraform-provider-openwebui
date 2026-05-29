package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

var _ resource.Resource = &toolServersConfigResource{}
var _ resource.ResourceWithConfigure = &toolServersConfigResource{}
var _ resource.ResourceWithImportState = &toolServersConfigResource{}

// toolServersConfigResource manages tool server configuration.
type toolServersConfigResource struct {
	client *client.Client
}

type toolServerConnectionModel struct {
	URL         types.String `tfsdk:"url"`
	Path        types.String `tfsdk:"path"`
	Type        types.String `tfsdk:"type"`
	AuthType    types.String `tfsdk:"auth_type"`
	HeadersJSON types.String `tfsdk:"headers_json"`
	Key         types.String `tfsdk:"key"`
	ConfigJSON  types.String `tfsdk:"config_json"`
}

type toolServersConfigModel struct {
	ID          types.String                `tfsdk:"id"`
	Connections []toolServerConnectionModel `tfsdk:"connections"`
}

// NewToolServersConfigResource constructs a new tool servers config resource.
func NewToolServersConfigResource() resource.Resource {
	return &toolServersConfigResource{}
}

// Metadata sets the resource type name.
func (r *toolServersConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tool_servers_config"
}

// Schema defines the tool servers config schema.
func (r *toolServersConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the list of external tool server registrations for Open WebUI.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Singleton identifier. Set by Open WebUI.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"connections": schema.ListNestedAttribute{
				Required:    true,
				Description: "List of tool server connection entries.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"url": schema.StringAttribute{
							Required:    true,
							Description: "Base URL of the tool server, e.g. `http://tools.internal:8080`.",
						},
						"path": schema.StringAttribute{
							Required:    true,
							Description: "Path to the tool server's OpenAPI spec, e.g. `/openapi.json`.",
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
							Description: "JSON object of extra HTTP headers to send to the tool server. e.g. `jsonencode({ X-Api-Version = \"2\" })`.",
						},
						"key": schema.StringAttribute{
							Optional:    true,
							Sensitive:   true,
							Description: "API key or bearer token for authenticating with the tool server. Sensitive.",
						},
						"config_json": schema.StringAttribute{
							Optional:    true,
							Description: "Additional JSON configuration sent to the tool server.",
						},
					},
				},
			},
		},
	}
}

// Configure assigns the API client.
func (r *toolServersConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	if client, ok := req.ProviderData.(*client.Client); ok {
		r.client = client
	}
}

// Create updates the tool servers config.
func (r *toolServersConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing tool servers config.")
		return
	}

	var plan toolServersConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applyToolServersConfig(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Read refreshes the tool servers config.
func (r *toolServersConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing tool servers config.")
		return
	}

	config, err := r.client.GetToolServersConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Read tool servers config failed", err.Error())
		return
	}

	state := toolServersConfigModel{
		ID:          types.StringValue("tool_servers"),
		Connections: flattenToolServerConnections(config.Connections),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update applies new tool servers config.
func (r *toolServersConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing tool servers config.")
		return
	}

	var plan toolServersConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applyToolServersConfig(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete removes the resource from state without changing remote configuration.
func (r *toolServersConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing tool servers config.")
		return
	}
}

// ImportState maps import identifiers onto the id attribute.
func (r *toolServersConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func applyToolServersConfig(ctx context.Context, apiClient *client.Client, plan toolServersConfigModel) (toolServersConfigModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	connections := make([]client.ToolServerConnection, 0, len(plan.Connections))
	for i, item := range plan.Connections {
		headers := decodeOptionalJSON(item.HeadersJSON, path.Root("connections").AtListIndex(i).AtName("headers_json"), &diags)
		config := decodeOptionalJSON(item.ConfigJSON, path.Root("connections").AtListIndex(i).AtName("config_json"), &diags)

		var authType *string
		if !item.AuthType.IsNull() && !item.AuthType.IsUnknown() {
			value := item.AuthType.ValueString()
			authType = &value
		}
		var connType *string
		if !item.Type.IsNull() && !item.Type.IsUnknown() {
			value := item.Type.ValueString()
			connType = &value
		}
		var key *string
		if !item.Key.IsNull() && !item.Key.IsUnknown() {
			value := item.Key.ValueString()
			key = &value
		}

		connections = append(connections, client.ToolServerConnection{
			URL:      item.URL.ValueString(),
			Path:     item.Path.ValueString(),
			Type:     connType,
			AuthType: authType,
			Headers:  headers,
			Key:      key,
			Config:   config,
		})
	}

	if diags.HasError() {
		return toolServersConfigModel{}, diags
	}

	updated, err := apiClient.SetToolServersConfig(ctx, client.ToolServersConfigForm{Connections: connections})
	if err != nil {
		diags.AddError("Update tool servers config failed", err.Error())
		return toolServersConfigModel{}, diags
	}

	state := toolServersConfigModel{
		ID:          types.StringValue("tool_servers"),
		Connections: flattenToolServerConnections(updated.Connections),
	}

	return state, diags
}

func flattenToolServerConnections(connections []client.ToolServerConnection) []toolServerConnectionModel {
	result := make([]toolServerConnectionModel, 0, len(connections))
	for _, conn := range connections {
		headersJSON, _ := encodeOptionalJSONValue(conn.Headers)
		configJSON, _ := encodeOptionalJSON(conn.Config)

		item := toolServerConnectionModel{
			URL:         types.StringValue(conn.URL),
			Path:        types.StringValue(conn.Path),
			Type:        types.StringNull(),
			AuthType:    types.StringNull(),
			HeadersJSON: headersJSON,
			Key:         types.StringNull(),
			ConfigJSON:  configJSON,
		}
		if conn.Type != nil {
			item.Type = types.StringValue(*conn.Type)
		}
		if conn.AuthType != nil {
			item.AuthType = types.StringValue(*conn.AuthType)
		}
		if conn.Key != nil {
			item.Key = types.StringValue(*conn.Key)
		}

		result = append(result, item)
	}

	return result
}
