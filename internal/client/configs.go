package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// ConnectionsConfigForm captures connection configuration.
type ConnectionsConfigForm struct {
	EnableDirectConnections bool `json:"ENABLE_DIRECT_CONNECTIONS"`
	EnableBaseModelsCache   bool `json:"ENABLE_BASE_MODELS_CACHE"`
}

// ImportConfigForm captures the config import payload.
type ImportConfigForm struct {
	Config map[string]any `json:"config"`
}

// OAuthClientRegistrationForm registers an OAuth client.
type OAuthClientRegistrationForm struct {
	URL        string  `json:"url"`
	ClientID   string  `json:"client_id"`
	ClientName *string `json:"client_name,omitempty"`
}

// ToolServerConnection captures a tool server connection entry.
type ToolServerConnection struct {
	URL      string         `json:"url"`
	Path     string         `json:"path"`
	Type     *string        `json:"type,omitempty"`
	AuthType *string        `json:"auth_type"`
	Headers  any            `json:"headers,omitempty"`
	Key      *string        `json:"key"`
	Config   map[string]any `json:"config"`
}

// ToolServersConfigForm captures tool server configuration.
type ToolServersConfigForm struct {
	Connections []ToolServerConnection `json:"TOOL_SERVER_CONNECTIONS"`
}

// CodeInterpreterConfigForm captures code execution configuration.
type CodeInterpreterConfigForm struct {
	EnableCodeExecution                bool    `json:"ENABLE_CODE_EXECUTION"`
	CodeExecutionEngine                string  `json:"CODE_EXECUTION_ENGINE"`
	CodeExecutionJupyterURL            *string `json:"CODE_EXECUTION_JUPYTER_URL"`
	CodeExecutionJupyterAuth           *string `json:"CODE_EXECUTION_JUPYTER_AUTH"`
	CodeExecutionJupyterAuthToken      *string `json:"CODE_EXECUTION_JUPYTER_AUTH_TOKEN"`
	CodeExecutionJupyterAuthPassword   *string `json:"CODE_EXECUTION_JUPYTER_AUTH_PASSWORD"`
	CodeExecutionJupyterTimeout        *int64  `json:"CODE_EXECUTION_JUPYTER_TIMEOUT"`
	EnableCodeInterpreter              bool    `json:"ENABLE_CODE_INTERPRETER"`
	CodeInterpreterEngine              string  `json:"CODE_INTERPRETER_ENGINE"`
	CodeInterpreterPromptTemplate      *string `json:"CODE_INTERPRETER_PROMPT_TEMPLATE"`
	CodeInterpreterJupyterURL          *string `json:"CODE_INTERPRETER_JUPYTER_URL"`
	CodeInterpreterJupyterAuth         *string `json:"CODE_INTERPRETER_JUPYTER_AUTH"`
	CodeInterpreterJupyterAuthToken    *string `json:"CODE_INTERPRETER_JUPYTER_AUTH_TOKEN"`
	CodeInterpreterJupyterAuthPassword *string `json:"CODE_INTERPRETER_JUPYTER_AUTH_PASSWORD"`
	CodeInterpreterJupyterTimeout      *int64  `json:"CODE_INTERPRETER_JUPYTER_TIMEOUT"`
}

// ModelsConfigForm captures default model configuration.
type ModelsConfigForm struct {
	DefaultModels        *string        `json:"DEFAULT_MODELS"`
	DefaultPinnedModels  *string        `json:"DEFAULT_PINNED_MODELS"`
	ModelOrderList       []string       `json:"MODEL_ORDER_LIST"`
	DefaultModelMetadata map[string]any `json:"DEFAULT_MODEL_METADATA,omitempty"`
	DefaultModelParams   map[string]any `json:"DEFAULT_MODEL_PARAMS,omitempty"`
}

