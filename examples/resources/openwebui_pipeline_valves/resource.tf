resource "openwebui_pipeline_valves" "example" {
  pipeline_id = openwebui_pipeline.example.pipeline_id
  url_idx     = openwebui_pipeline.example.url_idx

  valves_json = jsonencode({
    temperature = 0.7
  })
}
