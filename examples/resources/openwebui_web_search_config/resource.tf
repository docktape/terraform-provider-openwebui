# Only the attributes you set are changed; every other web search setting,
# including provider API keys you do not set here, keeps its current
# Open WebUI value.
resource "openwebui_web_search_config" "example" {
  enable_web_search       = true
  web_search_engine       = "searxng"
  searxng_query_url       = "http://searxng:8080/search?q=<query>"
  web_search_result_count = 3

  # Entries starting with "!" block a domain; [] means no filter.
  web_search_domain_filter_list = ["wikipedia.org", "!example.com"]

  web_loader_engine                  = "safe_web"
  enable_web_loader_ssl_verification = true
  youtube_loader_language            = ["en"]
}
