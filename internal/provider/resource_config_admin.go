package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

var _ resource.Resource = &adminConfigResource{}
var _ resource.ResourceWithConfigure = &adminConfigResource{}
var _ resource.ResourceWithImportState = &adminConfigResource{}

type adminConfigResource struct {
	client *client.Client
}

type adminConfigModel struct {
	ID                                types.String `tfsdk:"id"`
	ShowAdminDetails                  types.Bool   `tfsdk:"show_admin_details"`
	AdminEmail                        types.String `tfsdk:"admin_email"`
	WebUIURL                          types.String `tfsdk:"webui_url"`
	EnableSignup                      types.Bool   `tfsdk:"enable_signup"`
	EnableAPIKeys                     types.Bool   `tfsdk:"enable_api_keys"`
	EnableAPIKeysEndpointRestrictions types.Bool   `tfsdk:"enable_api_keys_endpoint_restrictions"`
	APIKeysAllowedEndpoints           types.String `tfsdk:"api_keys_allowed_endpoints"`
	DefaultUserRole                   types.String `tfsdk:"default_user_role"`
	DefaultGroupID                    types.String `tfsdk:"default_group_id"`
	JWTExpiresIn                      types.String `tfsdk:"jwt_expires_in"`
	EnableCommunitySharing            types.Bool   `tfsdk:"enable_community_sharing"`
	EnableMessageRating               types.Bool   `tfsdk:"enable_message_rating"`
	EnableFolders                     types.Bool   `tfsdk:"enable_folders"`
	FolderMaxFileCount                types.Int64  `tfsdk:"folder_max_file_count"`
	AutomationMaxCount                types.Int64  `tfsdk:"automation_max_count"`
	AutomationMinInterval             types.Int64  `tfsdk:"automation_min_interval"`
	EnableAutomations                 types.Bool   `tfsdk:"enable_automations"`
	EnableChannels                    types.Bool   `tfsdk:"enable_channels"`
	EnableCalendar                    types.Bool   `tfsdk:"enable_calendar"`
	EnableMemories                    types.Bool   `tfsdk:"enable_memories"`
	EnableMemorySystemContext         types.Bool   `tfsdk:"enable_memory_system_context"`
	EnableNotes                       types.Bool   `tfsdk:"enable_notes"`
	EnableUserWebhooks                types.Bool   `tfsdk:"enable_user_webhooks"`
	EnableUserStatus                  types.Bool   `tfsdk:"enable_user_status"`
	PendingUserOverlayTitle           types.String `tfsdk:"pending_user_overlay_title"`
	PendingUserOverlayContent         types.String `tfsdk:"pending_user_overlay_content"`
	ResponseWatermark                 types.String `tfsdk:"response_watermark"`
}

func NewAdminConfigResource() resource.Resource {
	return &adminConfigResource{}
}

func (r *adminConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_admin_config"
}

