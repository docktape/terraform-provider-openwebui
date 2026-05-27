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

var _ resource.Resource = &oauthClientResource{}
var _ resource.ResourceWithConfigure = &oauthClientResource{}
var _ resource.ResourceWithImportState = &oauthClientResource{}

// oauthClientResource registers OAuth clients.
type oauthClientResource struct {
	client *client.Client
}

type oauthClientModel struct {
	ID         types.String `tfsdk:"id"`
	URL        types.String `tfsdk:"url"`
	ClientID   types.String `tfsdk:"client_id"`
	ClientName types.String `tfsdk:"client_name"`
	Type       types.String `tfsdk:"type"`
}

// NewOAuthClientResource constructs a new OAuth client resource.
func NewOAuthClientResource() resource.Resource {
	return &oauthClientResource{}
}

// Metadata sets the resource type name.
func (r *oauthClientResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_oauth_client"
}

// Schema defines the OAuth client schema.
func (r *oauthClientResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "OAuth client identifier (mirrors client_id).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"url": schema.StringAttribute{
				Required:    true,
				Description: "OAuth provider URL.",
			},
			"client_id": schema.StringAttribute{
				Required:    true,
				Description: "OAuth client identifier to register.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"client_name": schema.StringAttribute{
				Optional:    true,
				Description: "Optional display name for the OAuth client.",
			},
			"type": schema.StringAttribute{
				Optional:    true,
				Description: "Optional OAuth client type query parameter.",
			},
		},
	}
}

// Configure assigns the API client.
func (r *oauthClientResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	if client, ok := req.ProviderData.(*client.Client); ok {
		r.client = client
	}
}

// Create registers the OAuth client.
func (r *oauthClientResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing OAuth clients.")
		return
	}

	var plan oauthClientModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applyOAuthClientRegistration(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Read preserves current state (no read endpoint available).
func (r *oauthClientResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state oauthClientModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update re-registers the OAuth client.
func (r *oauthClientResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing OAuth clients.")
		return
	}

	var plan oauthClientModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applyOAuthClientRegistration(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete removes the resource from state without changing remote configuration.
func (r *oauthClientResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing OAuth clients.")
		return
	}
}

// ImportState maps import identifiers to client_id.
func (r *oauthClientResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("client_id"), req, resp)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func applyOAuthClientRegistration(ctx context.Context, apiClient *client.Client, plan oauthClientModel) (oauthClientModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	var clientName *string
	if !plan.ClientName.IsNull() && !plan.ClientName.IsUnknown() {
		value := plan.ClientName.ValueString()
		clientName = &value
	}

	form := client.OAuthClientRegistrationForm{
		URL:        plan.URL.ValueString(),
		ClientID:   plan.ClientID.ValueString(),
		ClientName: clientName,
	}

	var clientType *string
	if !plan.Type.IsNull() && !plan.Type.IsUnknown() {
		value := plan.Type.ValueString()
		clientType = &value
	}

	if _, err := apiClient.RegisterOAuthClient(ctx, form, clientType); err != nil {
		diags.AddError("Register OAuth client failed", err.Error())
		return oauthClientModel{}, diags
	}

	state := plan
	state.ID = types.StringValue(plan.ClientID.ValueString())

	return state, diags
}
