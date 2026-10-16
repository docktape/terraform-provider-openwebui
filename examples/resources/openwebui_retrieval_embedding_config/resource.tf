# Only the attributes you set are changed; everything else, including
# connection keys you do not set here, keeps its current Open WebUI value.
# Every update makes Open WebUI reload the embedding model.
resource "openwebui_retrieval_embedding_config" "example" {
  rag_embedding_engine = "openai"
  rag_embedding_model  = "text-embedding-3-small"

  rag_embedding_batch_size          = 16
  enable_async_embedding            = true
  rag_embedding_concurrent_requests = 4 # 0 = unlimited

  # Stored by Open WebUI only while a remote engine (ollama, openai or
  # azure_openai) is selected.
  openai_config = {
    url = "https://api.openai.com/v1"
    key = var.openai_embedding_api_key
  }
}

variable "openai_embedding_api_key" {
  type      = string
  sensitive = true
}
