package provider

import (
	"context"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
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

// TestAccSubagentsConfigResource applies a known subagents config and verifies idempotency.
func TestAccSubagentsConfigResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "openwebui_subagents_config" "test" {
  enable_subagents              = true
  subagents_background_enabled  = false
  subagents_max_concurrent      = 5
  subagents_max_async           = 3
  subagents_max_iterations      = 10
  subagents_max_output          = 2048
  subagents_system_prompt       = "You are a helpful agent."
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_subagents_config.test", "enable_subagents", "true"),
					resource.TestCheckResourceAttr("openwebui_subagents_config.test", "subagents_background_enabled", "false"),
					resource.TestCheckResourceAttr("openwebui_subagents_config.test", "subagents_max_concurrent", "5"),
					resource.TestCheckResourceAttr("openwebui_subagents_config.test", "subagents_max_async", "3"),
					resource.TestCheckResourceAttr("openwebui_subagents_config.test", "subagents_max_iterations", "10"),
					resource.TestCheckResourceAttr("openwebui_subagents_config.test", "subagents_max_output", "2048"),
					resource.TestCheckResourceAttr("openwebui_subagents_config.test", "subagents_system_prompt", "You are a helpful agent."),
					resource.TestCheckResourceAttrSet("openwebui_subagents_config.test", "id"),
				),
			},
			{
				// Idempotency: re-plan with same config, expect no diff
				Config: testAccProviderConfig() + `
resource "openwebui_subagents_config" "test" {
  enable_subagents              = true
  subagents_background_enabled  = false
  subagents_max_concurrent      = 5
  subagents_max_async           = 3
  subagents_max_iterations      = 10
  subagents_max_output          = 2048
  subagents_system_prompt       = "You are a helpful agent."
}`,
				PlanOnly: true,
			},
		},
	})
}

// TestAccAdminConfigResource applies a known admin config and verifies idempotency.
func TestAccAdminConfigResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "openwebui_admin_config" "test" {
  show_admin_details                   = true
  admin_email                          = "admin@test.local"
  webui_url                            = "http://localhost:3000"
  enable_signup                        = false
  enable_api_keys                      = true
  enable_api_keys_endpoint_restrictions = false
  api_keys_allowed_endpoints           = "/api/v1/chat/completions"
  default_user_role                    = "user"
  default_group_id                     = "default"
  jwt_expires_in                       = "30d"
  enable_community_sharing             = true
  enable_message_rating                = false
  enable_folders                       = true
  folder_max_file_count                = 100
  automation_max_count                 = 50
  automation_min_interval              = 1000
  enable_automations                   = true
  enable_channels                      = false
  channel_model_response_mode          = "channel"
  enable_calendar                      = true
  enable_memories                      = true
  enable_memory_system_context         = true
  enable_notes                         = false
  enable_user_webhooks                 = true
  enable_user_status                   = false
  pending_user_overlay_title           = "Pending Approval"
  pending_user_overlay_content         = "Your account is pending admin approval."
  response_watermark                   = "Powered by Open WebUI"
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_admin_config.test", "show_admin_details", "true"),
					resource.TestCheckResourceAttr("openwebui_admin_config.test", "admin_email", "admin@test.local"),
					resource.TestCheckResourceAttr("openwebui_admin_config.test", "webui_url", "http://localhost:3000"),
					resource.TestCheckResourceAttr("openwebui_admin_config.test", "enable_signup", "false"),
					resource.TestCheckResourceAttr("openwebui_admin_config.test", "enable_api_keys", "true"),
					resource.TestCheckResourceAttr("openwebui_admin_config.test", "default_user_role", "user"),
					resource.TestCheckResourceAttr("openwebui_admin_config.test", "folder_max_file_count", "100"),
					resource.TestCheckResourceAttr("openwebui_admin_config.test", "channel_model_response_mode", "channel"),
					resource.TestCheckResourceAttr("openwebui_admin_config.test", "enable_memory_system_context", "true"),
					resource.TestCheckResourceAttr("openwebui_admin_config.test", "pending_user_overlay_title", "Pending Approval"),
					resource.TestCheckResourceAttrSet("openwebui_admin_config.test", "id"),
				),
			},
			{
				// Idempotency: re-plan with same config, expect no diff
				Config: testAccProviderConfig() + `
resource "openwebui_admin_config" "test" {
  show_admin_details                   = true
  admin_email                          = "admin@test.local"
  webui_url                            = "http://localhost:3000"
  enable_signup                        = false
  enable_api_keys                      = true
  enable_api_keys_endpoint_restrictions = false
  api_keys_allowed_endpoints           = "/api/v1/chat/completions"
  default_user_role                    = "user"
  default_group_id                     = "default"
  jwt_expires_in                       = "30d"
  enable_community_sharing             = true
  enable_message_rating                = false
  enable_folders                       = true
  folder_max_file_count                = 100
  automation_max_count                 = 50
  automation_min_interval              = 1000
  enable_automations                   = true
  enable_channels                      = false
  channel_model_response_mode          = "channel"
  enable_calendar                      = true
  enable_memories                      = true
  enable_memory_system_context         = true
  enable_notes                         = false
  enable_user_webhooks                 = true
  enable_user_status                   = false
  pending_user_overlay_title           = "Pending Approval"
  pending_user_overlay_content         = "Your account is pending admin approval."
  response_watermark                   = "Powered by Open WebUI"
}`,
				PlanOnly: true,
			},
		},
	})
}