// ChatConfigForm captures chat context-compaction settings.
type ChatConfigForm struct {
	EnableContextCompaction              bool   `json:"ENABLE_CONTEXT_COMPACTION"`
	ContextCompactionTokenThreshold      int64  `json:"CONTEXT_COMPACTION_TOKEN_THRESHOLD"`
	ContextCompactionTokenCap            *int64 `json:"CONTEXT_COMPACTION_TOKEN_CAP"`
	ContextCompactionRetentionPercentage int64  `json:"CONTEXT_COMPACTION_RETENTION_PERCENTAGE"`
	ContextCompactionPromptTemplate      string `json:"CONTEXT_COMPACTION_PROMPT_TEMPLATE"`
	ContextCompactionModel               string `json:"CONTEXT_COMPACTION_MODEL"`
}

// TaskConfigForm captures title/tag/autocomplete/follow-up task-model settings.
type TaskConfigForm struct {
	TaskModel                            *string `json:"TASK_MODEL"`
	TaskModelExternal                    *string `json:"TASK_MODEL_EXTERNAL"`
	EnableTitleGeneration                bool    `json:"ENABLE_TITLE_GENERATION"`
	TitleGenerationPromptTemplate        string  `json:"TITLE_GENERATION_PROMPT_TEMPLATE"`
	ImagePromptGenerationPromptTemplate  string  `json:"IMAGE_PROMPT_GENERATION_PROMPT_TEMPLATE"`
	EnableAutocompleteGeneration         bool    `json:"ENABLE_AUTOCOMPLETE_GENERATION"`
	AutocompleteGenerationInputMaxLength int64   `json:"AUTOCOMPLETE_GENERATION_INPUT_MAX_LENGTH"`
	AutocompleteGenerationPromptTemplate string  `json:"AUTOCOMPLETE_GENERATION_PROMPT_TEMPLATE"`
	TagsGenerationPromptTemplate         string  `json:"TAGS_GENERATION_PROMPT_TEMPLATE"`
	FollowUpGenerationPromptTemplate     string  `json:"FOLLOW_UP_GENERATION_PROMPT_TEMPLATE"`
	EnableFollowUpGeneration             bool    `json:"ENABLE_FOLLOW_UP_GENERATION"`
	EnableTagsGeneration                 bool    `json:"ENABLE_TAGS_GENERATION"`
	EnableSearchQueryGeneration          bool    `json:"ENABLE_SEARCH_QUERY_GENERATION"`
	EnableRetrievalQueryGeneration       bool    `json:"ENABLE_RETRIEVAL_QUERY_GENERATION"`
	QueryGenerationPromptTemplate        string  `json:"QUERY_GENERATION_PROMPT_TEMPLATE"`
	ToolsFunctionCallingPromptTemplate   string  `json:"TOOLS_FUNCTION_CALLING_PROMPT_TEMPLATE"`
	EnableVoiceModePrompt                bool    `json:"ENABLE_VOICE_MODE_PROMPT"`
	VoiceModePromptTemplate              *string `json:"VOICE_MODE_PROMPT_TEMPLATE"`
}

// PromptSuggestion captures a default prompt suggestion.
type PromptSuggestion struct {
	Title   []string `json:"title"`
	Content string   `json:"content"`
}

// SetDefaultSuggestionsForm updates default suggestions.
type SetDefaultSuggestionsForm struct {
	Suggestions []PromptSuggestion `json:"suggestions"`
}

// BannerModel captures banner configuration.
type BannerModel struct {
	ID          string  `json:"id"`
	Type        string  `json:"type"`
	Title       *string `json:"title,omitempty"`
	Content     string  `json:"content"`
	Dismissible bool    `json:"dismissible"`
	Timestamp   int64   `json:"timestamp"`
}

// SetBannersForm updates banners configuration.
type SetBannersForm struct {
	Banners []BannerModel `json:"banners"`
}

