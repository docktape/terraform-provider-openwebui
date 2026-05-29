resource "openwebui_knowledge" "example" {
  name        = "Support FAQ"
  description = "Knowledge base backing the support chatbot"

  read_groups  = ["Support"]
  write_groups = ["Support"]
}