// TestAccLDAPConfigResource applies LDAP config and verifies idempotency.
func TestAccLDAPConfigResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "openwebui_ldap_config" "test" {
  enable_ldap = false
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_ldap_config.test", "enable_ldap", "false"),
					resource.TestCheckResourceAttrSet("openwebui_ldap_config.test", "id"),
				),
			},
			{
				// Idempotency: re-plan with same config, expect no diff
				Config: testAccProviderConfig() + `
resource "openwebui_ldap_config" "test" {
  enable_ldap = false
}`,
				PlanOnly: true,
			},
		},
	})
}

// TestAccLDAPServerConfigResource applies the LDAP server config twice: first
// with only the required attributes (checking server defaults and null/unset
// round-tripping), then with every attribute set including group management
// and the sensitive bind password, followed by a PlanOnly idempotency step.
func TestAccLDAPServerConfigResource(t *testing.T) {
	const name = "openwebui_ldap_server_config.test"
	full := testAccProviderConfig() + `
resource "openwebui_ldap_server_config" "test" {
  label                   = "Acc LDAP"
  host                    = "ldap.acc.example.com"
  port                    = 636
  attribute_for_mail      = "mail"
  attribute_for_username  = "sAMAccountName"
  app_dn                  = "cn=svc,dc=example,dc=com"
  app_dn_password         = "acc-test-password"
  search_base             = "ou=people,dc=example,dc=com"
  search_filters          = "(objectClass=person)"
  use_tls                 = false
  certificate_path        = "/etc/ssl/certs/ldap-ca.pem"
  validate_cert           = false
  ciphers                 = "HIGH"
  enable_group_management = true
  enable_group_creation   = true
  attribute_for_groups    = "memberOf"
}`
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "openwebui_ldap_server_config" "test" {
  label       = "Acc LDAP minimal"
  host        = "ldap.acc.example.com"
  search_base = "ou=people,dc=example,dc=com"
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "id", "ldap_server"),
					resource.TestCheckResourceAttr(name, "label", "Acc LDAP minimal"),
					resource.TestCheckResourceAttr(name, "attribute_for_mail", "mail"),
					resource.TestCheckResourceAttr(name, "attribute_for_username", "uid"),
					resource.TestCheckResourceAttr(name, "use_tls", "true"),
					resource.TestCheckResourceAttr(name, "validate_cert", "true"),
					resource.TestCheckResourceAttr(name, "ciphers", "ALL"),
					resource.TestCheckResourceAttr(name, "app_dn", ""),
					resource.TestCheckResourceAttr(name, "app_dn_password", ""),
					resource.TestCheckResourceAttr(name, "enable_group_management", "false"),
					resource.TestCheckResourceAttr(name, "attribute_for_groups", "memberOf"),
				),
			},
			{
				Config: testAccProviderConfig() + `
resource "openwebui_ldap_server_config" "test" {
  label       = "Acc LDAP minimal"
  host        = "ldap.acc.example.com"
  search_base = "ou=people,dc=example,dc=com"
}`,
				PlanOnly: true,
			},
			{
				Config: full,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "label", "Acc LDAP"),
					resource.TestCheckResourceAttr(name, "port", "636"),
					resource.TestCheckResourceAttr(name, "attribute_for_username", "sAMAccountName"),
					resource.TestCheckResourceAttr(name, "app_dn_password", "acc-test-password"),
					resource.TestCheckResourceAttr(name, "use_tls", "false"),
					resource.TestCheckResourceAttr(name, "validate_cert", "false"),
					resource.TestCheckResourceAttr(name, "certificate_path", "/etc/ssl/certs/ldap-ca.pem"),
					resource.TestCheckResourceAttr(name, "ciphers", "HIGH"),
					resource.TestCheckResourceAttr(name, "enable_group_management", "true"),
					resource.TestCheckResourceAttr(name, "enable_group_creation", "true"),
				),
			},
			{
				// Idempotency: re-plan with same config, expect no diff
				Config:   full,
				PlanOnly: true,
			},
		},
	})
}

