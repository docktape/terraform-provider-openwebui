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

var _ resource.Resource = &openAIConnectionsResource{}
var _ resource.ResourceWithConfigure = &openAIConnectionsResource{}
var _ resource.ResourceWithImportState = &openAIConnectionsResource{}

type openAIConnectionsResource struct {
	client *client.Client
}

type openAIConnectionItemModel struct {
	URL            types.String `tfsdk:"url"`
	Key            types.String `tfsdk:"key"`
	Enable         types.Bool   `tfsdk:"enable"`
	ConnectionType types.String `tfsdk:"connection_type"`
	AuthType       types.String `tfsdk:"auth_type"`
	PrefixID       types.String `tfsdk:"prefix_id"`
	ModelIDs       types.List   `tfsdk:"model_ids"`
	Tags           types.List   `tfsdk:"tags"`
	HeadersJSON    types.String `tfsdk:"headers_json"`
	Provider       types.String `tfsdk:"provider"`
	APIVersion     types.String `tfsdk:"api_version"`
	APIType        types.String `tfsdk:"api_type"`
}

type openAIConnectionsModel struct {
	ID          types.String               `tfsdk:"id"`
	Enabled     types.Bool                 `tfsdk:"enabled"`
	Connections []openAIConnectionItemModel `tfsdk:"connections"`
}

// NewOpenAIConnectionsResource constructs a new openai_connections resource.
func NewOpenAIConnectionsResource() resource.Resource {
	return &openAIConnectionsResource{}
}

func (r *openAIConnectionsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_openai_connections"
}

func (r *openAIConnectionsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the list of OpenAI-compatible API connections for Open WebUI. This is a singleton resource; the full list is replaced on every apply.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Singleton identifier.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"enabled": schema.BoolAttribute{
				Required:    true,
				Description: "Global toggle for all OpenAI-compatible connections (`ENABLE_OPENAI_API`).",
			},
			"connections": schema.ListNestedAttribute{
				Required:    true,
				Description: "Ordered list of OpenAI-compatible connection entries. Order determines request priority.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"url": schema.StringAttribute{
							Required:    true,
							Description: "Base URL of the OpenAI-compatible endpoint, e.g. `https://api.openai.com/v1`.",
						},
						"key": schema.StringAttribute{
							Optional:    true,
							Sensitive:   true,
							Description: "API key or bearer token. Sensitive. Set to `null` or omit to leave empty.",
						},
						"enable": schema.BoolAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Whether this specific connection is active. Defaults to `true`.",
						},
						"connection_type": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Connection type label, e.g. `external`. Stored as-is; Open WebUI uses it for model tagging.",
						},
						"auth_type": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Authentication method: `bearer` (default), `none`, `session`, `system_oauth`, `azure_ad`, or `microsoft_entra_id`.",
						},
						"prefix_id": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "String prepended to every model ID from this connection, e.g. `openai`. Empty string disables prefixing.",
						},
						"model_ids": schema.ListAttribute{
							Optional:    true,
							Computed:    true,
							ElementType: types.StringType,
							Description: "Allowlist of model IDs exposed from this connection. Empty list exposes all models.",
						},
						"tags": schema.ListAttribute{
							Optional:    true,
							Computed:    true,
							ElementType: types.StringType,
							Description: "Labels attached to models from this connection.",
						},
						"headers_json": schema.StringAttribute{
							Optional:    true,
							Sensitive:   true,
							Description: "JSON object of custom HTTP headers sent to the endpoint, e.g. `jsonencode({ X-Org = \"acme\" })`.",
						},
						"provider": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Provider hint. Set to `\"azure\"` to enable Azure OpenAI mode.",
						},
						"api_version": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Azure API version string, e.g. `2024-02-01`. Required when `provider = \"azure\"`.",
						},
						"api_type": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "API variant. Set to `\"responses\"` to use the Responses API; leave empty for Chat Completions.",
						},
					},
				},
			},
		},
	}
}

func (r *openAIConnectionsResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	if c, ok := req.ProviderData.(*client.Client); ok {
		r.client = c
	}
}

func (r *openAIConnectionsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing OpenAI connections.")
		return
	}
	var plan openAIConnectionsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	state, diags := applyOpenAIConnections(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *openAIConnectionsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing OpenAI connections.")
		return
	}

	var priorState openAIConnectionsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &priorState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	enabled, entries, err := r.client.GetOpenAIConnections(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Read OpenAI connections failed", err.Error())
		return
	}
	connections, diags := flattenOpenAIConnections(ctx, entries)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve keys from prior state: the API may mask or omit keys on GET.
	// If we had a key in state and the API returned empty, keep the prior value.
	for i := range connections {
		if !connections[i].Key.IsNull() {
			continue // API returned a key; use it
		}
		if i < len(priorState.Connections) && !priorState.Connections[i].Key.IsNull() {
			connections[i].Key = priorState.Connections[i].Key
		}
	}

	state := openAIConnectionsModel{
		ID:          types.StringValue("openai"),
		Enabled:     types.BoolValue(enabled),
		Connections: connections,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *openAIConnectionsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing OpenAI connections.")
		return
	}
	var plan openAIConnectionsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	state, diags := applyOpenAIConnections(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *openAIConnectionsResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// Singleton: remove from state only.
}

