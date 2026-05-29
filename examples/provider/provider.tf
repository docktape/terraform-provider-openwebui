terraform {
  required_providers {
    openwebui = {
      source  = "docktape/openwebui"
      version = "~> 0.1"
    }
  }
}

provider "openwebui" {
  endpoint = "https://openwebui.example.com/api/v1"
  token    = var.openwebui_token
}

variable "openwebui_token" {
  type        = string
  sensitive   = true
  description = "API token for the Open WebUI instance."
}