// TestAccOAuthConfigResource covers the generic OIDC config: an empty apply
// (server defaults, null/"" round-tripping), a PlanOnly idempotency step, a
// full apply with every kind of value including the sensitive secret and the
// union-typed string attributes, and finally a restore to the stock defaults so
// the shared instance is left as it was found. Nothing here can lock the admin
// out: auto-redirect stays false and no provider URL is configured.
func TestAccOAuthConfigResource(t *testing.T) {
	const name = "openwebui_oauth_config.test"
	const empty = `
resource "openwebui_oauth_config" "test" {
}`
	full := testAccProviderConfig() + `
resource "openwebui_oauth_config" "test" {
  enable_oauth_signup               = false
  oauth_auto_redirect               = false
  oauth_allowed_domains             = "acc.example.com,example.org"
  oauth_blocked_groups              = "[]"
  enable_oauth_role_management      = false
  oauth_roles_claim                 = "acc_roles"
  oauth_admin_roles                 = "Acc Admin,admin"
  oauth_allowed_roles               = "user,admin"
  oauth_group_claim                 = "acc_groups"
  oauth_group_default_share         = "members"
  oauth_provider_name               = "Acc SSO"
  oauth_client_id                   = "acc-client"
  oauth_client_secret               = "acc-secret"
  oauth_scopes                      = "openid email"
  oauth_code_challenge_method       = "S256"
  oauth_token_endpoint_auth_method  = "client_secret_post"
  oauth_timeout                     = "30"
  oauth_client_timeout              = "45"
  oauth_email_claim                 = "acc_email"
  oauth_username_claim              = "acc_name"
  oauth_picture_claim               = "acc_picture"
  oauth_sub_claim                   = "acc_sub"
  oauth_audience                    = "acc-audience"
  oauth_update_email_on_login       = true
  oauth_update_name_on_login        = true
  oauth_update_picture_on_login     = true
  oauth_refresh_token_include_scope = true
}`
	restore := testAccProviderConfig() + `
resource "openwebui_oauth_config" "test" {
  oauth_auto_redirect               = false
  oauth_allowed_domains             = "*"
  oauth_blocked_groups              = "[]"
  oauth_roles_claim                 = "roles"
  oauth_admin_roles                 = "admin"
  oauth_allowed_roles               = "user,admin"
  oauth_group_claim                 = "groups"
  oauth_group_default_share         = "true"
  oauth_provider_name               = "SSO"
  oauth_client_id                   = ""
  oauth_client_secret               = ""
  oauth_scopes                      = "openid email profile"
  oauth_code_challenge_method       = ""
  oauth_token_endpoint_auth_method  = ""
  oauth_timeout                     = ""
  oauth_client_timeout              = ""
  oauth_email_claim                 = "email"
  oauth_username_claim              = "name"
  oauth_picture_claim               = "picture"
  oauth_sub_claim                   = ""
  oauth_audience                    = ""
  oauth_update_email_on_login       = false
  oauth_update_name_on_login        = false
  oauth_update_picture_on_login     = false
  oauth_refresh_token_include_scope = false
}`
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + empty,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "id", "oauth_config"),
					resource.TestCheckResourceAttrSet(name, "enable_oauth_signup"),
					resource.TestCheckResourceAttrSet(name, "oauth_scopes"),
					resource.TestCheckResourceAttr(name, "oauth_timeout", ""),
				),
			},
			{
				Config:   testAccProviderConfig() + empty,
				PlanOnly: true,
			},
			{
				Config: full,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "oauth_allowed_domains", "acc.example.com,example.org"),
					resource.TestCheckResourceAttr(name, "oauth_admin_roles", "Acc Admin,admin"),
					resource.TestCheckResourceAttr(name, "oauth_group_default_share", "members"),
					resource.TestCheckResourceAttr(name, "oauth_provider_name", "Acc SSO"),
					resource.TestCheckResourceAttr(name, "oauth_client_secret", "acc-secret"),
					resource.TestCheckResourceAttr(name, "oauth_code_challenge_method", "S256"),
					resource.TestCheckResourceAttr(name, "oauth_timeout", "30"),
					resource.TestCheckResourceAttr(name, "oauth_client_timeout", "45"),
					resource.TestCheckResourceAttr(name, "oauth_sub_claim", "acc_sub"),
					resource.TestCheckResourceAttr(name, "oauth_update_email_on_login", "true"),
					resource.TestCheckResourceAttr(name, "oauth_refresh_token_include_scope", "true"),
					resource.TestCheckResourceAttr(name, "oauth_auto_redirect", "false"),
				),
			},
			{
				Config:   full,
				PlanOnly: true,
			},
			{
				Config: restore,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "oauth_allowed_domains", "*"),
					resource.TestCheckResourceAttr(name, "oauth_group_default_share", "true"),
					resource.TestCheckResourceAttr(name, "oauth_client_secret", ""),
					resource.TestCheckResourceAttr(name, "oauth_timeout", ""),
					resource.TestCheckResourceAttr(name, "oauth_code_challenge_method", ""),
					resource.TestCheckResourceAttr(name, "oauth_update_email_on_login", "false"),
				),
			},
			{
				Config:   restore,
				PlanOnly: true,
			},
		},
	})
}

