terraform {
  required_providers {
    openwebui = {
      source = "docktape/openwebui"
    }
  }
}

provider "openwebui" {
  endpoint = var.openwebui_endpoint
  token    = var.openwebui_token
}

resource "openwebui_function" "example_filter" {
  function_id = "example_filter"
  name        = "Example Filter"
  description = "Demonstration filter managed by Terraform"
  is_active   = true

  content = <<-PY
    """
    title: Example Filter
    author: docktape
    version: 0.1.0
    """

    from pydantic import BaseModel


    class Filter:
        class Valves(BaseModel):
            priority: int = 0

        def __init__(self):
            self.valves = self.Valves()

        def inlet(self, body: dict) -> dict:
            return body
  PY
}

resource "openwebui_function_valves" "example_filter" {
  function_id = openwebui_function.example_filter.id

  valves_json = jsonencode({
    priority = 5
  })
}

variable "openwebui_endpoint" {
  type        = string
  description = "Base URL for the Open WebUI API"
  default     = "http://localhost:3000/api/v1"
}

variable "openwebui_token" {
  type        = string
  description = "API token for Open WebUI"
  sensitive   = true
}
