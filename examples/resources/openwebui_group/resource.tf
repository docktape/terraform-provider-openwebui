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
