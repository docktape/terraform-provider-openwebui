resource "openwebui_function" "example" {
  function_id = "content_filter"
  name        = "Content Filter"
  description = "Filters inappropriate content from model responses"
  is_active   = true
  is_global   = false

  content = file("${path.module}/functions/content_filter.py")
}
