resource "openwebui_file" "example" {
  source_path = "${path.module}/files/document.pdf"
  process     = true
}
