resource "openwebui_banners_config" "example" {
  banners = [
    {
      id          = "maintenance-2024"
      type        = "warning"
      title       = "Scheduled Maintenance"
      content     = "The system will be unavailable Saturday 2 AM – 4 AM UTC."
      dismissible = true
      timestamp   = 1700000000
    },
  ]
}
