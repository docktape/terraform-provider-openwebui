data "openwebui_terminal_server_verify" "dev" {
  url       = "http://open-terminal.internal:8000"
  auth_type = "none"
}

output "terminal_server_type" {
  value = data.openwebui_terminal_server_verify.dev.server_type
}