// TestAccImagesConfigResource covers the full-object images config through the
// provider's fetch-then-merge: an empty apply (every value read back from the
// server), a PlanOnly idempotency step, an apply touching every kind of value
// (secrets, nullable size/steps, JSON object and node-list attributes written
// with non-canonical key order and whitespace to prove semantic comparison,
// verbatim workflow text) and a restore to the stock values so the shared
// instance is left as found. The engine stays "openai" and generation stays
// disabled, so the server never contacts the placeholder engine URLs.
func TestAccImagesConfigResource(t *testing.T) {
	const name = "openwebui_images_config.test"
	const empty = `
resource "openwebui_images_config" "test" {
}`
	full := testAccProviderConfig() + `
resource "openwebui_images_config" "test" {
  enable_image_generation        = false
  enable_image_prompt_generation = false
  image_generation_engine        = "openai"
  image_size                     = "768x768"
  image_steps                    = 12
  images_openai_api_version      = "acc-version"
  images_openai_api_params_json  = "{ \"quality\": \"hd\",  \"n\": 2 }"
  automatic1111_base_url         = "http://a1111.invalid:7860"
  automatic1111_api_auth         = "acc:secret"
  automatic1111_params_json      = jsonencode({ cfg_scale = 7 })
  comfyui_base_url               = "http://comfy.invalid:8188"
  comfyui_api_key                = "acc-comfy-key"
  comfyui_workflow_nodes_json    = jsonencode([{ type = "prompt", key = "text", node_ids = ["6"] }])
  images_gemini_endpoint_method  = "predict"
  image_edit_size                = "1024x1024"
  images_edit_comfyui_workflow   = "{\"1\": {}}"

  images_edit_comfyui_workflow_nodes_json = "[ ]"
}`
	restore := testAccProviderConfig() + `
resource "openwebui_images_config" "test" {
  enable_image_generation        = false
  enable_image_prompt_generation = true
  image_generation_engine        = "openai"
  image_generation_model         = ""
  image_size                     = "512x512"
  image_steps                    = 50
  images_openai_api_version      = ""
  images_openai_api_params_json  = "{}"
  automatic1111_base_url         = ""
  automatic1111_api_auth         = ""
  automatic1111_params_json      = "{}"
  comfyui_base_url               = ""
  comfyui_api_key                = ""
  comfyui_workflow_nodes_json    = "[]"
  images_gemini_endpoint_method  = ""
  image_edit_size                = ""
  images_edit_comfyui_workflow   = ""

  images_edit_comfyui_workflow_nodes_json = "[]"
}`
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + empty,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "id", "images"),
					resource.TestCheckResourceAttrSet(name, "image_generation_engine"),
					resource.TestCheckResourceAttrSet(name, "image_steps"),
					resource.TestCheckResourceAttrSet(name, "comfyui_workflow"),
					resource.TestCheckResourceAttrSet(name, "comfyui_workflow_nodes_json"),
				),
			},
			{
				Config:   testAccProviderConfig() + empty,
				PlanOnly: true,
			},
			{
				Config: full,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "enable_image_prompt_generation", "false"),
					resource.TestCheckResourceAttr(name, "image_size", "768x768"),
					resource.TestCheckResourceAttr(name, "image_steps", "12"),
					resource.TestCheckResourceAttr(name, "images_openai_api_params_json", "{ \"quality\": \"hd\",  \"n\": 2 }"),
					resource.TestCheckResourceAttr(name, "automatic1111_api_auth", "acc:secret"),
					resource.TestCheckResourceAttr(name, "automatic1111_params_json", `{"cfg_scale":7}`),
					resource.TestCheckResourceAttr(name, "comfyui_base_url", "http://comfy.invalid:8188"),
					resource.TestCheckResourceAttr(name, "comfyui_api_key", "acc-comfy-key"),
					resource.TestCheckResourceAttr(name, "comfyui_workflow_nodes_json", `[{"key":"text","node_ids":["6"],"type":"prompt"}]`),
					resource.TestCheckResourceAttr(name, "image_edit_size", "1024x1024"),
					resource.TestCheckResourceAttr(name, "images_edit_comfyui_workflow", `{"1": {}}`),
					resource.TestCheckResourceAttr(name, "images_edit_comfyui_workflow_nodes_json", "[ ]"),
					resource.TestCheckResourceAttr(name, "image_generation_engine", "openai"),
				),
			},
			{
				Config:   full,
				PlanOnly: true,
			},
			{
				Config: restore,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "image_size", "512x512"),
					resource.TestCheckResourceAttr(name, "image_steps", "50"),
					resource.TestCheckResourceAttr(name, "images_openai_api_params_json", "{}"),
					resource.TestCheckResourceAttr(name, "automatic1111_api_auth", ""),
					resource.TestCheckResourceAttr(name, "comfyui_workflow_nodes_json", "[]"),
					resource.TestCheckResourceAttr(name, "image_edit_size", ""),
				),
			},
			{
				Config:   restore,
				PlanOnly: true,
			},
		},
	})
}

