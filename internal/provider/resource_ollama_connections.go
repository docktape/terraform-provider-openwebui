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

var _ resource.Resource = &ollamaConnectionsResource{}
var _ resource.ResourceWithConfigure = &ollamaConnectionsResource{}
var _ resource.ResourceWithImportState = &ollamaConnectionsResource{}

type ollamaConnectionsResource struct {
	client *client.Client
}

type ollamaConnectionItemModel struct {
	URL            types.String `tfsdk:"url"`
	Key            types.String `tfsdk:"key"`
	Enable         types.Bool   `tfsdk:"enable"`
	ConnectionType types.String `tfsdk:"connection_type"`
	PrefixID       types.String `tfsdk:"prefix_id"`
	ModelIDs       types.List   `tfsdk:"model_ids"`
	Tags           types.List   `tfsdk:"tags"`
}

type ollamaConnectionsModel struct {
	ID          types.String               `tfsdk:"id"`
	Enabled     types.Bool                 `tfsdk:"enabled"`
	Connections []ollamaConnectionItemModel `tfsdk:"connections"`
}

// NewOllamaConnectionsResource constructs a new ollama_connections resource.
func NewOllamaConnectionsResource() resource.Resource {
	return &ollamaConnectionsResource{}
}

func (r *ollamaConnectionsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ollama_connections"
}

func (r *ollamaConnectionsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the list of Ollama backend connections for Open WebUI. This is a singleton resource; the full list is replaced on every apply.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Singleton identifier.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"enabled": schema.BoolAttribute{
				Required:    true,
				Description: "Global toggle for all Ollama connections (`ENABLE_OLLAMA_API`).",
			},
			"connections": schema.ListNestedAttribute{
				Required:    true,
				Description: "Ordered list of Ollama connection entries.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"url": schema.StringAttribute{
							Required:    true,
							Description: "Base URL of the Ollama backend, e.g. `http://localhost:11434`.",
						},
						"key": schema.StringAttribute{
							Optional:    true,
							Sensitive:   true,
							Description: "Optional API key for authenticated Ollama instances. Sensitive.",
						},
						"enable": schema.BoolAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Whether this specific connection is active. Defaults to `true`.",
						},
						"connection_type": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Connection type label, e.g. `local`. Stored as-is for model tagging.",
						},
						"prefix_id": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "String prepended to every model name from this connection. Empty string disables prefixing.",
						},
						"model_ids": schema.ListAttribute{
							Optional:    true,
							Computed:    true,
							ElementType: types.StringType,
							Description: "Allowlist of model names exposed from this connection. Empty list exposes all models.",
						},
						"tags": schema.ListAttribute{
							Optional:    true,
							Computed:    true,
							ElementType: types.StringType,
							Description: "Labels attached to models from this connection.",
						},
					},
				},
			},
		},
	}
}

func (r *ollamaConnectionsResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	if c, ok := req.ProviderData.(*client.Client); ok {
		r.client = c
	}
}

func (r *ollamaConnectionsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing Ollama connections.")
		return
	}
	var plan ollamaConnectionsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	state, diags := applyOllamaConnections(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ollamaConnectionsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing Ollama connections.")
		return
	}

	var priorState ollamaConnectionsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &priorState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	enabled, entries, err := r.client.GetOllamaConnections(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Read Ollama connections failed", err.Error())
		return
	}
	connections, diags := flattenOllamaConnections(ctx, entries)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve keys from prior state: the API may mask or omit keys on GET.
	for i := range connections {
		if !connections[i].Key.IsNull() {
			continue // API returned a key; use it
		}
		if i < len(priorState.Connections) && !priorState.Connections[i].Key.IsNull() {
			connections[i].Key = priorState.Connections[i].Key
		}
	}

	state := ollamaConnectionsModel{
		ID:          types.StringValue("ollama"),
		Enabled:     types.BoolValue(enabled),
		Connections: connections,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ollamaConnectionsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing Ollama connections.")
		return
	}
	var plan ollamaConnectionsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	state, diags := applyOllamaConnections(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ollamaConnectionsResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// Singleton: remove from state only.
}

func (r *ollamaConnectionsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func applyOllamaConnections(ctx context.Context, apiClient *client.Client, plan ollamaConnectionsModel) (ollamaConnectionsModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	entries, d := expandOllamaConnections(ctx, plan.Connections)
	diags.Append(d...)
	if diags.HasError() {
		return ollamaConnectionsModel{}, diags
	}

	enabled, updated, err := apiClient.SetOllamaConnections(ctx, plan.Enabled.ValueBool(), entries)
	if err != nil {
		diags.AddError("Update Ollama connections failed", err.Error())
		return ollamaConnectionsModel{}, diags
	}

	connections, d := flattenOllamaConnections(ctx, updated)
	diags.Append(d...)
	if diags.HasError() {
		return ollamaConnectionsModel{}, diags
	}

	return ollamaConnectionsModel{
		ID:          types.StringValue("ollama"),
		Enabled:     types.BoolValue(enabled),
		Connections: connections,
	}, diags
}

func expandOllamaConnections(ctx context.Context, items []ollamaConnectionItemModel) ([]client.OllamaConnectionEntry, diag.Diagnostics) {
	var diags diag.Diagnostics
	entries := make([]client.OllamaConnectionEntry, 0, len(items))

	for _, item := range items {
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

		key := ""
		if !item.Key.IsNull() && !item.Key.IsUnknown() {
			key = item.Key.ValueString()
		}

		enable := true
		if !item.Enable.IsNull() && !item.Enable.IsUnknown() {
			enable = item.Enable.ValueBool()
		}

		entries = append(entries, client.OllamaConnectionEntry{
			URL: item.URL.ValueString(),
			Config: client.OllamaConnectionConfig{
				Enable:         enable,
				Tags:           tags,
				PrefixID:       item.PrefixID.ValueString(),
				ModelIDs:       modelIDs,
				ConnectionType: item.ConnectionType.ValueString(),
				Key:            key,
			},
		})
	}

	return entries, diags
}

func flattenOllamaConnections(ctx context.Context, entries []client.OllamaConnectionEntry) ([]ollamaConnectionItemModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	result := make([]ollamaConnectionItemModel, 0, len(entries))

	for _, entry := range entries {
		modelIDs, d := types.ListValueFrom(ctx, types.StringType, entry.Config.ModelIDs)
		diags.Append(d...)

		tags, d := types.ListValueFrom(ctx, types.StringType, entry.Config.Tags)
		diags.Append(d...)

		key := types.StringNull()
		if entry.Config.Key != "" {
			key = types.StringValue(entry.Config.Key)
		}

		result = append(result, ollamaConnectionItemModel{
			URL:            types.StringValue(entry.URL),
			Key:            key,
			Enable:         types.BoolValue(entry.Config.Enable),
			ConnectionType: types.StringValue(entry.Config.ConnectionType),
			PrefixID:       types.StringValue(entry.Config.PrefixID),
			ModelIDs:       modelIDs,
			Tags:           tags,
		})
	}

	return result, diags
}
