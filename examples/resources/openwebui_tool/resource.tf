resource "openwebui_tool" "example" {
  tool_id     = "web_scraper"
  name        = "Web Scraper"
  description = "Fetches and parses web page content"
  content     = file("${path.module}/tools/web_scraper.py")

  read_groups  = ["Support"]
  write_groups = ["Support"]
}
