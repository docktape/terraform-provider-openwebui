resource "openwebui_model" "example" {
  model_id      = "custom-rag"
  name          = "Custom RAG Model"
  base_model_id = "llama3.2"
  is_active     = true
  description   = "RAG-tuned model for the internal knowledge base"

  read_groups  = ["Support"]
  write_groups = ["Support"]

  params {
    temperature = 0.1
    num_ctx     = 4096
  }

  capabilities {
    vision     = false
    web_search = true
  }
}
