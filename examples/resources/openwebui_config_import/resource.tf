data "openwebui_config_export" "current" {}

resource "openwebui_config_import" "restore" {
  config_json = data.openwebui_config_export.current.config_json
}
