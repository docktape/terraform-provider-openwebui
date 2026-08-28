resource "openwebui_subagents_config" "example" {
  enable_subagents             = true
  subagents_background_enabled = true
  subagents_max_concurrent     = 5
  subagents_max_async          = 3
  subagents_max_iterations     = 10
  subagents_max_output         = 4096
  subagents_system_prompt      = "You are a helpful assistant designed to break down complex tasks into smaller sub-tasks."
}
