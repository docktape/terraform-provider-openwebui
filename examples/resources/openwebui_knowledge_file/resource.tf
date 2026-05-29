resource "openwebui_file" "doc" {
  source_path = "${path.module}/files/faq.pdf"
  process     = true
}

resource "openwebui_knowledge" "faq" {
  name        = "Support FAQ"
  description = "Knowledge base backing the support chatbot"
}

resource "openwebui_knowledge_file" "example" {
  knowledge_id = openwebui_knowledge.faq.id
  file_id      = openwebui_file.doc.id
  delete_file  = true
}
