# Terraform Provider for Open WebUI

A Terraform provider that manages [Open WebUI](https://openwebui.com) resources
through its REST API. Declaratively manage knowledge bases, models, prompts,
groups, tools, pipelines, functions, files, model connections, and admin-level
configuration in any Open WebUI deployment.

- **Provider address:** `registry.terraform.io/docktape/openwebui`
- **Module path:** `github.com/docktape/terraform-provider-openwebui`

## Compatibility

| Component      | Requirement                                             |
| -------------- | ------------------------------------------------------- |
| Terraform      | 1.6 or newer                                            |
| Open WebUI     | v0.9.0 – v0.9.5                                         |
| Go             | 1.25+ (only required to build the provider)             |

Attributes backed by API endpoints that appeared in a later patch release are
marked optional and noted in their schema descriptions so older deployments keep
working.

## Provider Configuration

```hcl
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
  description = "Admin API token for the Open WebUI instance."
}
```

| Argument               | Env var              | Required | Description                                                           |
| ---------------------- | -------------------- | -------- | --------------------------------------------------------------------- |
| `endpoint`             | `OPENWEBUI_ENDPOINT` | Yes      | Base URL of the instance, e.g. `https://openwebui.example.com`.       |
| `token`                | `OPENWEBUI_TOKEN`    | Yes      | Bearer token used to authenticate API requests (sensitive).           |
| `insecure_skip_verify` | `OPENWEBUI_INSECURE` | No       | Disable TLS certificate verification. Not recommended for production. |

## Resources

| Resource                          | Description                                     |
| --------------------------------- | ----------------------------------------------- |
| `openwebui_knowledge`             | Knowledge base entries                          |
| `openwebui_knowledge_file`        | File attachments on a knowledge base            |
| `openwebui_model`                 | Custom model definitions                        |
| `openwebui_prompt`                | Reusable prompt commands                        |
| `openwebui_group`                 | User groups with permissions                    |
| `openwebui_tool`                  | Python tools                                    |
| `openwebui_tool_valves`           | Settings (valves) for a tool                    |
| `openwebui_pipeline`              | Pipeline registrations                          |
| `openwebui_pipeline_valves`       | Settings (valves) for a pipeline                |
| `openwebui_function`              | Python functions                                |
| `openwebui_function_valves`       | Settings (valves) for a function                |
| `openwebui_file`                  | Uploaded files                                  |
| `openwebui_openai_connections`    | OpenAI-compatible model connections             |
| `openwebui_ollama_connections`    | Ollama model connections                        |
| `openwebui_connections_config`    | Direct connection settings                      |
| `openwebui_tool_servers_config`   | External tool server registrations              |
| `openwebui_code_execution_config` | Code execution and interpreter settings         |
| `openwebui_models_config`         | Default model ordering settings                 |
| `openwebui_suggestions_config`    | Default prompt suggestions                      |
| `openwebui_banners_config`        | UI announcement banners                         |
| `openwebui_oauth_client`          | OAuth client registrations                      |
| `openwebui_config_import`         | Bulk configuration restore                      |

## Data Sources

| Data Source                          | Description                                        |
| ------------------------------------ | -------------------------------------------------- |
| `openwebui_model`                    | Look up a model by name or ID                      |
| `openwebui_knowledge`                | Look up a knowledge base by name or ID             |
| `openwebui_prompt`                   | Look up a prompt by command or ID                  |
| `openwebui_group`                    | Look up a group by name or ID                      |
| `openwebui_tool`                     | Look up a tool by name or ID                       |
| `openwebui_pipeline`                 | Look up a pipeline by name or ID                   |
| `openwebui_file`                     | Look up a single uploaded file                     |
| `openwebui_files`                    | List uploaded files                                |
| `openwebui_user`                     | Look up a user by email or ID                      |
| `openwebui_config_export`            | Export the current Open WebUI configuration        |
| `openwebui_tool_server_verify`       | Verify connectivity to an external tool server     |
| `openwebui_openai_connection_verify` | Verify an OpenAI-compatible connection             |
| `openwebui_ollama_connection_verify` | Verify an Ollama connection                        |

Full per-resource and per-data-source documentation is in [`docs/`](docs), with
runnable configurations under [`examples/`](examples).

## Usage Examples

### Knowledge base

```hcl
resource "openwebui_knowledge" "example" {
  name        = "Support FAQ"
  description = "Knowledge base backing the support chatbot"

  read_groups  = ["Support"]
  write_groups = ["Support"]
}
```

### Model

```hcl
resource "openwebui_model" "example" {
  model_id      = "custom-rag"
  name          = "Custom RAG Model"
  base_model_id = "llama3.2"
  is_active     = true
  description   = "RAG-tuned model for the internal knowledge base"

  read_groups  = ["Support"]
  write_groups = ["Support"]

  params {
    temperature = 0.1
    num_ctx     = 4096
  }

  capabilities {
    vision     = false
    web_search = true
  }
}
```

### Group

```hcl
resource "openwebui_group" "example" {
  name        = "Support"
  description = "Support team access group"

  users = [
    "alice@example.com",
    "bob@example.com",
  ]

  permissions = {
    workspace = {
      models    = true
      knowledge = true
      prompts   = true
      tools     = false
    }
    chat = {
      file_upload = true
      delete      = true
      edit        = true
    }
    features = {
      web_search = true
    }
  }
}
```

