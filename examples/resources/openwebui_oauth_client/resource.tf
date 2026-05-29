resource "openwebui_oauth_client" "example" {
  url         = "https://auth.example.com"
  client_id   = "my-terraform-client"
  client_name = "Terraform Managed Client"
  type        = "confidential"
}
