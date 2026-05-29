resource "openwebui_tool_servers_config" "example" {
  connections = [
    {
      url       = "http://tools.internal:8080"
      path      = "/openapi.json"
      auth_type = "bearer"
      key       = var.tool_server_api_key
    },
  ]
}

variable "tool_server_api_key" {
  type      = string
  sensitive = true
}
