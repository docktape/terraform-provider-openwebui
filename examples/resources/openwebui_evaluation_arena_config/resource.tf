resource "openwebui_evaluation_arena_config" "example" {
  enable_evaluation_arena_models = true
  evaluation_arena_models        = ["gpt-4o", "claude-3-opus", "llama3.2"]
}
