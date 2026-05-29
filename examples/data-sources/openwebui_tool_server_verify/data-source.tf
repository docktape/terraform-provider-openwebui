data "openwebui_tool_server_verify" "local_tools" {
  url  = "http://tools.internal:8080"
  path = "/openapi.json"
}