// TestAccAudioConfigResource covers the nested tts/stt audio config through the
// provider's fetch-then-merge: an empty apply (both sub-objects read back from
// the server), an apply with only `tts` (params JSON written with
// non-canonical whitespace), an apply with only `stt` (tts values must
// survive), and all three list shapes ([] / populated / omitted). Each apply
// step is followed by a PlanOnly idempotency step. stt.engine and
// stt.whisper_model are never changed, so the server only reloads the Whisper
// model it already has cached. The pre-test configuration is captured and
// POSTed back on cleanup, so the instance is left exactly as found.
func TestAccAudioConfigResource(t *testing.T) {
	testAccPreCheck(t)
	const name = "openwebui_audio_config.test"

	apiClient, err := client.NewClient(os.Getenv("OPENWEBUI_ENDPOINT"), os.Getenv("OPENWEBUI_TOKEN"), os.Getenv("OPENWEBUI_INSECURE") != "")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	original, err := apiClient.GetAudioConfig(context.Background())
	if err != nil {
		t.Fatalf("capture pre-test audio config: %v", err)
	}
	t.Cleanup(func() {
		if _, err := apiClient.SetAudioConfig(context.Background(), *original); err != nil {
			t.Errorf("restore pre-test audio config: %v", err)
		}
	})

	empty := testAccProviderConfig() + `
resource "openwebui_audio_config" "test" {
}`
	ttsOnly := testAccProviderConfig() + `
resource "openwebui_audio_config" "test" {
  tts = {
    voice              = "acc-voice"
    split_on           = "paragraphs"
    api_key            = "acc-tts-key"
    openai_params_json = "{ \"speed\":  1.25 }"
  }
}`
	sttOnly := testAccProviderConfig() + `
resource "openwebui_audio_config" "test" {
  stt = {
    supported_content_types      = []
    allowed_extensions           = ["mp3", "wav"]
    azure_max_speakers           = "2"
    azure_locales                = "en-US,de-DE"
    deepgram_api_key             = "acc-dg-key"
    mistral_use_chat_completions = true
    openai_api_request_format    = "json"
  }
}`
	sttLists := testAccProviderConfig() + `
resource "openwebui_audio_config" "test" {
  stt = {
    supported_content_types = ["audio/*", "video/webm"]
    allowed_extensions      = []
  }
}`
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: empty,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "id", "audio"),
					resource.TestCheckResourceAttr(name, "tts.model", original.TTS.Model),
					resource.TestCheckResourceAttr(name, "stt.whisper_model", original.STT.WhisperModel),
					resource.TestCheckResourceAttrSet(name, "stt.allowed_extensions.#"),
					resource.TestCheckResourceAttrSet(name, "stt.supported_content_types.#"),
				),
			},
			{Config: empty, PlanOnly: true},
			{
				Config: ttsOnly,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "tts.voice", "acc-voice"),
					resource.TestCheckResourceAttr(name, "tts.split_on", "paragraphs"),
					resource.TestCheckResourceAttr(name, "tts.api_key", "acc-tts-key"),
					resource.TestCheckResourceAttr(name, "tts.openai_params_json", "{ \"speed\":  1.25 }"),
					resource.TestCheckResourceAttr(name, "tts.engine", original.TTS.Engine),
					resource.TestCheckResourceAttr(name, "stt.engine", original.STT.Engine),
				),
			},
			{Config: ttsOnly, PlanOnly: true},
			{
				Config: sttOnly,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "stt.supported_content_types.#", "0"),
					resource.TestCheckResourceAttr(name, "stt.allowed_extensions.#", "2"),
					resource.TestCheckResourceAttr(name, "stt.allowed_extensions.1", "wav"),
					resource.TestCheckResourceAttr(name, "stt.azure_max_speakers", "2"),
					resource.TestCheckResourceAttr(name, "stt.azure_locales", "en-US,de-DE"),
					resource.TestCheckResourceAttr(name, "stt.deepgram_api_key", "acc-dg-key"),
					resource.TestCheckResourceAttr(name, "stt.mistral_use_chat_completions", "true"),
					resource.TestCheckResourceAttr(name, "stt.openai_api_request_format", "json"),
					resource.TestCheckResourceAttr(name, "stt.whisper_model", original.STT.WhisperModel),
					// tts was removed from the configuration: values persist.
					resource.TestCheckResourceAttr(name, "tts.voice", "acc-voice"),
					resource.TestCheckResourceAttr(name, "tts.openai_params_json", "{ \"speed\":  1.25 }"),
				),
			},
			{Config: sttOnly, PlanOnly: true},
			{
				Config: sttLists,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "stt.supported_content_types.#", "2"),
					resource.TestCheckResourceAttr(name, "stt.supported_content_types.0", "audio/*"),
					resource.TestCheckResourceAttr(name, "stt.allowed_extensions.#", "0"),
					resource.TestCheckResourceAttr(name, "stt.deepgram_api_key", "acc-dg-key"),
				),
			},
			{Config: sttLists, PlanOnly: true},
		},
	})
}

