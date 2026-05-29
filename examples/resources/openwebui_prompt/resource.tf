resource "openwebui_prompt" "example" {
  command = "summarize"
  name    = "Summarize Text"
  content = "Summarize the following in 3 concise bullet points:\n\n{{text}}"

  read_groups  = ["Support"]
  write_groups = ["Support"]
}
