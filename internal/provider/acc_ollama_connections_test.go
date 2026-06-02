package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccOllamaConnectionsResource tests the openwebui_ollama_connections singleton resource.
// Uses http://localhost:11434 (default Ollama). Skipped if OPENWEBUI_TEST_OLLAMA_URL is unset
// and localhost:11434 is not available.
func TestAccOllamaConnectionsResource(t *testing.T) {
	ollamaURL := "http://localhost:11434"
	if envURL := testAccOptionalEnv(t, "OPENWEBUI_TEST_OLLAMA_URL"); envURL != "" {
		ollamaURL = envURL
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			// Step 1: Create with one connection.
			{
				Config: testAccProviderConfig() + `
resource "openwebui_ollama_connections" "test" {
  enabled = true
  connections = [
    {
      url             = "` + ollamaURL + `"
      connection_type = "local"
      tags            = ["acc-test"]
    }
  ]
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_ollama_connections.test", "id", "ollama"),
					resource.TestCheckResourceAttr("openwebui_ollama_connections.test", "enabled", "true"),
					resource.TestCheckResourceAttr("openwebui_ollama_connections.test", "connections.#", "1"),
					resource.TestCheckResourceAttr("openwebui_ollama_connections.test", "connections.0.url", ollamaURL),
					resource.TestCheckResourceAttr("openwebui_ollama_connections.test", "connections.0.enable", "true"),
				),
			},
			// Step 2: Idempotency.
			{
				Config: testAccProviderConfig() + `
resource "openwebui_ollama_connections" "test" {
  enabled = true
  connections = [
    {
      url             = "` + ollamaURL + `"
      connection_type = "local"
      tags            = ["acc-test"]
    }
  ]
}`,
				PlanOnly: true,
			},
			// Step 3: Import by singleton ID.
			{
				ResourceName:      "openwebui_ollama_connections.test",
				ImportState:       true,
				ImportStateId:     "ollama",
				ImportStateVerify: true,
			},
			// Step 4: Clear connections.
			{
				Config: testAccProviderConfig() + `
resource "openwebui_ollama_connections" "test" {
  enabled     = true
  connections = []
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_ollama_connections.test", "connections.#", "0"),
				),
			},
		},
	})
}
