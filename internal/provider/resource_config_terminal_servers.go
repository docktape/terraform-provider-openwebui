package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

var _ resource.Resource = &terminalServersConfigResource{}
var _ resource.ResourceWithConfigure = &terminalServersConfigResource{}
var _ resource.ResourceWithImportState = &terminalServersConfigResource{}

const terminalServersConfigID = "terminal_servers"

type terminalServersConfigResource struct {
	client *client.Client
}

type terminalServerConnectionModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	Enabled    types.Bool   `tfsdk:"enabled"`
	URL        types.String `tfsdk:"url"`
	Path       types.String `tfsdk:"path"`
	Key        types.String `tfsdk:"key"`
	AuthType   types.String `tfsdk:"auth_type"`
	ConfigJSON types.String `tfsdk:"config_json"`
	ServerType types.String `tfsdk:"server_type"`
	PolicyID   types.String `tfsdk:"policy_id"`
}

type terminalServersConfigModel struct {
	ID          types.String                    `tfsdk:"id"`
	Connections []terminalServerConnectionModel `tfsdk:"connections"`
}

// NewTerminalServersConfigResource constructs the terminal servers config resource.
func NewTerminalServersConfigResource() resource.Resource {
	return &terminalServersConfigResource{}
}

func (r *terminalServersConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_terminal_servers_config"
}

func (r *terminalServersConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	str := func(desc, def string, validators ...validator.String) schema.StringAttribute {
		return schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString(def), Description: desc, MarkdownDescription: desc, Validators: validators}
	}
	optional := func(desc string, validators ...validator.String) schema.StringAttribute {
		return schema.StringAttribute{Optional: true, Description: desc, MarkdownDescription: desc, Validators: validators}
	}
	const idDesc = "Singleton identifier. Always `terminal_servers`."
	const listDesc = "The complete list of terminal server connections, in order. Open WebUI replaces the whole list on every apply, so connections created outside Terraform are removed, and an empty list removes them all."
	const keyDesc = "API key or bearer token sent to the terminal server when `auth_type` is `bearer`. Stored in Terraform state because Open WebUI returns it in plain text. Defaults to an empty string."

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages Open WebUI's list of admin-configured terminal servers (Open Terminal instances or orchestrators), the `TERMINAL_SERVER_CONNECTIONS` setting.\n\n" +
			"~> Open WebUI replaces the whole connection list on every apply. Entries added in the admin UI that are not in your configuration are removed, and extra per-connection keys Open WebUI stores but this resource does not model (for example a cached `policy`) are dropped.\n\n" +
			"~> Each apply makes Open WebUI contact every enabled connection to cache its OpenAPI specification. Unreachable servers do not fail the apply, but an apply against many slow or unreachable servers takes correspondingly longer.\n\n" +
			"~> Connection keys are stored in Terraform state because Open WebUI returns them in plain text. Protect your state accordingly.\n\n" +
			"Deleting this resource only removes it from state; the connections stay configured in Open WebUI. The endpoint is available in Open WebUI v0.9.6 and later.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         idDesc,
				MarkdownDescription: idDesc,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"connections": schema.ListNestedAttribute{
				Required:            true,
				Description:         listDesc,
				MarkdownDescription: listDesc,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":   str("Identifier Open WebUI uses to refer to the terminal (for example in chats and automations). Open WebUI does not generate one, so set a unique value; an empty id makes the terminal unusable. Defaults to an empty string.", ""),
						"name": str("Display name. Defaults to an empty string.", ""),
						"enabled": schema.BoolAttribute{
							Optional: true, Computed: true, Default: booldefault.StaticBool(true),
							Description:         "Whether the terminal is enabled. Defaults to `true`.",
							MarkdownDescription: "Whether the terminal is enabled. Defaults to `true`.",
						},
						"url": schema.StringAttribute{
							Required:            true,
							Description:         "Base URL of the terminal server, e.g. `http://terminal:8000`.",
							MarkdownDescription: "Base URL of the terminal server, e.g. `http://terminal:8000`.",
							Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
						},
						"path": str("Path of the server's OpenAPI specification relative to `url`. Defaults to `/openapi.json`.", "/openapi.json"),
						"key": schema.StringAttribute{
							Optional: true, Computed: true, Sensitive: true, Default: stringdefault.StaticString(""),
							Description: keyDesc, MarkdownDescription: keyDesc,
						},
						"auth_type": str("How Open WebUI authenticates to the terminal server: `bearer` (send `key`), `none`, `session` (forward the signed-in user's token) or `system_oauth` (forward the user's OAuth token). Defaults to `bearer`.", "bearer",
							stringvalidator.OneOf("bearer", "none", "session", "system_oauth")),
						"config_json": optional("Extra connection configuration as a JSON object, e.g. `jsonencode({ access_grants = [...] })`. Open WebUI reads `access_grants` here to decide which users and groups may use the terminal; without grants the terminal is admin-only. Null sends no configuration.", jsonObjectValidator()),
						"server_type": optional("Server type: `orchestrator` for a policy-managing orchestrator or `terminal` for a plain Open Terminal. Null leaves it unset."),
						"policy_id":   optional("Policy identifier on an orchestrator; when set, requests are routed to `<url>/p/<policy_id>`. Null leaves it unset."),
					},
				},
			},
		},
	}
}

