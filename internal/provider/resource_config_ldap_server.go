package provider

import (
	"context"
	"strings"

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

var _ resource.Resource = &ldapServerConfigResource{}
var _ resource.ResourceWithConfigure = &ldapServerConfigResource{}
var _ resource.ResourceWithImportState = &ldapServerConfigResource{}
var _ resource.ResourceWithValidateConfig = &ldapServerConfigResource{}

type ldapServerConfigResource struct {
	client *client.Client
}

type ldapServerConfigModel struct {
	ID                    types.String `tfsdk:"id"`
	Label                 types.String `tfsdk:"label"`
	Host                  types.String `tfsdk:"host"`
	Port                  types.Int64  `tfsdk:"port"`
	AttributeForMail      types.String `tfsdk:"attribute_for_mail"`
	AttributeForUsername  types.String `tfsdk:"attribute_for_username"`
	AppDN                 types.String `tfsdk:"app_dn"`
	AppDNPassword         types.String `tfsdk:"app_dn_password"`
	SearchBase            types.String `tfsdk:"search_base"`
	SearchFilters         types.String `tfsdk:"search_filters"`
	UseTLS                types.Bool   `tfsdk:"use_tls"`
	CertificatePath       types.String `tfsdk:"certificate_path"`
	ValidateCert          types.Bool   `tfsdk:"validate_cert"`
	Ciphers               types.String `tfsdk:"ciphers"`
	EnableGroupManagement types.Bool   `tfsdk:"enable_group_management"`
	EnableGroupCreation   types.Bool   `tfsdk:"enable_group_creation"`
	AttributeForGroups    types.String `tfsdk:"attribute_for_groups"`
}

func NewLDAPServerConfigResource() resource.Resource {
	return &ldapServerConfigResource{}
}

func (r *ldapServerConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ldap_server_config"
}

func (r *ldapServerConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	str := func(desc string, validators ...validator.String) schema.StringAttribute {
		return schema.StringAttribute{Optional: true, Computed: true, Description: desc, MarkdownDescription: desc, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}, Validators: validators}
	}
	b := func(desc string) schema.BoolAttribute {
		return schema.BoolAttribute{Optional: true, Computed: true, Description: desc, MarkdownDescription: desc, PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()}}
	}
	required := func(desc string) schema.StringAttribute {
		return schema.StringAttribute{Required: true, Description: desc, MarkdownDescription: desc, Validators: []validator.String{stringvalidator.LengthAtLeast(1)}}
	}
	const v011 = " Added in Open WebUI v0.11.0; older releases ignore it and read it back as null, so setting it against an older server fails the apply with an inconsistent-result error."
	appPasswordDesc := "Password used to bind as `app_dn`. When unset, keeps the current server value. Open WebUI returns this value in plain text from its read endpoint, so it is also kept in Terraform state."
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages Open WebUI's LDAP server connection settings (`/api/v1/auths/admin/config/ldap/server`). This is a singleton resource; use `openwebui_ldap_config` to enable or disable LDAP authentication itself.\n\n" +
			"~> Open WebUI replaces its whole LDAP server configuration on every write, so each apply first reads the current configuration and sends it back with your configured values applied. An optional attribute you do not set keeps its current server value on first apply and its last-read (state) value afterwards; it is never reset to a default. Removing a previously-set attribute from your configuration therefore leaves the old value in place: to clear or reset it, set an explicit value (for example `app_dn = \"\"` and `app_dn_password = \"\"` for an anonymous bind). The nullable `port`, `certificate_path` and `ciphers` cannot be set back to null from Terraform once they hold a value.\n\n" +
			"~> `app_dn_password` is stored in Terraform state. Protect your state accordingly.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Singleton identifier.",
				MarkdownDescription: "Singleton identifier.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"label": required("Display label for the LDAP server shown on the sign-in page. Must not be empty."),
			"host":  required("Hostname or IP address of the LDAP server. Must not be empty."),
			"port": schema.Int64Attribute{
				Optional: true, Computed: true,
				Description:         "LDAP server port. When unset, keeps the current server value (null if never set).",
				MarkdownDescription: "LDAP server port. When unset, keeps the current server value (null if never set).",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"attribute_for_mail":     str("LDAP attribute that holds the user's email address. Must not be empty. Server default: `mail`.", stringvalidator.LengthAtLeast(1)),
			"attribute_for_username": str("LDAP attribute that holds the username. Must not be empty. Server default: `uid`.", stringvalidator.LengthAtLeast(1)),
			"app_dn":                 str("Distinguished name of the application (bind) account. An empty string means anonymous bind. When unset, keeps the current server value."),
			"app_dn_password": schema.StringAttribute{
				Optional: true, Computed: true, Sensitive: true,
				Description:         appPasswordDesc,
				MarkdownDescription: appPasswordDesc,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"search_base":             required("Base DN under which users are searched. Must not be empty."),
			"search_filters":          str("LDAP search filter applied when looking up users. Server default: empty string."),
			"use_tls":                 b("Whether to connect using TLS. Server default: `true`."),
			"certificate_path":        str("Path, on the Open WebUI server, to a CA certificate file used to verify the LDAP server. Null if never set."),
			"validate_cert":           b("Whether to validate the LDAP server's TLS certificate. Server default: `true`."),
			"ciphers":                 str("TLS cipher suite string. Server default: `ALL`."),
			"enable_group_management": b("Whether LDAP group membership is synchronised into Open WebUI groups. Requires a non-blank `attribute_for_groups`. Server default: `false`." + v011),
			"enable_group_creation":   b("Whether groups found in LDAP are created in Open WebUI when they do not exist. Server default: `false`." + v011),
			"attribute_for_groups":    str("LDAP attribute that lists a user's group memberships. Must not be blank when `enable_group_management` is `true`. Server default: `memberOf`." + v011),
		},
	}
}