func (r *openAIConnectionsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func applyOpenAIConnections(ctx context.Context, apiClient *client.Client, plan openAIConnectionsModel) (openAIConnectionsModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	entries, d := expandOpenAIConnections(ctx, plan.Connections)
	diags.Append(d...)
	if diags.HasError() {
		return openAIConnectionsModel{}, diags
	}

	enabled, updated, err := apiClient.SetOpenAIConnections(ctx, plan.Enabled.ValueBool(), entries)
	if err != nil {
		diags.AddError("Update OpenAI connections failed", err.Error())
		return openAIConnectionsModel{}, diags
	}

	connections, d := flattenOpenAIConnections(ctx, updated)
	diags.Append(d...)
	if diags.HasError() {
		return openAIConnectionsModel{}, diags
	}

	return openAIConnectionsModel{
		ID:          types.StringValue("openai"),
		Enabled:     types.BoolValue(enabled),
		Connections: connections,
	}, diags
}

func expandOpenAIConnections(ctx context.Context, items []openAIConnectionItemModel) ([]client.OpenAIConnectionEntry, diag.Diagnostics) {
	var diags diag.Diagnostics
	entries := make([]client.OpenAIConnectionEntry, 0, len(items))

	for i, item := range items {
		var modelIDs []string
		if !item.ModelIDs.IsNull() && !item.ModelIDs.IsUnknown() {
			diags.Append(item.ModelIDs.ElementsAs(ctx, &modelIDs, false)...)
		}
		if modelIDs == nil {
			modelIDs = []string{}
		}

		var tags []string
		if !item.Tags.IsNull() && !item.Tags.IsUnknown() {
			diags.Append(item.Tags.ElementsAs(ctx, &tags, false)...)
		}
		if tags == nil {
			tags = []string{}
		}

		headers := decodeOptionalJSON(item.HeadersJSON, path.Root("connections").AtListIndex(i).AtName("headers_json"), &diags)

		key := ""
		if !item.Key.IsNull() && !item.Key.IsUnknown() {
			key = item.Key.ValueString()
		}

		enable := true
		if !item.Enable.IsNull() && !item.Enable.IsUnknown() {
			enable = item.Enable.ValueBool()
		}

		entries = append(entries, client.OpenAIConnectionEntry{
			URL: item.URL.ValueString(),
			Key: key,
			Config: client.OpenAIConnectionConfig{
				Enable:         enable,
				Tags:           tags,
				PrefixID:       item.PrefixID.ValueString(),
				ModelIDs:       modelIDs,
				ConnectionType: item.ConnectionType.ValueString(),
				AuthType:       item.AuthType.ValueString(),
				Headers:        headers,
				Provider:       item.Provider.ValueString(),
				APIVersion:     item.APIVersion.ValueString(),
				APIType:        item.APIType.ValueString(),
			},
		})
	}

	return entries, diags
}

func flattenOpenAIConnections(ctx context.Context, entries []client.OpenAIConnectionEntry) ([]openAIConnectionItemModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	result := make([]openAIConnectionItemModel, 0, len(entries))

	for _, entry := range entries {
		modelIDs, d := types.ListValueFrom(ctx, types.StringType, entry.Config.ModelIDs)
		diags.Append(d...)

		tags, d := types.ListValueFrom(ctx, types.StringType, entry.Config.Tags)
		diags.Append(d...)

		var headersJSON types.String
		if len(entry.Config.Headers) > 0 {
			encoded, err := encodeOptionalJSONValue(entry.Config.Headers)
			if err != nil {
				diags.AddError("Failed to encode headers", err.Error())
			}
			headersJSON = encoded
		} else {
			headersJSON = types.StringNull()
		}

		key := types.StringNull()
		if entry.Key != "" {
			key = types.StringValue(entry.Key)
		}

		result = append(result, openAIConnectionItemModel{
			URL:            types.StringValue(entry.URL),
			Key:            key,
			Enable:         types.BoolValue(entry.Config.Enable),
			ConnectionType: types.StringValue(entry.Config.ConnectionType),
			AuthType:       types.StringValue(entry.Config.AuthType),
			PrefixID:       types.StringValue(entry.Config.PrefixID),
			ModelIDs:       modelIDs,
			Tags:           tags,
			HeadersJSON:    headersJSON,
			Provider:       types.StringValue(entry.Config.Provider),
			APIVersion:     types.StringValue(entry.Config.APIVersion),
			APIType:        types.StringValue(entry.Config.APIType),
		})
	}

	return result, diags
}
