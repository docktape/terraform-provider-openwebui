# Only the attributes you set are sent; every other retrieval setting keeps
# its current Open WebUI value.
resource "openwebui_retrieval_config" "example" {
  top_k                    = 5
  enable_rag_hybrid_search = false

  content_extraction_engine = "tika"
  tika_server_url           = "http://tika:9998"

  text_splitter = "token"
  chunk_size    = 1000
  chunk_overlap = 100

  # -1 means "no limit".
  file_max_size           = 50
  file_max_count          = -1
  allowed_file_extensions = ["pdf", "docx", "md", "txt"]

  docling_params_json = jsonencode({ do_ocr = true })
}