func (r *adminConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	b := func(desc string) schema.BoolAttribute {
		return schema.BoolAttribute{Optional: true, Computed: true, Description: desc, MarkdownDescription: desc, PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()}}
	}
	i := func(desc string) schema.Int64Attribute {
		return schema.Int64Attribute{Optional: true, Computed: true, Description: desc, MarkdownDescription: desc, PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}}
	}
	s := func(desc string) schema.StringAttribute {
		return schema.StringAttribute{Optional: true, Computed: true, Description: desc, MarkdownDescription: desc, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}}
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages Open WebUI admin configuration. This is a singleton resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Singleton identifier.",
				MarkdownDescription: "Singleton identifier.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"show_admin_details": b("Whether to show admin details in the UI."),
			"admin_email":        s("Email address of the admin account."),
			"webui_url": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "URL of the Open WebUI instance.",
				MarkdownDescription: "URL of the Open WebUI instance.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"enable_signup":                         b("Whether user signup is enabled."),
			"enable_api_keys":                       b("Whether API key authentication is enabled."),
			"enable_api_keys_endpoint_restrictions": b("Whether API key endpoint restrictions are enabled."),
			"api_keys_allowed_endpoints":            s("Comma-separated list of allowed endpoints for API keys."),
			"default_user_role": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Default role for new users (pending, user, or admin).",
				MarkdownDescription: "Default role for new users (pending, user, or admin).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Validators: []validator.String{
					stringvalidator.OneOf("pending", "user", "admin"),
				},
			},
			"default_group_id":         s("Default group ID for new users."),
			"jwt_expires_in":           s("JWT token expiration duration (e.g., 30d, 24h)."),
			"enable_community_sharing": b("Whether community sharing is enabled."),
			"enable_message_rating":    b("Whether message rating is enabled."),
			"enable_folders":           b("Whether folders are enabled."),
			"folder_max_file_count":    i("Maximum number of files per folder."),
			"automation_max_count":     i("Maximum number of concurrent automations."),
			"automation_min_interval":  i("Minimum interval between automations in milliseconds."),
			"enable_automations":       b("Whether automations are enabled."),
			"enable_channels":          b("Whether channels are enabled."),
			"enable_calendar":          b("Whether calendar is enabled."),
			"enable_memories":          b("Whether memories are enabled."),
			"enable_memory_system_context": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Whether memory system context is enabled. Available in v0.10.2+.",
				MarkdownDescription: "Whether memory system context is enabled. Available in v0.10.2+.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"enable_notes":                 b("Whether notes are enabled."),
			"enable_user_webhooks":         b("Whether user webhooks are enabled."),
			"enable_user_status":           b("Whether user status is enabled."),
			"pending_user_overlay_title":   s("Title for pending user overlay."),
			"pending_user_overlay_content": s("Content for pending user overlay."),
			"response_watermark":           s("Watermark to add to AI responses."),
		},
	}
}

func (r *adminConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	if c, ok := req.ProviderData.(*client.Client); ok {
		r.client = c
	}
}

func (r *adminConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing admin config.")
		return
	}

	var plan adminConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applyAdminConfig(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *adminConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing admin config.")
		return
	}

	var plan adminConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applyAdminConfig(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func applyAdminConfig(ctx context.Context, apiClient *client.Client, plan adminConfigModel) (adminConfigModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	form := adminConfigFormFromModel(plan)
	updated, err := apiClient.SetAdminConfig(ctx, form)
	if err != nil {
		diags.AddError("Update admin config failed", err.Error())
		return adminConfigModel{}, diags
	}

	state := adminConfigModelFromForm(*updated)
	return state, diags
}

func (r *adminConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing admin config.")
		return
	}

	current, err := r.client.GetAdminConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Read admin config failed", err.Error())
		return
	}

	state := adminConfigModelFromForm(*current)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *adminConfigResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// Singleton config: destroying the resource leaves the remote config untouched.
}

