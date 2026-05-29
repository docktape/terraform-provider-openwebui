resource "openwebui_models_config" "example" {
  default_models        = "llama3.2"
  default_pinned_models = "llama3.2,gpt-4o"
  model_order_list      = ["llama3.2", "gpt-4o", "custom-rag"]
}
