package provider

import (
	"context"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

var _ resource.Resource = &oauthConfigResource{}
var _ resource.ResourceWithConfigure = &oauthConfigResource{}
var _ resource.ResourceWithImportState = &oauthConfigResource{}

// commaListValidator accepts the empty string or a comma-separated list whose
// items are non-empty and have no leading or trailing whitespace. The backend
// splits on commas, trims each item, drops empty ones and re-joins with plain
// commas on read, so any other shape would not round-trip.
var commaListValidator = stringvalidator.RegexMatches(
	regexp.MustCompile(`^$|^[^,\s]([^,]*[^,\s])?(,[^,\s]([^,]*[^,\s])?)*$`),
	"must be a comma-separated list of non-empty items without leading or trailing whitespace, or an empty string",
)

// secondsValidator accepts a canonical non-negative integer string (no leading zeros, so it round-trips through a JSON number) or the empty string.
var secondsValidator = stringvalidator.RegexMatches(
	regexp.MustCompile(`^(0|[1-9][0-9]*)?$`),
	"must be a non-negative whole number of seconds without leading zeros, or an empty string",
)

type oauthConfigResource struct {
	client *client.Client
}

type oauthConfigModel struct {
	ID                            types.String `tfsdk:"id"`
	EnableOAuth                   types.Bool   `tfsdk:"enable_oauth"`
	EnableOAuthSignup             types.Bool   `tfsdk:"enable_oauth_signup"`
	OAuthMergeAccountsByEmail     types.Bool   `tfsdk:"oauth_merge_accounts_by_email"`
	OAuthAutoRedirect             types.Bool   `tfsdk:"oauth_auto_redirect"`
	OAuthAllowedDomains           types.String `tfsdk:"oauth_allowed_domains"`
	OAuthBlockedGroups            types.String `tfsdk:"oauth_blocked_groups"`
	EnableOAuthRoleManagement     types.Bool   `tfsdk:"enable_oauth_role_management"`
	OAuthRolesClaim               types.String `tfsdk:"oauth_roles_claim"`
	OAuthAdminRoles               types.String `tfsdk:"oauth_admin_roles"`
	OAuthAllowedRoles             types.String `tfsdk:"oauth_allowed_roles"`
	EnableOAuthGroupManagement    types.Bool   `tfsdk:"enable_oauth_group_management"`
	EnableOAuthGroupCreation      types.Bool   `tfsdk:"enable_oauth_group_creation"`
	OAuthGroupClaim               types.String `tfsdk:"oauth_group_claim"`
	OAuthGroupDefaultShare        types.String `tfsdk:"oauth_group_default_share"`
	OAuthProviderName             types.String `tfsdk:"oauth_provider_name"`
	OpenIDProviderURL             types.String `tfsdk:"openid_provider_url"`
	OAuthClientID                 types.String `tfsdk:"oauth_client_id"`
	OAuthClientSecret             types.String `tfsdk:"oauth_client_secret"`
	OpenIDRedirectURI             types.String `tfsdk:"openid_redirect_uri"`
	OAuthScopes                   types.String `tfsdk:"oauth_scopes"`
	OAuthCodeChallengeMethod      types.String `tfsdk:"oauth_code_challenge_method"`
	OAuthTokenEndpointAuthMethod  types.String `tfsdk:"oauth_token_endpoint_auth_method"`
	OpenIDEndSessionEndpoint      types.String `tfsdk:"openid_end_session_endpoint"`
	OAuthTimeout                  types.String `tfsdk:"oauth_timeout"`
	OAuthClientTimeout            types.String `tfsdk:"oauth_client_timeout"`
	OAuthEmailClaim               types.String `tfsdk:"oauth_email_claim"`
	OAuthUsernameClaim            types.String `tfsdk:"oauth_username_claim"`
	OAuthPictureClaim             types.String `tfsdk:"oauth_picture_claim"`
	OAuthSubClaim                 types.String `tfsdk:"oauth_sub_claim"`
	OAuthAudience                 types.String `tfsdk:"oauth_audience"`
	OAuthUpdateEmailOnLogin       types.Bool   `tfsdk:"oauth_update_email_on_login"`
	OAuthUpdateNameOnLogin        types.Bool   `tfsdk:"oauth_update_name_on_login"`
	OAuthUpdatePictureOnLogin     types.Bool   `tfsdk:"oauth_update_picture_on_login"`
	OAuthRefreshTokenIncludeScope types.Bool   `tfsdk:"oauth_refresh_token_include_scope"`
}

func NewOAuthConfigResource() resource.Resource {
	return &oauthConfigResource{}
}

func (r *oauthConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_oauth_config"
}

func (r *oauthConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	str := func(desc string, validators ...validator.String) schema.StringAttribute {
		return schema.StringAttribute{Optional: true, Computed: true, Description: desc, MarkdownDescription: desc, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}, Validators: validators}
	}
	b := func(desc string) schema.BoolAttribute {
		return schema.BoolAttribute{Optional: true, Computed: true, Description: desc, MarkdownDescription: desc, PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()}}
	}
	secret := func(desc string) schema.StringAttribute {
		return schema.StringAttribute{Optional: true, Computed: true, Sensitive: true, Description: desc, MarkdownDescription: desc, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}}
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages Open WebUI's single generic OIDC provider configuration and cross-cutting OAuth behavior (role/group mapping, domain allowlist, claims). Configuring per-provider (Google/Microsoft/GitHub) client credentials is done via environment variables, not this resource.\n\n" +
			"~> **Requires Open WebUI v0.10.0+.** This endpoint did not exist in earlier releases.\n\n" +
			"~> This resource applies a partial update: only attributes that have a value in the plan are sent, and everything else is left untouched on the server. Removing an attribute from your configuration therefore does not reset it to its default; set it to the value you want instead (an empty string clears a string attribute). Open WebUI cannot store null for these settings.\n\n" +
			"~> `oauth_client_secret` is stored in Terraform state because Open WebUI returns it in plain text. Protect your state accordingly.\n\n" +
			"~> Be careful with `oauth_auto_redirect` and `enable_oauth`: a misconfigured provider combined with auto-redirect can lock password-based admins out of the sign-in page.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Singleton identifier.",
				MarkdownDescription: "Singleton identifier.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"enable_oauth":                      b("Master switch for OAuth login. Server default: `true`. Added in Open WebUI v0.11.0; v0.10.x ignores it and reads it back as null, so setting it against a v0.10.x server fails the apply with an inconsistent-result error."),
			"enable_oauth_signup":               b("Whether new users may sign up through OAuth. Server default: `false`."),
			"oauth_merge_accounts_by_email":     b("Whether an OAuth login is merged into an existing local account with the same email. Only enable this for identity providers that verify email ownership. Server default: `false`."),
			"oauth_auto_redirect":               b("Whether the sign-in page redirects straight to the OAuth provider. Enabling it can lock password-based admins out of the UI. Server default: `false`."),
			"oauth_allowed_domains":             str("Comma-separated list of allowed email domains; `*` allows all. Items must be non-empty with no leading or trailing whitespace, because the server trims and re-joins them with plain commas. An empty string stores an empty list. Server default: `*`.", commaListValidator),
			"oauth_blocked_groups":              str("Blocked OAuth groups, stored as a string (the server default is the JSON list text `[]`)."),
			"enable_oauth_role_management":      b("Whether roles are mapped from the provider's roles claim. Server default: `false`."),
			"oauth_roles_claim":                 str("Claim that carries the user's roles. Server default: `roles`."),
			"oauth_admin_roles":                 str("Comma-separated provider roles that map to the Open WebUI admin role. Items must be non-empty with no leading or trailing whitespace. Server default: `admin`.", commaListValidator),
			"oauth_allowed_roles":               str("Comma-separated provider roles allowed to sign in. Items must be non-empty with no leading or trailing whitespace. Server default: `user,admin`.", commaListValidator),
			"enable_oauth_group_management":     b("Whether group membership is synchronised from the provider's group claim. Server default: `false`."),
			"enable_oauth_group_creation":       b("Whether groups found in the claim are created when missing. Server default: `false`."),
			"oauth_group_claim":                 str("Claim that carries the user's groups. Server default: `groups`."),
			"oauth_group_default_share":         str("Default sharing for groups created from OAuth claims. One of `true`, `false` or `members`. The server stores a boolean for `true`/`false` and the string `members`; the attribute is always a string in Terraform. Server default: `true`.", stringvalidator.OneOf("true", "false", "members")),
			"oauth_provider_name":               str("Display name of the OIDC provider on the sign-in button. Server default: `SSO`."),
			"openid_provider_url":               str("OIDC discovery URL (`.well-known/openid-configuration`). Empty means no generic OIDC provider is configured."),
			"oauth_client_id":                   str("OAuth client ID of the generic OIDC provider."),
			"oauth_client_secret":               secret("OAuth client secret of the generic OIDC provider. Open WebUI returns it in plain text from its read endpoint, so it is also kept in Terraform state."),
			"openid_redirect_uri":               str("Redirect URI registered with the provider. Empty lets Open WebUI derive it."),
			"oauth_scopes":                      str("Space-separated scopes requested from the provider. Server default: `openid email profile`."),
			"oauth_code_challenge_method":       str("PKCE code challenge method. Only `S256` is supported by Open WebUI; an empty string disables PKCE. Null until set.", stringvalidator.OneOf("", "S256")),
			"oauth_token_endpoint_auth_method":  str("Client authentication method at the token endpoint (for example `client_secret_post`). Null until set."),
			"openid_end_session_endpoint":       str("Provider end-session (logout) endpoint. Empty means none."),
			"oauth_timeout":                     str("HTTP timeout in seconds for provider requests, as a whole-number string without leading zeros (for example `\"30\"`), or an empty string for no explicit timeout. The server accepts an integer or a string; Terraform always sees a string. Server default: empty string.", secondsValidator),
			"oauth_client_timeout":              str("HTTP client timeout in seconds, as a whole-number string without leading zeros, or an empty string for none. Server default: empty string.", secondsValidator),
			"oauth_email_claim":                 str("Claim that carries the user's email. Server default: `email`."),
			"oauth_username_claim":              str("Claim that carries the user's display name. Server default: `name`."),
			"oauth_picture_claim":               str("Claim that carries the user's profile picture URL. Server default: `picture`."),
			"oauth_sub_claim":                   str("Claim used as the subject identifier. Null until set (the provider default `sub` applies)."),
			"oauth_audience":                    str("Audience sent to the provider. Empty means none."),
			"oauth_update_email_on_login":       b("Whether the user's email is refreshed from the provider on every login. Server default: `false`."),
			"oauth_update_name_on_login":        b("Whether the user's name is refreshed on every login. Server default: `false`."),
			"oauth_update_picture_on_login":     b("Whether the user's picture is refreshed on every login. Server default: `false`."),
			"oauth_refresh_token_include_scope": b("Whether the scope is included in refresh-token requests. Server default: `false`."),
		},
	}
}

