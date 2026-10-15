variable "ldap_bind_password" {
  type      = string
  sensitive = true
}

resource "openwebui_ldap_server_config" "example" {
  label                  = "Corporate LDAP"
  host                   = "ldap.example.com"
  port                   = 636
  attribute_for_mail     = "mail"
  attribute_for_username = "uid"
  app_dn                 = "cn=svc-openwebui,ou=services,dc=example,dc=com"
  app_dn_password        = var.ldap_bind_password
  search_base            = "ou=people,dc=example,dc=com"
  search_filters         = "(objectClass=person)"
  use_tls                = true
  validate_cert          = true

  # Group synchronisation requires Open WebUI v0.11.0 or later.
  enable_group_management = true
  enable_group_creation   = false
  attribute_for_groups    = "memberOf"
}