// GetConnectionsConfig retrieves connections config.
func (c *Client) GetConnectionsConfig(ctx context.Context) (*ConnectionsConfigForm, error) {
	var resp ConnectionsConfigForm
	if err := c.do(ctx, http.MethodGet, "configs/connections", nil, nil, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// SetConnectionsConfig updates connections config.
func (c *Client) SetConnectionsConfig(ctx context.Context, form ConnectionsConfigForm) (*ConnectionsConfigForm, error) {
	var resp ConnectionsConfigForm
	if err := c.do(ctx, http.MethodPost, "configs/connections", nil, form, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetToolServersConfig retrieves tool server configuration.
func (c *Client) GetToolServersConfig(ctx context.Context) (*ToolServersConfigForm, error) {
	var resp ToolServersConfigForm
	if err := c.do(ctx, http.MethodGet, "configs/tool_servers", nil, nil, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// SetToolServersConfig updates tool server configuration.
func (c *Client) SetToolServersConfig(ctx context.Context, form ToolServersConfigForm) (*ToolServersConfigForm, error) {
	var resp ToolServersConfigForm
	if err := c.do(ctx, http.MethodPost, "configs/tool_servers", nil, form, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetCodeExecutionConfig retrieves code execution configuration.
func (c *Client) GetCodeExecutionConfig(ctx context.Context) (*CodeInterpreterConfigForm, error) {
	var resp CodeInterpreterConfigForm
	if err := c.do(ctx, http.MethodGet, "configs/code_execution", nil, nil, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// SetCodeExecutionConfig updates code execution configuration.
func (c *Client) SetCodeExecutionConfig(ctx context.Context, form CodeInterpreterConfigForm) (*CodeInterpreterConfigForm, error) {
	var resp CodeInterpreterConfigForm
	if err := c.do(ctx, http.MethodPost, "configs/code_execution", nil, form, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetModelsConfig retrieves models configuration.
func (c *Client) GetModelsConfig(ctx context.Context) (*ModelsConfigForm, error) {
	var resp ModelsConfigForm
	if err := c.do(ctx, http.MethodGet, "configs/models", nil, nil, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// SetModelsConfig updates models configuration.
func (c *Client) SetModelsConfig(ctx context.Context, form ModelsConfigForm) (*ModelsConfigForm, error) {
	var resp ModelsConfigForm
	if err := c.do(ctx, http.MethodPost, "configs/models", nil, form, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetBanners retrieves banners configuration.
func (c *Client) GetBanners(ctx context.Context) ([]BannerModel, error) {
	var resp []BannerModel
	if err := c.do(ctx, http.MethodGet, "configs/banners", nil, nil, &resp); err != nil {
		return nil, err
	}

	return resp, nil
}

// SetBanners updates banners configuration.
func (c *Client) SetBanners(ctx context.Context, banners []BannerModel) ([]BannerModel, error) {
	var resp []BannerModel
	form := SetBannersForm{Banners: banners}
	if err := c.do(ctx, http.MethodPost, "configs/banners", nil, form, &resp); err != nil {
		return nil, err
	}

	return resp, nil
}

// SetDefaultSuggestions updates default suggestions configuration.
func (c *Client) SetDefaultSuggestions(ctx context.Context, suggestions []PromptSuggestion) ([]PromptSuggestion, error) {
	var resp []PromptSuggestion
	form := SetDefaultSuggestionsForm{Suggestions: suggestions}
	if err := c.do(ctx, http.MethodPost, "configs/suggestions", nil, form, &resp); err != nil {
		return nil, err
	}

	return resp, nil
}

// ExportConfig retrieves the full configuration export.
func (c *Client) ExportConfig(ctx context.Context) (map[string]any, error) {
	var resp map[string]any
	if err := c.do(ctx, http.MethodGet, "configs/export", nil, nil, &resp); err != nil {
		return nil, err
	}

	return resp, nil
}

// ImportConfig applies a configuration export payload.
func (c *Client) ImportConfig(ctx context.Context, config map[string]any) (map[string]any, error) {
	var resp map[string]any
	form := ImportConfigForm{Config: config}
	if err := c.do(ctx, http.MethodPost, "configs/import", nil, form, &resp); err != nil {
		return nil, err
	}

	return resp, nil
}

// RegisterOAuthClient registers an OAuth client with optional type.
func (c *Client) RegisterOAuthClient(ctx context.Context, form OAuthClientRegistrationForm, clientType *string) (map[string]any, error) {
	var resp map[string]any
	var query url.Values
	if clientType != nil {
		query = url.Values{}
		query.Set("type", *clientType)
	}

	if err := c.do(ctx, http.MethodPost, "configs/oauth/clients/register", query, form, &resp); err != nil {
		return nil, err
	}

	return resp, nil
}

// VerifyToolServer verifies tool server connectivity.
func (c *Client) VerifyToolServer(ctx context.Context, connection ToolServerConnection) error {
	if err := c.do(ctx, http.MethodPost, "configs/tool_servers/verify", nil, connection, nil); err != nil {
		return err
	}

	return nil
}

// GetChatConfig retrieves chat context-compaction configuration.
func (c *Client) GetChatConfig(ctx context.Context) (*ChatConfigForm, error) {
	var resp ChatConfigForm
	if err := c.do(ctx, http.MethodGet, "chats/config", nil, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// SetChatConfig updates chat context-compaction configuration.
func (c *Client) SetChatConfig(ctx context.Context, form ChatConfigForm) (*ChatConfigForm, error) {
	var resp ChatConfigForm
	if err := c.do(ctx, http.MethodPost, "chats/config", nil, form, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetTasksConfig retrieves task-model generation settings.
func (c *Client) GetTasksConfig(ctx context.Context) (*TaskConfigForm, error) {
	var resp TaskConfigForm
	if err := c.do(ctx, http.MethodGet, "tasks/config", nil, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// SetTasksConfig updates task-model generation settings.
func (c *Client) SetTasksConfig(ctx context.Context, form TaskConfigForm) (*TaskConfigForm, error) {
	var resp TaskConfigForm
	if err := c.do(ctx, http.MethodPost, "tasks/config/update", nil, form, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ArenaModelMeta captures the metadata block of an evaluation arena model entry.
type ArenaModelMeta struct {
	ProfileImageURL *string  `json:"profile_image_url,omitempty"`
	Description     *string  `json:"description,omitempty"`
	ModelIDs        []string `json:"model_ids"`
}

// ArenaModel captures a single evaluation arena model entry. The Open WebUI
// backend models EVALUATION_ARENA_MODELS as a list of these objects, not a
// list of plain model-ID strings.
type ArenaModel struct {
	ID   string          `json:"id"`
	Name string          `json:"name"`
	Meta *ArenaModelMeta `json:"meta,omitempty"`
}

// EvaluationArenaConfigForm captures evaluation arena configuration.
//
// EnableEvaluationArenaModels is nullable because the real backend
// (backend/open_webui/routers/evaluations.py's UpdateConfigForm) treats
// ENABLE_EVALUATION_ARENA_MODELS as Optional[bool] = None and only applies it
// when not None - so a config that sets only evaluation_arena_models must be
// able to send this key omitted rather than forcing it to false.
type EvaluationArenaConfigForm struct {
	EnableEvaluationArenaModels *bool        `json:"ENABLE_EVALUATION_ARENA_MODELS,omitempty"`
	EvaluationArenaModels       []ArenaModel `json:"EVALUATION_ARENA_MODELS"`
}

// GetEvaluationArenaConfig retrieves evaluation arena configuration.
func (c *Client) GetEvaluationArenaConfig(ctx context.Context) (*EvaluationArenaConfigForm, error) {
	var resp EvaluationArenaConfigForm
	if err := c.do(ctx, http.MethodGet, "evaluations/config", nil, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// SetEvaluationArenaConfig updates evaluation arena configuration.
func (c *Client) SetEvaluationArenaConfig(ctx context.Context, form EvaluationArenaConfigForm) (*EvaluationArenaConfigForm, error) {
	var resp EvaluationArenaConfigForm
	if err := c.do(ctx, http.MethodPost, "evaluations/config", nil, form, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// SubagentsConfigForm captures sub-agent orchestration settings.
type SubagentsConfigForm struct {
	EnableSubagents            bool   `json:"ENABLE_SUBAGENTS"`
	SubagentsBackgroundEnabled bool   `json:"SUBAGENTS_BACKGROUND_ENABLED"`
	SubagentsMaxConcurrent     int64  `json:"SUBAGENTS_MAX_CONCURRENT"`
	SubagentsMaxAsync          int64  `json:"SUBAGENTS_MAX_ASYNC"`
	SubagentsMaxIterations     int64  `json:"SUBAGENTS_MAX_ITERATIONS"`
	SubagentsMaxOutput         int64  `json:"SUBAGENTS_MAX_OUTPUT"`
	SubagentsSystemPrompt      string `json:"SUBAGENTS_SYSTEM_PROMPT"`
}

// GetSubagentsConfig retrieves sub-agent orchestration configuration.
func (c *Client) GetSubagentsConfig(ctx context.Context) (*SubagentsConfigForm, error) {
	var resp SubagentsConfigForm
	if err := c.do(ctx, http.MethodGet, "configs/subagents", nil, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// SetSubagentsConfig updates sub-agent orchestration configuration.
func (c *Client) SetSubagentsConfig(ctx context.Context, form SubagentsConfigForm) (*SubagentsConfigForm, error) {
	var resp SubagentsConfigForm
	if err := c.do(ctx, http.MethodPost, "configs/subagents", nil, form, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// AdminConfigForm captures general admin settings (signup, API keys, JWT,
// feature toggles, watermarking).
type AdminConfigForm struct {
	ShowAdminDetails                  bool    `json:"SHOW_ADMIN_DETAILS"`
	AdminEmail                        *string `json:"ADMIN_EMAIL"`
	WebUIURL                          string  `json:"WEBUI_URL"`
	EnableSignup                      bool    `json:"ENABLE_SIGNUP"`
	EnableAPIKeys                     bool    `json:"ENABLE_API_KEYS"`
	EnableAPIKeysEndpointRestrictions bool    `json:"ENABLE_API_KEYS_ENDPOINT_RESTRICTIONS"`
	APIKeysAllowedEndpoints           string  `json:"API_KEYS_ALLOWED_ENDPOINTS"`
	DefaultUserRole                   string  `json:"DEFAULT_USER_ROLE"`
	DefaultGroupID                    string  `json:"DEFAULT_GROUP_ID"`
	JWTExpiresIn                      string  `json:"JWT_EXPIRES_IN"`
	EnableCommunitySharing            bool    `json:"ENABLE_COMMUNITY_SHARING"`
	EnableMessageRating               bool    `json:"ENABLE_MESSAGE_RATING"`
	EnableFolders                     bool    `json:"ENABLE_FOLDERS"`
	FolderMaxFileCount                *int64  `json:"FOLDER_MAX_FILE_COUNT"`
	AutomationMaxCount                *int64  `json:"AUTOMATION_MAX_COUNT"`
	AutomationMinInterval             *int64  `json:"AUTOMATION_MIN_INTERVAL"`
	EnableAutomations                 bool    `json:"ENABLE_AUTOMATIONS"`
	EnableChannels                    bool    `json:"ENABLE_CHANNELS"`
	ChannelModelResponseMode          string  `json:"CHANNEL_MODEL_RESPONSE_MODE"`
	EnableCalendar                    bool    `json:"ENABLE_CALENDAR"`
	EnableMemories                    bool    `json:"ENABLE_MEMORIES"`
	EnableMemorySystemContext         bool    `json:"ENABLE_MEMORY_SYSTEM_CONTEXT"`
	EnableNotes                       bool    `json:"ENABLE_NOTES"`
	EnableUserWebhooks                bool    `json:"ENABLE_USER_WEBHOOKS"`
	EnableUserStatus                  bool    `json:"ENABLE_USER_STATUS"`
	PendingUserOverlayTitle           *string `json:"PENDING_USER_OVERLAY_TITLE"`
	PendingUserOverlayContent         *string `json:"PENDING_USER_OVERLAY_CONTENT"`
	ResponseWatermark                 *string `json:"RESPONSE_WATERMARK"`
}

// UnmarshalJSON decodes AdminConfigForm, tolerating FOLDER_MAX_FILE_COUNT,
// AUTOMATION_MAX_COUNT, and AUTOMATION_MIN_INTERVAL being served as either a
// JSON number or the JSON string "". The real backend
// (backend/open_webui/config.py) defaults each of these via
// os.getenv(key, ""), and the update handler
// (backend/open_webui/routers/auths.py's update_admin_config) stores
// int(v) if v else "" - so on a stock, never-configured instance (or any
// instance where one of these was reset to empty) the GET response serves
// "" for that key rather than omitting it or serving null. A plain *int64
// field fails to unmarshal a JSON string at all, which crashed GetAdminConfig
// against a fresh server before any write had ever happened.
func (f *AdminConfigForm) UnmarshalJSON(data []byte) error {
	type alias AdminConfigForm
	aux := struct {
		FolderMaxFileCount    json.RawMessage `json:"FOLDER_MAX_FILE_COUNT"`
		AutomationMaxCount    json.RawMessage `json:"AUTOMATION_MAX_COUNT"`
		AutomationMinInterval json.RawMessage `json:"AUTOMATION_MIN_INTERVAL"`
		*alias
	}{alias: (*alias)(f)}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	var err error
	if f.FolderMaxFileCount, err = decodeIntOrEmptyString(aux.FolderMaxFileCount); err != nil {
		return fmt.Errorf("FOLDER_MAX_FILE_COUNT: %w", err)
	}
	if f.AutomationMaxCount, err = decodeIntOrEmptyString(aux.AutomationMaxCount); err != nil {
		return fmt.Errorf("AUTOMATION_MAX_COUNT: %w", err)
	}
	if f.AutomationMinInterval, err = decodeIntOrEmptyString(aux.AutomationMinInterval); err != nil {
		return fmt.Errorf("AUTOMATION_MIN_INTERVAL: %w", err)
	}

	return nil
}

// decodeIntOrEmptyString decodes a JSON value that may be a number, the
// empty string, a non-empty numeric string, or absent/null - returning nil
// for all of the "no value set" cases (absent, null, or "").
func decodeIntOrEmptyString(raw json.RawMessage) (*int64, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}

	var n int64
	if err := json.Unmarshal(raw, &n); err == nil {
		return &n, nil
	}

	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("expected a JSON number or string, got %s", string(raw))
	}
	if s == "" {
		return nil, nil
	}

	parsed, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("non-numeric string %q: %w", s, err)
	}
	return &parsed, nil
}

// LDAPConfigForm captures the LDAP enable toggle as returned by both the GET
// and POST /auths/admin/config/ldap endpoints (backend/open_webui/routers/
// auths.py's get_ldap_config/update_ldap_config both return
// {"ENABLE_LDAP": ...}, uppercase).
type LDAPConfigForm struct {
	EnableLDAP *bool `json:"ENABLE_LDAP"`
}

// ldapConfigUpdateForm is the wire shape of the POST /auths/admin/config/ldap
// request body. The backend's LdapConfigForm (backend/open_webui/routers/
// auths.py) declares a lowercase `enable_ldap` field, distinct from the
// uppercase ENABLE_LDAP used in both endpoints' responses - sending the
// response-shaped key here would be silently ignored by Pydantic and the
// backend would reset LDAP to disabled on every apply.
type ldapConfigUpdateForm struct {
	EnableLDAP *bool `json:"enable_ldap"`
}

// GetAdminConfig retrieves admin configuration.
func (c *Client) GetAdminConfig(ctx context.Context) (*AdminConfigForm, error) {
	var resp AdminConfigForm
	if err := c.do(ctx, http.MethodGet, "auths/admin/config", nil, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// SetAdminConfig updates admin configuration.
func (c *Client) SetAdminConfig(ctx context.Context, form AdminConfigForm) (*AdminConfigForm, error) {
	var resp AdminConfigForm
	if err := c.do(ctx, http.MethodPost, "auths/admin/config", nil, form, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetLDAPConfig retrieves LDAP configuration.
func (c *Client) GetLDAPConfig(ctx context.Context) (*LDAPConfigForm, error) {
	var resp LDAPConfigForm
	if err := c.do(ctx, http.MethodGet, "auths/admin/config/ldap", nil, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// SetLDAPConfig updates LDAP configuration. The request body uses the
// lowercase enable_ldap key expected by the backend's LdapConfigForm, while
// the response is decoded using the uppercase ENABLE_LDAP shape both LDAP
// config endpoints return.
func (c *Client) SetLDAPConfig(ctx context.Context, form LDAPConfigForm) (*LDAPConfigForm, error) {
	updateForm := ldapConfigUpdateForm(form)

	var resp LDAPConfigForm
	if err := c.do(ctx, http.MethodPost, "auths/admin/config/ldap", nil, updateForm, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