func (r *oauthConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	if c, ok := req.ProviderData.(*client.Client); ok {
		r.client = c
	}
}

func (r *oauthConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing OAuth config.")
		return
	}

	var plan oauthConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applyOAuthConfig(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *oauthConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing OAuth config.")
		return
	}

	var plan oauthConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applyOAuthConfig(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// applyOAuthConfig sends the plan as a partial update. Null and unknown
// attributes are omitted from the request, so the server keeps its stored
// value for them (no fetch-then-merge is needed because the backend stores
// model_dump(exclude_none=True)). The returned state is flattened from the
// server's response, which carries every stored value including the ones the
// plan left unknown.
func applyOAuthConfig(ctx context.Context, apiClient *client.Client, plan oauthConfigModel) (oauthConfigModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	updated, err := apiClient.SetOAuthConfig(ctx, oauthConfigFormFromModel(plan))
	if err != nil {
		diags.AddError("Update OAuth config failed", err.Error())
		return oauthConfigModel{}, diags
	}

	return oauthConfigModelFromForm(*updated), diags
}

func (r *oauthConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing OAuth config.")
		return
	}

	current, err := r.client.GetOAuthConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Read OAuth config failed", err.Error())
		return
	}

	state := oauthConfigModelFromForm(*current)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *oauthConfigResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing OAuth config.")
		return
	}
	// Singleton config: destroying the resource leaves the remote config untouched.
}

