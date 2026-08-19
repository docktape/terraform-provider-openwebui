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

// TestAccTasksConfigResource applies a known tasks config and verifies idempotency.
func TestAccTasksConfigResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "openwebui_tasks_config" "test" {
  enable_title_generation        = true
  title_generation_prompt_template = "Summarize this conversation: {{prompt}}"
  enable_tags_generation         = true
  tags_generation_prompt_template = "Generate tags: {{prompt}}"
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_tasks_config.test", "enable_title_generation", "true"),
					resource.TestCheckResourceAttrSet("openwebui_tasks_config.test", "id"),
				),
			},
			{
				Config: testAccProviderConfig() + `
resource "openwebui_tasks_config" "test" {
  enable_title_generation        = true
  title_generation_prompt_template = "Summarize this conversation: {{prompt}}"
  enable_tags_generation         = true
  tags_generation_prompt_template = "Generate tags: {{prompt}}"
}`,
				PlanOnly: true,
			},
		},
	})
}

// TestAccChatConfigResource applies a known chat config and verifies idempotency.
func TestAccChatConfigResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "openwebui_chat_config" "test" {
  enable_context_compaction                 = true
  context_compaction_token_threshold        = 8000
  context_compaction_token_cap              = 12000
  context_compaction_retention_percentage   = 40
  context_compaction_prompt_template        = "Compact this context: {{context}}"
  context_compaction_model                  = "gpt-4o-mini"
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_chat_config.test", "enable_context_compaction", "true"),
					resource.TestCheckResourceAttrSet("openwebui_chat_config.test", "id"),
				),
			},
			{
				Config: testAccProviderConfig() + `
resource "openwebui_chat_config" "test" {
  enable_context_compaction                 = true
  context_compaction_token_threshold        = 8000
  context_compaction_token_cap              = 12000
  context_compaction_retention_percentage   = 40
  context_compaction_prompt_template        = "Compact this context: {{context}}"
  context_compaction_model                  = "gpt-4o-mini"
}`,
				PlanOnly: true,
			},
		},
	})
}

// TestAccEvaluationArenaConfigResource applies a known evaluation arena config and verifies idempotency.
func TestAccEvaluationArenaConfigResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				// The second entry deliberately omits model_ids to exercise
				// the real Open WebUI backend default (meta.model_ids: None).
				// model_ids is Optional (not Computed), so the provider must
				// return exactly null for it, not an empty list - see
				// TestFlattenArenaModelsPreservesNullModelIDs and the fix in
				// flattenArenaModels. The third entry explicitly sets model_ids = []
				// to cover the case from the most recent fix (d2baaff).
				Config: testAccProviderConfig() + `
resource "openwebui_evaluation_arena_config" "test" {
  enable_evaluation_arena_models = true
  evaluation_arena_models = [
    {
      id        = "arena-test"
      name      = "Arena Test"
      model_ids = ["gpt-4o", "claude-3-opus"]
    },
    {
      id   = "arena-no-model-ids"
      name = "Arena No Model IDs"
    },
    {
      id        = "arena-empty-model-ids"
      name      = "Arena Empty Model IDs"
      model_ids = []
    }
  ]
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_evaluation_arena_config.test", "enable_evaluation_arena_models", "true"),
					resource.TestCheckResourceAttrSet("openwebui_evaluation_arena_config.test", "id"),
					resource.TestCheckResourceAttr("openwebui_evaluation_arena_config.test", "evaluation_arena_models.0.id", "arena-test"),
					resource.TestCheckResourceAttr("openwebui_evaluation_arena_config.test", "evaluation_arena_models.0.name", "Arena Test"),
					resource.TestCheckResourceAttr("openwebui_evaluation_arena_config.test", "evaluation_arena_models.1.id", "arena-no-model-ids"),
					resource.TestCheckNoResourceAttr("openwebui_evaluation_arena_config.test", "evaluation_arena_models.1.model_ids"),
					resource.TestCheckResourceAttr("openwebui_evaluation_arena_config.test", "evaluation_arena_models.2.id", "arena-empty-model-ids"),
					resource.TestCheckResourceAttr("openwebui_evaluation_arena_config.test", "evaluation_arena_models.2.model_ids.#", "0"),
				),
			},
			{
				Config: testAccProviderConfig() + `
resource "openwebui_evaluation_arena_config" "test" {
  enable_evaluation_arena_models = true
  evaluation_arena_models = [
    {
      id        = "arena-test"
      name      = "Arena Test"
      model_ids = ["gpt-4o", "claude-3-opus"]
    },
    {
      id   = "arena-no-model-ids"
      name = "Arena No Model IDs"
    },
    {
      id        = "arena-empty-model-ids"
      name      = "Arena Empty Model IDs"
      model_ids = []
    }
  ]
}`,
				PlanOnly: true,
			},
		},
	})
}
