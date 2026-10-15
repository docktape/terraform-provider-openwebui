variable "oidc_client_secret" {
  type      = string
  sensitive = true
}

# Requires Open WebUI v0.10.0 or later.
resource "openwebui_oauth_config" "example" {
  oauth_provider_name = "Corporate SSO"
  openid_provider_url = "https://idp.example.com/.well-known/openid-configuration"
  oauth_client_id     = "open-webui"
  oauth_client_secret = var.oidc_client_secret
  oauth_scopes        = "openid email profile"

  enable_oauth_signup   = true
  oauth_allowed_domains = "example.com,example.org"

  enable_oauth_role_management = true
  oauth_roles_claim            = "roles"
  oauth_admin_roles            = "openwebui-admin"
  oauth_allowed_roles          = "openwebui-user,openwebui-admin"

  enable_oauth_group_management = true
  enable_oauth_group_creation   = false
  oauth_group_claim             = "groups"
  oauth_group_default_share     = "members"
}