func (r *oauthConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func oauthConfigFormFromModel(m oauthConfigModel) client.OAuthConfigForm {
	return client.OAuthConfigForm{
		EnableOAuth:                   boolPtr(m.EnableOAuth),
		EnableOAuthSignup:             boolPtr(m.EnableOAuthSignup),
		OAuthMergeAccountsByEmail:     boolPtr(m.OAuthMergeAccountsByEmail),
		OAuthAutoRedirect:             boolPtr(m.OAuthAutoRedirect),
		OAuthAllowedDomains:           stringPtr(m.OAuthAllowedDomains),
		OAuthBlockedGroups:            stringPtr(m.OAuthBlockedGroups),
		EnableOAuthRoleManagement:     boolPtr(m.EnableOAuthRoleManagement),
		OAuthRolesClaim:               stringPtr(m.OAuthRolesClaim),
		OAuthAdminRoles:               stringPtr(m.OAuthAdminRoles),
		OAuthAllowedRoles:             stringPtr(m.OAuthAllowedRoles),
		EnableOAuthGroupManagement:    boolPtr(m.EnableOAuthGroupManagement),
		EnableOAuthGroupCreation:      boolPtr(m.EnableOAuthGroupCreation),
		OAuthGroupClaim:               stringPtr(m.OAuthGroupClaim),
		OAuthGroupDefaultShare:        stringPtr(m.OAuthGroupDefaultShare),
		OAuthProviderName:             stringPtr(m.OAuthProviderName),
		OpenIDProviderURL:             stringPtr(m.OpenIDProviderURL),
		OAuthClientID:                 stringPtr(m.OAuthClientID),
		OAuthClientSecret:             stringPtr(m.OAuthClientSecret),
		OpenIDRedirectURI:             stringPtr(m.OpenIDRedirectURI),
		OAuthScopes:                   stringPtr(m.OAuthScopes),
		OAuthCodeChallengeMethod:      stringPtr(m.OAuthCodeChallengeMethod),
		OAuthTokenEndpointAuthMethod:  stringPtr(m.OAuthTokenEndpointAuthMethod),
		OpenIDEndSessionEndpoint:      stringPtr(m.OpenIDEndSessionEndpoint),
		OAuthTimeout:                  stringPtr(m.OAuthTimeout),
		OAuthClientTimeout:            stringPtr(m.OAuthClientTimeout),
		OAuthEmailClaim:               stringPtr(m.OAuthEmailClaim),
		OAuthUsernameClaim:            stringPtr(m.OAuthUsernameClaim),
		OAuthPictureClaim:             stringPtr(m.OAuthPictureClaim),
		OAuthSubClaim:                 stringPtr(m.OAuthSubClaim),
		OAuthAudience:                 stringPtr(m.OAuthAudience),
		OAuthUpdateEmailOnLogin:       boolPtr(m.OAuthUpdateEmailOnLogin),
		OAuthUpdateNameOnLogin:        boolPtr(m.OAuthUpdateNameOnLogin),
		OAuthUpdatePictureOnLogin:     boolPtr(m.OAuthUpdatePictureOnLogin),
		OAuthRefreshTokenIncludeScope: boolPtr(m.OAuthRefreshTokenIncludeScope),
	}
}

func oauthConfigModelFromForm(f client.OAuthConfigForm) oauthConfigModel {
	return oauthConfigModel{
		ID:                            types.StringValue("oauth_config"),
		EnableOAuth:                   boolValueOrNull(f.EnableOAuth),
		EnableOAuthSignup:             boolValueOrNull(f.EnableOAuthSignup),
		OAuthMergeAccountsByEmail:     boolValueOrNull(f.OAuthMergeAccountsByEmail),
		OAuthAutoRedirect:             boolValueOrNull(f.OAuthAutoRedirect),
		OAuthAllowedDomains:           stringValueOrNull(f.OAuthAllowedDomains),
		OAuthBlockedGroups:            stringValueOrNull(f.OAuthBlockedGroups),
		EnableOAuthRoleManagement:     boolValueOrNull(f.EnableOAuthRoleManagement),
		OAuthRolesClaim:               stringValueOrNull(f.OAuthRolesClaim),
		OAuthAdminRoles:               stringValueOrNull(f.OAuthAdminRoles),
		OAuthAllowedRoles:             stringValueOrNull(f.OAuthAllowedRoles),
		EnableOAuthGroupManagement:    boolValueOrNull(f.EnableOAuthGroupManagement),
		EnableOAuthGroupCreation:      boolValueOrNull(f.EnableOAuthGroupCreation),
		OAuthGroupClaim:               stringValueOrNull(f.OAuthGroupClaim),
		OAuthGroupDefaultShare:        stringValueOrNull(f.OAuthGroupDefaultShare),
		OAuthProviderName:             stringValueOrNull(f.OAuthProviderName),
		OpenIDProviderURL:             stringValueOrNull(f.OpenIDProviderURL),
		OAuthClientID:                 stringValueOrNull(f.OAuthClientID),
		OAuthClientSecret:             stringValueOrNull(f.OAuthClientSecret),
		OpenIDRedirectURI:             stringValueOrNull(f.OpenIDRedirectURI),
		OAuthScopes:                   stringValueOrNull(f.OAuthScopes),
		OAuthCodeChallengeMethod:      stringValueOrNull(f.OAuthCodeChallengeMethod),
		OAuthTokenEndpointAuthMethod:  stringValueOrNull(f.OAuthTokenEndpointAuthMethod),
		OpenIDEndSessionEndpoint:      stringValueOrNull(f.OpenIDEndSessionEndpoint),
		OAuthTimeout:                  stringValueOrNull(f.OAuthTimeout),
		OAuthClientTimeout:            stringValueOrNull(f.OAuthClientTimeout),
		OAuthEmailClaim:               stringValueOrNull(f.OAuthEmailClaim),
		OAuthUsernameClaim:            stringValueOrNull(f.OAuthUsernameClaim),
		OAuthPictureClaim:             stringValueOrNull(f.OAuthPictureClaim),
		OAuthSubClaim:                 stringValueOrNull(f.OAuthSubClaim),
		OAuthAudience:                 stringValueOrNull(f.OAuthAudience),
		OAuthUpdateEmailOnLogin:       boolValueOrNull(f.OAuthUpdateEmailOnLogin),
		OAuthUpdateNameOnLogin:        boolValueOrNull(f.OAuthUpdateNameOnLogin),
		OAuthUpdatePictureOnLogin:     boolValueOrNull(f.OAuthUpdatePictureOnLogin),
		OAuthRefreshTokenIncludeScope: boolValueOrNull(f.OAuthRefreshTokenIncludeScope),
	}
}
