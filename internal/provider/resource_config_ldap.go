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

var _ resource.Resource = &ldapConfigResource{}
var _ resource.ResourceWithConfigure = &ldapConfigResource{}
var _ resource.ResourceWithImportState = &ldapConfigResource{}

type ldapConfigResource struct {
	client *client.Client
}

type ldapConfigModel struct {
	ID         types.String `tfsdk:"id"`
	EnableLDAP types.Bool   `tfsdk:"enable_ldap"`
}

func NewLDAPConfigResource() resource.Resource {
	return &ldapConfigResource{}
}

func (r *ldapConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ldap_config"
}

func (r *ldapConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages Open WebUI LDAP enable toggle. This is a singleton resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Singleton identifier.",
				MarkdownDescription: "Singleton identifier.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"enable_ldap": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Whether LDAP authentication is enabled.",
				MarkdownDescription: "Whether LDAP authentication is enabled.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *ldapConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	if c, ok := req.ProviderData.(*client.Client); ok {
		r.client = c
	}
}

func (r *ldapConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing LDAP config.")
		return
	}

	var plan ldapConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applyLDAPConfig(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ldapConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing LDAP config.")
		return
	}

	var plan ldapConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applyLDAPConfig(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func applyLDAPConfig(ctx context.Context, apiClient *client.Client, plan ldapConfigModel) (ldapConfigModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	form := client.LDAPConfigForm{
		EnableLDAP: boolPtr(plan.EnableLDAP),
	}

	updated, err := apiClient.SetLDAPConfig(ctx, form)
	if err != nil {
		diags.AddError("Update LDAP config failed", err.Error())
		return ldapConfigModel{}, diags
	}

	state := ldapConfigModel{
		ID:         types.StringValue("ldap"),
		EnableLDAP: boolValueOrNull(updated.EnableLDAP),
	}

	return state, diags
}

func (r *ldapConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing LDAP config.")
		return
	}

	current, err := r.client.GetLDAPConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Read LDAP config failed", err.Error())
		return
	}

	state := ldapConfigModel{
		ID:         types.StringValue("ldap"),
		EnableLDAP: boolValueOrNull(current.EnableLDAP),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ldapConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing LDAP config.")
		return
	}
	// Singleton config: destroying the resource leaves the remote config untouched.
}

func (r *ldapConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
