data "openwebui_config_export" "current" {}

output "config_backup" {
  value     = data.openwebui_config_export.current.config_json
  sensitive = true
}
