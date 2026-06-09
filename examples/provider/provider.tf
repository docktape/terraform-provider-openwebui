terraform {
  required_providers {
    openwebui = {
      source  = "docktape/openwebui"
      version = "~> 1.0"
    }
  }
}

provider "openwebui" {
  endpoint = "https://openwebui.example.com"
  token    = var.openwebui_token
}

variable "openwebui_token" {
  type        = string
  sensitive   = true
  description = "API token for the Open WebUI instance."
}