func (r *adminConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func adminConfigFormFromModel(m adminConfigModel) client.AdminConfigForm {
	return client.AdminConfigForm{
		ShowAdminDetails:                  m.ShowAdminDetails.ValueBool(),
		AdminEmail:                        stringPtr(m.AdminEmail),
		WebUIURL:                          m.WebUIURL.ValueString(),
		EnableSignup:                      m.EnableSignup.ValueBool(),
		EnableAPIKeys:                     m.EnableAPIKeys.ValueBool(),
		EnableAPIKeysEndpointRestrictions: m.EnableAPIKeysEndpointRestrictions.ValueBool(),
		APIKeysAllowedEndpoints:           m.APIKeysAllowedEndpoints.ValueString(),
		DefaultUserRole:                   m.DefaultUserRole.ValueString(),
		DefaultGroupID:                    m.DefaultGroupID.ValueString(),
		JWTExpiresIn:                      m.JWTExpiresIn.ValueString(),
		EnableCommunitySharing:            m.EnableCommunitySharing.ValueBool(),
		EnableMessageRating:               m.EnableMessageRating.ValueBool(),
		EnableFolders:                     m.EnableFolders.ValueBool(),
		FolderMaxFileCount:                int64Ptr(m.FolderMaxFileCount),
		AutomationMaxCount:                int64Ptr(m.AutomationMaxCount),
		AutomationMinInterval:             int64Ptr(m.AutomationMinInterval),
		EnableAutomations:                 m.EnableAutomations.ValueBool(),
		EnableChannels:                    m.EnableChannels.ValueBool(),
		EnableCalendar:                    m.EnableCalendar.ValueBool(),
		EnableMemories:                    m.EnableMemories.ValueBool(),
		EnableMemorySystemContext:         boolPtr(m.EnableMemorySystemContext),
		EnableNotes:                       m.EnableNotes.ValueBool(),
		EnableUserWebhooks:                m.EnableUserWebhooks.ValueBool(),
		EnableUserStatus:                  m.EnableUserStatus.ValueBool(),
		PendingUserOverlayTitle:           stringPtr(m.PendingUserOverlayTitle),
		PendingUserOverlayContent:         stringPtr(m.PendingUserOverlayContent),
		ResponseWatermark:                 stringPtr(m.ResponseWatermark),
	}
}

func adminConfigModelFromForm(f client.AdminConfigForm) adminConfigModel {
	return adminConfigModel{
		ID:                                types.StringValue("admin"),
		ShowAdminDetails:                  types.BoolValue(f.ShowAdminDetails),
		AdminEmail:                        stringValueOrNull(f.AdminEmail),
		WebUIURL:                          types.StringValue(f.WebUIURL),
		EnableSignup:                      types.BoolValue(f.EnableSignup),
		EnableAPIKeys:                     types.BoolValue(f.EnableAPIKeys),
		EnableAPIKeysEndpointRestrictions: types.BoolValue(f.EnableAPIKeysEndpointRestrictions),
		APIKeysAllowedEndpoints:           types.StringValue(f.APIKeysAllowedEndpoints),
		DefaultUserRole:                   types.StringValue(f.DefaultUserRole),
		DefaultGroupID:                    types.StringValue(f.DefaultGroupID),
		JWTExpiresIn:                      types.StringValue(f.JWTExpiresIn),
		EnableCommunitySharing:            types.BoolValue(f.EnableCommunitySharing),
		EnableMessageRating:               types.BoolValue(f.EnableMessageRating),
		EnableFolders:                     types.BoolValue(f.EnableFolders),
		FolderMaxFileCount:                int64ValueOrNull(f.FolderMaxFileCount),
		AutomationMaxCount:                int64ValueOrNull(f.AutomationMaxCount),
		AutomationMinInterval:             int64ValueOrNull(f.AutomationMinInterval),
		EnableAutomations:                 types.BoolValue(f.EnableAutomations),
		EnableChannels:                    types.BoolValue(f.EnableChannels),
		EnableCalendar:                    types.BoolValue(f.EnableCalendar),
		EnableMemories:                    types.BoolValue(f.EnableMemories),
		EnableMemorySystemContext:         boolValueOrNull(f.EnableMemorySystemContext),
		EnableNotes:                       types.BoolValue(f.EnableNotes),
		EnableUserWebhooks:                types.BoolValue(f.EnableUserWebhooks),
		EnableUserStatus:                  types.BoolValue(f.EnableUserStatus),
		PendingUserOverlayTitle:           stringValueOrNull(f.PendingUserOverlayTitle),
		PendingUserOverlayContent:         stringValueOrNull(f.PendingUserOverlayContent),
		ResponseWatermark:                 stringValueOrNull(f.ResponseWatermark),
	}
}

func boolPtr(value types.Bool) *bool {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	result := value.ValueBool()
	return &result
}

func boolValueOrNull(value *bool) types.Bool {
	if value == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*value)
}
