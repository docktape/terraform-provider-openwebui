# Changelog

## [1.2.0] - 2026-11-06

Adds support for Open WebUI v0.11.0. Supported range is now v0.9.0 – v0.11.0.

### Upgrade notes

- `openwebui_tool`, `openwebui_knowledge`, `openwebui_model`, `openwebui_prompt` and `openwebui_skill` now validate at plan time that every entry in `write_groups` is also in `read_groups`. Open WebUI always stores write groups as read groups too; such configurations previously failed at apply time with "Provider produced inconsistent result after apply".
- `openwebui_prompt` and `openwebui_knowledge` now store empty group lists as `[]` instead of null. This is a state-only change that shows up on the next refresh.
- Several new admin config resources write the whole config object on every apply, so attributes left unset are reset to their defaults. Each affected resource's documentation says so.

### Bug Fixes

- **model:** Stop stripping `file_context`, `terminal`, `builtin_tools` and `memory` capabilities on apply
- **model:** Add the new capability keys to the `openwebui_model` data source
- **knowledge_file:** Stop hardcoding page 1 when looking up a file by ID
- **access-control:** Keep write_groups a subset of read_groups at plan time
- **client:** Longer timeout for model-reloading retrieval calls

### Documentation

- Bump compatibility range to Open WebUI v0.9.0 – v0.11.0
- **knowledge:** Mark `data_json`/`meta_json` as deprecated and inert
- Document known behavior gaps found in the v0.11.0 compatibility audit

### Features

- **group:** Support Open WebUI v0.10.0/v0.11.0 permission keys (`skills_import`, `skills_export`, sharing `folders`, chat `import`, features `webhooks`, access grants `allow_groups`)
- **models_config:** Support `default_model_metadata_json` and `default_model_params_json`
- **skill:** Add `openwebui_skill` resource and data source
- **tasks_config:** Add `openwebui_tasks_config` resource
- **chat_config:** Add `openwebui_chat_config` resource
- **evaluation_arena_config:** Add `openwebui_evaluation_arena_config` resource
- **subagents_config:** Add `openwebui_subagents_config` resource
- **admin_config:** Add `openwebui_admin_config` resource
- **ldap_config:** Add `openwebui_ldap_config` resource
- **ldap_server_config:** Add `openwebui_ldap_server_config` resource
- **oauth_config:** Add `openwebui_oauth_config` resource (Open WebUI v0.10.0+)
- **images_config:** Add `openwebui_images_config` resource
- **audio_config:** Add `openwebui_audio_config` resource
- **terminal_servers_config:** Add `openwebui_terminal_servers_config` resource and `openwebui_terminal_server_verify` data source
- **retrieval_config:** Add `openwebui_retrieval_config` resource
- **web_search_config:** Add `openwebui_web_search_config` resource
- **retrieval_embedding_config:** Add `openwebui_retrieval_embedding_config` resource

## [1.1.0] - 2026-06-09

### Bug Fixes

- **group:** Fix acc test TF_ACC guard and add update step
- **prompt:** Handle nil tags as null list, use state for unknown on is_active and tags
- **model:** Add hidden=false and null coverage in tests
- Data source group schema and prompt acc test

### Documentation

- Update README for v0.9.6 coverage
- Regenerate provider docs for group, model, and prompt changes
- Update provider version constraint to ~> 1.0

### Features

- **group:** Add access_grants and settings permission categories
- **prompt:** Add is_active, tags, data_json, meta_json fields
- **model:** Add hidden field

## [1.0.2] - 2026-06-09

### Bug Fixes

- **group:** Add UseStateForUnknown to permissions block

## [1.0.1] - 2026-06-09

### Bug Fixes

- Forward-compat with new OpenWebUI permissions keys
- Add missing sharing and features permission keys for v0.9.5+

## [1.0.0] - 2026-06-03

### Bug Fixes