// ValidateConfig rejects enable_group_management = true combined with a blank
// attribute_for_groups, which the backend answers with HTTP 400.
func (r *ldapServerConfigResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var management types.Bool
	var attribute types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("enable_group_management"), &management)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("attribute_for_groups"), &attribute)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if management.IsNull() || management.IsUnknown() || !management.ValueBool() {
		return
	}
	if attribute.IsNull() || attribute.IsUnknown() {
		return
	}
	if strings.TrimSpace(attribute.ValueString()) == "" {
		resp.Diagnostics.AddAttributeError(path.Root("attribute_for_groups"),
			"Invalid attribute_for_groups",
			"attribute_for_groups must be non-blank when enable_group_management is true; Open WebUI rejects the update otherwise.")
	}
}

func (r *ldapServerConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	if c, ok := req.ProviderData.(*client.Client); ok {
		r.client = c
	}
}

func (r *ldapServerConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing LDAP server config.")
		return
	}

	var plan ldapServerConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applyLDAPServerConfig(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ldapServerConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing LDAP server config.")
		return
	}

	var plan ldapServerConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applyLDAPServerConfig(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// applyLDAPServerConfig reads the current server configuration, overlays the
// known plan values and writes the merged object back. The backend POST is a
// full-object replace, so the read keeps every attribute the plan leaves
// unknown (unset on first apply) at its current server value instead of
// resetting it to the Pydantic default.
func applyLDAPServerConfig(ctx context.Context, apiClient *client.Client, plan ldapServerConfigModel) (ldapServerConfigModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	current, err := apiClient.GetLDAPServerConfig(ctx)
	if err != nil {
		diags.AddError("Read LDAP server config failed", err.Error())
		return ldapServerConfigModel{}, diags
	}

	updated, err := apiClient.SetLDAPServerConfig(ctx, overlayLDAPServerConfigPlan(*current, plan))
	if err != nil {
		diags.AddError("Update LDAP server config failed", err.Error())
		return ldapServerConfigModel{}, diags
	}

	return ldapServerConfigModelFromForm(*updated), diags
}

func (r *ldapServerConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing LDAP server config.")
		return
	}

	current, err := r.client.GetLDAPServerConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Read LDAP server config failed", err.Error())
		return
	}

	state := ldapServerConfigModelFromForm(*current)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ldapServerConfigResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing LDAP server config.")
		return
	}
	// Singleton config: destroying the resource leaves the remote config untouched.
}

func (r *ldapServerConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// overlayLDAPServerConfigPlan returns current with every known (non-null,
// non-unknown) plan value applied; every other field keeps the value just
// read from the server. Fields an older server does not return stay nil and
// are omitted from the request.
func overlayLDAPServerConfigPlan(current client.LDAPServerConfigForm, m ldapServerConfigModel) client.LDAPServerConfigForm {
	f := current
	setString := func(dst *string, v types.String) {
		if p := stringPtr(v); p != nil {
			*dst = *p
		}
	}
	setStringPtr := func(dst **string, v types.String) {
		if p := stringPtr(v); p != nil {
			*dst = p
		}
	}
	setBoolPtr := func(dst **bool, v types.Bool) {
		if p := boolPtr(v); p != nil {
			*dst = p
		}
	}
	setString(&f.Label, m.Label)
	setString(&f.Host, m.Host)
	if p := int64Ptr(m.Port); p != nil {
		f.Port = p
	}
	setStringPtr(&f.AttributeForMail, m.AttributeForMail)
	setStringPtr(&f.AttributeForUsername, m.AttributeForUsername)
	setString(&f.AppDN, m.AppDN)
	setString(&f.AppDNPassword, m.AppDNPassword)
	setString(&f.SearchBase, m.SearchBase)
	setStringPtr(&f.SearchFilters, m.SearchFilters)
	setBoolPtr(&f.UseTLS, m.UseTLS)
	setStringPtr(&f.CertificatePath, m.CertificatePath)
	setBoolPtr(&f.ValidateCert, m.ValidateCert)
	setStringPtr(&f.Ciphers, m.Ciphers)
	setBoolPtr(&f.EnableGroupManagement, m.EnableGroupManagement)
	setBoolPtr(&f.EnableGroupCreation, m.EnableGroupCreation)
	setStringPtr(&f.AttributeForGroups, m.AttributeForGroups)
	return f
}

func ldapServerConfigModelFromForm(f client.LDAPServerConfigForm) ldapServerConfigModel {
	return ldapServerConfigModel{
		ID:                    types.StringValue("ldap_server"),
		Label:                 types.StringValue(f.Label),
		Host:                  types.StringValue(f.Host),
		Port:                  int64ValueOrNull(f.Port),
		AttributeForMail:      stringValueOrNull(f.AttributeForMail),
		AttributeForUsername:  stringValueOrNull(f.AttributeForUsername),
		AppDN:                 types.StringValue(f.AppDN),
		AppDNPassword:         types.StringValue(f.AppDNPassword),
		SearchBase:            types.StringValue(f.SearchBase),
		SearchFilters:         stringValueOrNull(f.SearchFilters),
		UseTLS:                boolValueOrNull(f.UseTLS),
		CertificatePath:       stringValueOrNull(f.CertificatePath),
		ValidateCert:          boolValueOrNull(f.ValidateCert),
		Ciphers:               stringValueOrNull(f.Ciphers),
		EnableGroupManagement: boolValueOrNull(f.EnableGroupManagement),
		EnableGroupCreation:   boolValueOrNull(f.EnableGroupCreation),
		AttributeForGroups:    stringValueOrNull(f.AttributeForGroups),
	}
}
