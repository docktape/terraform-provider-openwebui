resource "openwebui_code_execution_config" "example" {
  enable_code_execution      = true
  code_execution_engine      = "jupyter"
  code_execution_jupyter_url = "http://jupyter.internal:8888"
  code_execution_jupyter_auth = "token"
  code_execution_jupyter_auth_token = var.jupyter_token

  enable_code_interpreter    = false
}

variable "jupyter_token" {
  type      = string
  sensitive = true
}