- **provider:** Mark derived function attrs as known-after-apply
- **client:** Send model delete id in request body
- **client:** Read group membership via export endpoint
- **client:** Drop unsupported limit param from user search
- **provider:** Stop caching server-derived updated_at/timestamp values
- **client:** Treat 401 NOT_FOUND group response as not found
- **client:** List knowledge via paginated GET /knowledge/
- **client:** List files via paginated GET /files/
- Make testacc cross-platform (Windows cmd.exe vs Unix)
- **client:** Clone DefaultTransport to preserve proxy and dial settings
- **client:** Guard against nil map in connection encode functions
- **provider:** Preserve keys from prior state in Read; treat empty headers as null
- **provider:** Rename reserved 'provider' attribute to 'api_provider'
- **model:** Resolve unknown value error when capabilities block is omitted
- **model:** Fix null meta fields leaking into additional JSON and merge precedence
- **group:** Prevent null users list causing inconsistent apply result
- **provider:** Accept base URL without /api/v1 suffix — append internally
- **client:** Guard against /api/v1 in endpoint; add trailing-slash and guard tests
- **function:** Suppress user_id churn with UseStateForUnknown

### Documentation

- Track function resource docs and example
- Improve provider schema description
- Schema descriptions for knowledge resource and data source
- Schema descriptions for knowledge_file resource
- Schema descriptions for model resource and data source
- Schema descriptions for prompt resource and data source
- Schema descriptions for group resource and data source
- Schema descriptions for tool resources and data source
- Schema descriptions for pipeline resources and data source
- Schema descriptions for file resources and data sources
- Schema descriptions for function resource and valves
- Schema descriptions for config import resource and export data source
- Schema descriptions for connections and tool_servers config resources
- Schema descriptions for code_execution and models config resources
- Schema descriptions for suggestions and banners config resources
- Schema descriptions for oauth_client, user, and tool_server_verify
- Add provider example
- Add examples for knowledge, knowledge_file, model resources
- Add examples for prompt and group resources
- Add examples for tool, tool_valves, pipeline, pipeline_valves resources
- Add examples for file and admin config resources
- Add examples for oauth_client, function, function_valves resources
- Add examples for all data sources
- Generate Registry docs via tfplugindocs
- Regenerate provider docs (insecure_skip_verify, model fields, pipeline key)
- Generate docs for openai_connections, ollama_connections resources and verify data sources
- **model:** Fix params and capabilities to attribute assignment syntax
- Add README with provider overview, resources, data sources, and examples

### Features

- **client:** Add function create/get/delete
- **client:** Add function update and toggle methods
- **client:** Add function valves methods
- **provider:** Add function toggle-convergence helper
- **provider:** Add openwebui_function resource
- **provider:** Add openwebui_function_valves resource
- **provider:** Register function resources
- **client:** Add access_control<->access_grants converters
- **client:** Translate model access_control to access_grants
- **client:** Translate tool access_control to access_grants
- **client:** Translate knowledge access_control to access_grants
- **client:** Rework prompts for UUID routing and access_grants
- **provider:** Align prompt resource to UUID/name/created_at API
- **provider:** Look up prompt data source by command via list
- **client:** Add insecure bool to NewClient for TLS skip
- **provider:** Add insecure_skip_verify attribute and OPENWEBUI_INSECURE env var
- **client:** Add rootURL field and doRaw helper for non-api/v1 endpoints
- **client:** Add connection types and encode/decode helpers
- **client:** Add httptest-based tests for GetOpenAIConnections and SetOpenAIConnections
- **client:** Add GetOllamaConnections and SetOllamaConnections methods
- **client:** Add VerifyOpenAIConnection and VerifyOllamaConnection tests
- **provider:** Add openwebui_openai_connections resource
- **provider:** Add openwebui_ollama_connections resource
- **provider:** Add data.openwebui_openai_connection_verify and data.openwebui_ollama_connection_verify data sources
- **provider:** Register openai_connections, ollama_connections resources and verify data sources
- **provider:** Require endpoint — remove silent default, fail fast if unset

