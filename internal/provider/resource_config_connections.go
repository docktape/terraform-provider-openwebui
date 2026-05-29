package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

var _ resource.Resource = &connectionsConfigResource{}
var _ resource.ResourceWithConfigure = &connectionsConfigResource{}
var _ resource.ResourceWithImportState = &connectionsConfigResource{}

// connectionsConfigResource manages connection settings.
type connectionsConfigResource struct {
	client *client.Client
}

type connectionsConfigModel struct {
	ID                      types.String `tfsdk:"id"`
	EnableDirectConnections types.Bool   `tfsdk:"enable_direct_connections"`
	EnableBaseModelsCache   types.Bool   `tfsdk:"enable_base_models_cache"`
}

// NewConnectionsConfigResource constructs a new connections config resource.
func NewConnectionsConfigResource() resource.Resource {
	return &connectionsConfigResource{}
}

// Metadata sets the resource type name.
func (r *connectionsConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connections_config"
}

// Schema defines the connections config schema.
func (r *connectionsConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages connection settings for Open WebUI. This is a singleton resource that updates global connection configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Singleton identifier. Set by Open WebUI.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"enable_direct_connections": schema.BoolAttribute{
				Required:      true,
				Description:   "Whether direct model connections from the browser are allowed.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"enable_base_models_cache": schema.BoolAttribute{
				Required:      true,
				Description:   "Whether base model lists are cached for performance.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

// Configure assigns the API client.
func (r *connectionsConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	if client, ok := req.ProviderData.(*client.Client); ok {
		r.client = client
	}
}

// Create updates the connections config.
func (r *connectionsConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing connections config.")
		return
	}

	var plan connectionsConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applyConnectionsConfig(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Read refreshes the connections config.
func (r *connectionsConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing connections config.")
		return
	}

	config, err := r.client.GetConnectionsConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Read connections config failed", err.Error())
		return
	}

	state := connectionsConfigModel{
		ID:                      types.StringValue("connections"),
		EnableDirectConnections: types.BoolValue(config.EnableDirectConnections),
		EnableBaseModelsCache:   types.BoolValue(config.EnableBaseModelsCache),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update applies new connections config.
func (r *connectionsConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing connections config.")
		return
	}

	var plan connectionsConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applyConnectionsConfig(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete removes the resource from state without changing remote configuration.
func (r *connectionsConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing connections config.")
		return
	}
}

// ImportState maps import identifiers onto the id attribute.
func (r *connectionsConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func applyConnectionsConfig(ctx context.Context, apiClient *client.Client, plan connectionsConfigModel) (connectionsConfigModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	form := client.ConnectionsConfigForm{
		EnableDirectConnections: plan.EnableDirectConnections.ValueBool(),
		EnableBaseModelsCache:   plan.EnableBaseModelsCache.ValueBool(),
	}

	updated, err := apiClient.SetConnectionsConfig(ctx, form)
	if err != nil {
		diags.AddError("Update connections config failed", err.Error())
		return connectionsConfigModel{}, diags
	}

	state := connectionsConfigModel{
		ID:                      types.StringValue("connections"),
		EnableDirectConnections: types.BoolValue(updated.EnableDirectConnections),
		EnableBaseModelsCache:   types.BoolValue(updated.EnableBaseModelsCache),
	}

	return state, diags
}
