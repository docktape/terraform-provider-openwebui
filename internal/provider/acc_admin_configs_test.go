package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccConnectionsConfigResource applies a known connections config and
// verifies idempotency. Mutates the target Open WebUI instance.
func TestAccConnectionsConfigResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "openwebui_connections_config" "test" {
  enable_direct_connections = false
  enable_base_models_cache  = false
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_connections_config.test", "enable_direct_connections", "false"),
					resource.TestCheckResourceAttr("openwebui_connections_config.test", "enable_base_models_cache", "false"),
					resource.TestCheckResourceAttrSet("openwebui_connections_config.test", "id"),
				),
			},
			{
				// Idempotency: re-plan with same config, expect no diff
				Config: testAccProviderConfig() + `
resource "openwebui_connections_config" "test" {
  enable_direct_connections = false
  enable_base_models_cache  = false
}`,
				PlanOnly: true,
			},
		},
	})
}

// TestAccModelsConfigResource applies a known models config and verifies idempotency.
func TestAccModelsConfigResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "openwebui_models_config" "test" {
  model_order_list = []
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("openwebui_models_config.test", "id"),
				),
			},
			{
				Config: testAccProviderConfig() + `
resource "openwebui_models_config" "test" {
  model_order_list = []
}`,
				PlanOnly: true,
			},
		},
	})
}

// TestAccSuggestionsConfigResource applies suggestions and verifies idempotency.
func TestAccSuggestionsConfigResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "openwebui_suggestions_config" "test" {
  suggestions = [
    {
      title   = ["Hello"]
      content = "Say hello."
    },
  ]
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("openwebui_suggestions_config.test", "id"),
					resource.TestCheckResourceAttr("openwebui_suggestions_config.test", "suggestions.0.content", "Say hello."),
				),
			},
			{
				Config: testAccProviderConfig() + `
resource "openwebui_suggestions_config" "test" {
  suggestions = [
    {
      title   = ["Hello"]
      content = "Say hello."
    },
  ]
}`,
				PlanOnly: true,
			},
		},
	})
}

// TestAccBannersConfigResource applies a banner and verifies idempotency.
func TestAccBannersConfigResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "openwebui_banners_config" "test" {
  banners = []
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("openwebui_banners_config.test", "id"),
				),
			},
			{
				Config: testAccProviderConfig() + `
resource "openwebui_banners_config" "test" {
  banners = []
}`,
				PlanOnly: true,
			},
		},
	})
}

// TestAccCodeExecutionConfigResource applies code execution config and verifies idempotency.
func TestAccCodeExecutionConfigResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "openwebui_code_execution_config" "test" {
  enable_code_execution   = false
  code_execution_engine   = "jupyter"
  enable_code_interpreter = false
  code_interpreter_engine = "jupyter"
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_code_execution_config.test", "enable_code_execution", "false"),
					resource.TestCheckResourceAttrSet("openwebui_code_execution_config.test", "id"),
				),
			},
			{
				Config: testAccProviderConfig() + `
resource "openwebui_code_execution_config" "test" {
  enable_code_execution   = false
  code_execution_engine   = "jupyter"
  enable_code_interpreter = false
  code_interpreter_engine = "jupyter"
}`,
				PlanOnly: true,
			},
		},
	})
}

// TestAccToolServersConfigResource applies an empty tool servers config and verifies idempotency.
func TestAccToolServersConfigResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "openwebui_tool_servers_config" "test" {
  connections = []
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("openwebui_tool_servers_config.test", "id"),
				),
			},
			{
				Config: testAccProviderConfig() + `
resource "openwebui_tool_servers_config" "test" {
  connections = []
}`,
				PlanOnly: true,
			},
		},
	})
}
