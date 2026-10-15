variable "terminal_key" {
  type      = string
  sensitive = true
}

# Open WebUI replaces the whole list on every apply.
resource "openwebui_terminal_servers_config" "example" {
  connections = [
    {
      id        = "dev-terminal"
      name      = "Dev terminal"
      url       = "http://open-terminal.internal:8000"
      auth_type = "bearer"
      key       = var.terminal_key

      # Let all signed-in users use this terminal (admin-only without grants).
      config_json = jsonencode({
        access_grants = [
          { principal_type = "user", principal_id = "*", permission = "read" },
        ]
      })
    },
    {
      id          = "orchestrator"
      name        = "Orchestrator"
      url         = "http://orchestrator.internal:8000"
      auth_type   = "none"
      server_type = "orchestrator"
      policy_id   = "default"
      enabled     = false
    },
  ]
}
