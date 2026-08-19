resource "openwebui_chat_config" "example" {
  enable_context_compaction                 = true
  context_compaction_token_threshold        = 8000
  context_compaction_token_cap              = 12000
  context_compaction_retention_percentage   = 50.5
  context_compaction_prompt_template        = "Please compact the following context to the most important parts."
}
