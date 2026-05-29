resource "openwebui_tool_valves" "example" {
  tool_id = openwebui_tool.example.id

  valves_json = jsonencode({
    max_pages = 5
    timeout   = 30
  })
}
