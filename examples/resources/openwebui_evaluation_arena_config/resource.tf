resource "openwebui_evaluation_arena_config" "example" {
  enable_evaluation_arena_models = true

  evaluation_arena_models = [
    {
      id                = "arena-model"
      name              = "Arena Model"
      profile_image_url = "/favicon.png"
      description       = "Compares gpt-4o against claude-3-opus"
      model_ids         = ["gpt-4o", "claude-3-opus"]
    },
    {
      id        = "arena-model-2"
      name      = "Local Models Arena"
      model_ids = ["llama3.2"]
    }
  ]
}