func (r *terminalServersConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	if c, ok := req.ProviderData.(*client.Client); ok {
		r.client = c
	}
}

func (r *terminalServersConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing terminal servers config.")
		return
	}
	var plan terminalServersConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	state, diags := applyTerminalServersConfig(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *terminalServersConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing terminal servers config.")
		return
	}
	var prior terminalServersConfigModel
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}
	config, err := r.client.GetTerminalServersConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Read terminal servers config failed", err.Error())
		return
	}
	state := terminalServersConfigModel{
		ID:          types.StringValue(terminalServersConfigID),
		Connections: preserveTerminalConnectionJSON(prior.Connections, flattenTerminalServerConnections(config.Connections)),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *terminalServersConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing terminal servers config.")
		return
	}
	var plan terminalServersConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	state, diags := applyTerminalServersConfig(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete removes the resource from state without changing remote configuration.
func (r *terminalServersConfigResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

func (r *terminalServersConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func stringOrEmpty(p *string) types.String {
	if p == nil {
		return types.StringValue("")
	}
	return types.StringValue(*p)
}

// expandTerminalServerConnection builds the wire form of one planned
// connection. config_json is sent in canonical form with its number literals
// kept verbatim, so large integers and decimals reach Open WebUI unchanged.
func expandTerminalServerConnection(item terminalServerConnectionModel, index int, diags *diag.Diagnostics) client.TerminalServerConnection {
	var config json.RawMessage
	if !item.ConfigJSON.IsNull() && !item.ConfigJSON.IsUnknown() {
		canonical, err := canonicalJSON([]byte(item.ConfigJSON.ValueString()))
		if err != nil {
			diags.AddAttributeError(path.Root("connections").AtListIndex(index).AtName("config_json"), "Invalid JSON value",
				fmt.Sprintf("Expected config_json to contain valid JSON object text: %v", err))
		} else {
			config = json.RawMessage(canonical)
		}
	}
	return client.TerminalServerConnection{
		ID:         stringPtr(item.ID),
		Name:       stringPtr(item.Name),
		Enabled:    boolPtr(item.Enabled),
		URL:        item.URL.ValueString(),
		Path:       stringPtr(item.Path),
		Key:        stringPtr(item.Key),
		AuthType:   stringPtr(item.AuthType),
		Config:     config,
		ServerType: stringPtr(item.ServerType),
		PolicyID:   stringPtr(item.PolicyID),
	}
}

func applyTerminalServersConfig(ctx context.Context, apiClient *client.Client, plan terminalServersConfigModel) (terminalServersConfigModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	connections := make([]client.TerminalServerConnection, 0, len(plan.Connections))
	for i, item := range plan.Connections {
		connections = append(connections, expandTerminalServerConnection(item, i, &diags))
	}
	if diags.HasError() {
		return terminalServersConfigModel{}, diags
	}

	updated, err := apiClient.SetTerminalServersConfig(ctx, client.TerminalServersConfigForm{Connections: connections})
	if err != nil {
		diags.AddError("Update terminal servers config failed", err.Error())
		return terminalServersConfigModel{}, diags
	}

	return terminalServersConfigModel{
		ID:          types.StringValue(terminalServersConfigID),
		Connections: preserveTerminalConnectionJSON(plan.Connections, flattenTerminalServerConnections(updated.Connections)),
	}, diags
}

// flattenTerminalServerConnections maps the server's list into state. It always
// returns a non-nil slice so an empty or null server list is state `[]`.
func flattenTerminalServerConnections(connections []client.TerminalServerConnection) []terminalServerConnectionModel {
	result := make([]terminalServerConnectionModel, 0, len(connections))
	for _, conn := range connections {
		// Open WebUI treats a stored null `enabled` as disabled
		// (dict.get returns None, not the default).
		enabled := conn.Enabled != nil && *conn.Enabled
		result = append(result, terminalServerConnectionModel{
			ID:         stringOrEmpty(conn.ID),
			Name:       stringOrEmpty(conn.Name),
			Enabled:    types.BoolValue(enabled),
			URL:        types.StringValue(conn.URL),
			Path:       stringOrEmpty(conn.Path),
			Key:        stringOrEmpty(conn.Key),
			AuthType:   stringOrEmpty(conn.AuthType),
			ConfigJSON: rawJSONToState(conn.Config),
			ServerType: stringValueOrNull(conn.ServerType),
			PolicyID:   stringValueOrNull(conn.PolicyID),
		})
	}
	return result
}

// preserveTerminalConnectionJSON keeps the user's own config_json formatting
// (key order, whitespace) for entries at the same index whose JSON is
// semantically unchanged.
func preserveTerminalConnectionJSON(preferred, fresh []terminalServerConnectionModel) []terminalServerConnectionModel {
	for i := range fresh {
		if i < len(preferred) {
			fresh[i].ConfigJSON = preserveEquivalentJSON(preferred[i].ConfigJSON, fresh[i].ConfigJSON)
		}
	}
	return fresh
}