// TestAccTerminalServersConfigResource applies a two-entry connection list
// (one fully specified, one relying on defaults) and then a reordered
// single-entry list, each followed by a PlanOnly idempotency step. The URLs
// point at a closed local port: Open WebUI contacts every enabled connection
// after the POST to cache its OpenAPI spec, but a refused connection is
// logged, not an error. The pre-test connection list is captured and POSTed
// back on cleanup.
func TestAccTerminalServersConfigResource(t *testing.T) {
	testAccPreCheck(t)
	const name = "openwebui_terminal_servers_config.test"

	apiClient, err := client.NewClient(os.Getenv("OPENWEBUI_ENDPOINT"), os.Getenv("OPENWEBUI_TOKEN"), os.Getenv("OPENWEBUI_INSECURE") != "")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	original, err := apiClient.GetTerminalServersConfig(context.Background())
	if err != nil {
		t.Fatalf("capture pre-test terminal servers config: %v", err)
	}
	t.Cleanup(func() {
		if _, err := apiClient.SetTerminalServersConfig(context.Background(), *original); err != nil {
			t.Errorf("restore pre-test terminal servers config: %v", err)
		}
	})

	two := testAccProviderConfig() + `
resource "openwebui_terminal_servers_config" "test" {
  connections = [
    {
      id          = "acc-term-a"
      name        = "Acc A"
      url         = "http://127.0.0.1:1"
      enabled     = false
      key         = "acc-secret"
      auth_type   = "bearer"
      server_type = "terminal"
      config_json = "{ \"access_grants\": [],  \"note\": \"acc\" }"
    },
    {
      id  = "acc-term-b"
      url = "http://127.0.0.1:1"
    },
  ]
}`
	one := testAccProviderConfig() + `
resource "openwebui_terminal_servers_config" "test" {
  connections = [
    {
      id        = "acc-term-b"
      name      = "Acc B"
      url       = "http://127.0.0.1:1"
      path      = "/spec.json"
      auth_type = "none"
      enabled   = false
    },
  ]
}`
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: two,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "id", "terminal_servers"),
					resource.TestCheckResourceAttr(name, "connections.#", "2"),
					resource.TestCheckResourceAttr(name, "connections.0.id", "acc-term-a"),
					resource.TestCheckResourceAttr(name, "connections.0.enabled", "false"),
					resource.TestCheckResourceAttr(name, "connections.0.key", "acc-secret"),
					resource.TestCheckResourceAttr(name, "connections.0.server_type", "terminal"),
					resource.TestCheckResourceAttr(name, "connections.0.config_json", "{ \"access_grants\": [],  \"note\": \"acc\" }"),
					resource.TestCheckResourceAttr(name, "connections.1.id", "acc-term-b"),
					resource.TestCheckResourceAttr(name, "connections.1.name", ""),
					resource.TestCheckResourceAttr(name, "connections.1.enabled", "true"),
					resource.TestCheckResourceAttr(name, "connections.1.path", "/openapi.json"),
					resource.TestCheckResourceAttr(name, "connections.1.auth_type", "bearer"),
					resource.TestCheckNoResourceAttr(name, "connections.1.config_json"),
				),
			},
			{Config: two, PlanOnly: true},
			{
				Config: one,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "connections.#", "1"),
					resource.TestCheckResourceAttr(name, "connections.0.id", "acc-term-b"),
					resource.TestCheckResourceAttr(name, "connections.0.path", "/spec.json"),
					resource.TestCheckResourceAttr(name, "connections.0.auth_type", "none"),
					resource.TestCheckResourceAttr(name, "connections.0.key", ""),
				),
			},
			{Config: one, PlanOnly: true},
		},
	})
}
