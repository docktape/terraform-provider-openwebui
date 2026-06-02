package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccOpenAIConnectionsResource tests the openwebui_openai_connections singleton resource.
// Requires OPENWEBUI_TEST_OPENAI_URL and OPENWEBUI_TEST_OPENAI_KEY env vars.
// The test mutates the target Open WebUI instance's OpenAI connections.
//
// If OPENWEBUI_PIPELINE_SERVER_URL / OPENWEBUI_PIPELINE_SERVER_KEY are set the cleanup
// step restores that connection so pipeline acceptance tests that run afterwards still work.
func TestAccOpenAIConnectionsResource(t *testing.T) {
	url := testAccRequireEnv(t, "OPENWEBUI_TEST_OPENAI_URL")
	key := testAccRequireEnv(t, "OPENWEBUI_TEST_OPENAI_KEY")

	t.Setenv("TF_VAR_openai_url", url)
	t.Setenv("TF_VAR_openai_key", key)

	// Optional: baseline connection to restore in cleanup (e.g. the pipeline server).
	restoreURL := testAccOptionalEnv(t, "OPENWEBUI_PIPELINE_SERVER_URL")
	restoreKey := testAccOptionalEnv(t, "OPENWEBUI_PIPELINE_SERVER_KEY")

	var cleanupConfig string
	if restoreURL != "" {
		cleanupConfig = testAccProviderConfig() + `
resource "openwebui_openai_connections" "test" {
  enabled = true
  connections = [
    {
      url = "` + restoreURL + `"
      key = "` + restoreKey + `"
    }
  ]
}`
	} else {
		cleanupConfig = testAccProviderConfig() + `
resource "openwebui_openai_connections" "test" {
  enabled     = true
  connections = []
}`
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			// Step 1: Create with one connection.
			{
				Config: testAccProviderConfig() + `
resource "openwebui_openai_connections" "test" {
  enabled = true
  connections = [
    {
      url  = "` + url + `"
      key  = "` + key + `"
      tags = ["acc-test"]
    }
  ]
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_openai_connections.test", "id", "openai"),
					resource.TestCheckResourceAttr("openwebui_openai_connections.test", "enabled", "true"),
					resource.TestCheckResourceAttr("openwebui_openai_connections.test", "connections.#", "1"),
					resource.TestCheckResourceAttr("openwebui_openai_connections.test", "connections.0.url", url),
					resource.TestCheckResourceAttr("openwebui_openai_connections.test", "connections.0.tags.0", "acc-test"),
					resource.TestCheckResourceAttr("openwebui_openai_connections.test", "connections.0.enable", "true"),
				),
			},
			// Step 2: Idempotency — same config, no diff.
			{
				Config: testAccProviderConfig() + `
resource "openwebui_openai_connections" "test" {
  enabled = true
  connections = [
    {
      url  = "` + url + `"
      key  = "` + key + `"
      tags = ["acc-test"]
    }
  ]
}`,
				PlanOnly: true,
			},
			// Step 3: Update — disable the connection.
			{
				Config: testAccProviderConfig() + `
resource "openwebui_openai_connections" "test" {
  enabled = true
  connections = [
    {
      url    = "` + url + `"
      key    = "` + key + `"
      enable = false
      tags   = ["acc-test"]
    }
  ]
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_openai_connections.test", "connections.0.enable", "false"),
				),
			},
			// Step 4: Import by singleton ID.
			{
				ResourceName:            "openwebui_openai_connections.test",
				ImportState:             true,
				ImportStateId:           "openai",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"connections.0.key"}, // key is write-only
			},
			// Step 5: Restore baseline connections (or clear if no baseline is configured).
			{
				Config: cleanupConfig,
			},
		},
	})
}
