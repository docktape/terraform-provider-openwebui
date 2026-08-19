resource "openwebui_tasks_config" "example" {
  enable_title_generation        = true
  title_generation_prompt_template = "Generate a concise 3-5 word title for this conversation."
  enable_tags_generation         = true
  enable_autocomplete_generation = true
}
