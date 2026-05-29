resource "openwebui_function_valves" "example" {
  function_id = openwebui_function.example.id

  valves_json = jsonencode({
    priority = 5
  })
}
