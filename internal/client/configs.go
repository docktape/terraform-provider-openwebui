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

// LDAPServerConfigForm captures the LDAP connection/server settings served by
// GET and accepted by POST /auths/admin/config/ldap/server. The backend's
// LdapServerConfig Pydantic model (backend/open_webui/routers/auths.py) uses
// lowercase snake_case keys in both directions.
//
// Fields that have a server-side default are pointers tagged omitempty so an
// unset value is omitted from the request body and the server default applies
// (sending the Go zero value would instead blank the field, or fail the
// backend's non-empty validation). Label, Host, AppDN, AppDNPassword and
// SearchBase are required by the model and always sent.
type LDAPServerConfigForm struct {
	Label                 string  `json:"label"`
	Host                  string  `json:"host"`
	Port                  *int64  `json:"port,omitempty"`
	AttributeForMail      *string `json:"attribute_for_mail,omitempty"`
	AttributeForUsername  *string `json:"attribute_for_username,omitempty"`
	AppDN                 string  `json:"app_dn"`
	AppDNPassword         string  `json:"app_dn_password"`
	SearchBase            string  `json:"search_base"`
	SearchFilters         *string `json:"search_filters,omitempty"`
	UseTLS                *bool   `json:"use_tls,omitempty"`
	CertificatePath       *string `json:"certificate_path,omitempty"`
	ValidateCert          *bool   `json:"validate_cert,omitempty"`
	Ciphers               *string `json:"ciphers,omitempty"`
	EnableGroupManagement *bool   `json:"enable_group_management,omitempty"`
	EnableGroupCreation   *bool   `json:"enable_group_creation,omitempty"`
	AttributeForGroups    *string `json:"attribute_for_groups,omitempty"`
}

// GetLDAPServerConfig retrieves the LDAP server configuration.
func (c *Client) GetLDAPServerConfig(ctx context.Context) (*LDAPServerConfigForm, error) {
	var resp LDAPServerConfigForm
	if err := c.do(ctx, http.MethodGet, "auths/admin/config/ldap/server", nil, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// SetLDAPServerConfig replaces the LDAP server configuration. The backend
// treats the POST as a full-object write, so every field the form omits is
// reset to its server-side default.
func (c *Client) SetLDAPServerConfig(ctx context.Context, form LDAPServerConfigForm) (*LDAPServerConfigForm, error) {
	var resp LDAPServerConfigForm
	if err := c.do(ctx, http.MethodPost, "auths/admin/config/ldap/server", nil, form, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// OAuthConfigForm mirrors the OAuthConfigForm Pydantic model served by GET and
// accepted by POST /auths/admin/config/oauth (backend/open_webui/routers/
// auths.py; the endpoint exists from Open WebUI v0.10.0). Every field is
// Optional[...] = None server-side and the handler stores
// model_dump(exclude_none=True), so the POST is a genuine partial update: a
// nil (omitted) field leaves the stored value untouched and there is no way to
// send null. Clearing a string is done by sending "".
//
// Three fields are unions on the server and get custom wire handling (see
// MarshalJSON/UnmarshalJSON): OAuthTimeout and OAuthClientTimeout are int|str
// and OAuthGroupDefaultShare is bool|str. All three are exposed as *string.
// ENABLE_OAUTH was added in v0.11.0 and is absent from v0.10.x responses.
type OAuthConfigForm struct {
	EnableOAuth                   *bool   `json:"ENABLE_OAUTH,omitempty"`
	EnableOAuthSignup             *bool   `json:"ENABLE_OAUTH_SIGNUP,omitempty"`
	OAuthMergeAccountsByEmail     *bool   `json:"OAUTH_MERGE_ACCOUNTS_BY_EMAIL,omitempty"`
	OAuthAutoRedirect             *bool   `json:"OAUTH_AUTO_REDIRECT,omitempty"`
	OAuthAllowedDomains           *string `json:"OAUTH_ALLOWED_DOMAINS,omitempty"`
	OAuthBlockedGroups            *string `json:"OAUTH_BLOCKED_GROUPS,omitempty"`
	EnableOAuthRoleManagement     *bool   `json:"ENABLE_OAUTH_ROLE_MANAGEMENT,omitempty"`
	OAuthRolesClaim               *string `json:"OAUTH_ROLES_CLAIM,omitempty"`
	OAuthAdminRoles               *string `json:"OAUTH_ADMIN_ROLES,omitempty"`
	OAuthAllowedRoles             *string `json:"OAUTH_ALLOWED_ROLES,omitempty"`
	EnableOAuthGroupManagement    *bool   `json:"ENABLE_OAUTH_GROUP_MANAGEMENT,omitempty"`
	EnableOAuthGroupCreation      *bool   `json:"ENABLE_OAUTH_GROUP_CREATION,omitempty"`
	OAuthGroupClaim               *string `json:"OAUTH_GROUP_CLAIM,omitempty"`
	OAuthGroupDefaultShare        *string `json:"-"`
	OAuthProviderName             *string `json:"OAUTH_PROVIDER_NAME,omitempty"`
	OpenIDProviderURL             *string `json:"OPENID_PROVIDER_URL,omitempty"`
	OAuthClientID                 *string `json:"OAUTH_CLIENT_ID,omitempty"`
	OAuthClientSecret             *string `json:"OAUTH_CLIENT_SECRET,omitempty"`
	OpenIDRedirectURI             *string `json:"OPENID_REDIRECT_URI,omitempty"`
	OAuthScopes                   *string `json:"OAUTH_SCOPES,omitempty"`
	OAuthCodeChallengeMethod      *string `json:"OAUTH_CODE_CHALLENGE_METHOD,omitempty"`
	OAuthTokenEndpointAuthMethod  *string `json:"OAUTH_TOKEN_ENDPOINT_AUTH_METHOD,omitempty"`
	OpenIDEndSessionEndpoint      *string `json:"OPENID_END_SESSION_ENDPOINT,omitempty"`
	OAuthTimeout                  *string `json:"-"`
	OAuthClientTimeout            *string `json:"-"`
	OAuthEmailClaim               *string `json:"OAUTH_EMAIL_CLAIM,omitempty"`
	OAuthUsernameClaim            *string `json:"OAUTH_USERNAME_CLAIM,omitempty"`
	OAuthPictureClaim             *string `json:"OAUTH_PICTURE_CLAIM,omitempty"`
	OAuthSubClaim                 *string `json:"OAUTH_SUB_CLAIM,omitempty"`
	OAuthAudience                 *string `json:"OAUTH_AUDIENCE,omitempty"`
	OAuthUpdateEmailOnLogin       *bool   `json:"OAUTH_UPDATE_EMAIL_ON_LOGIN,omitempty"`
	OAuthUpdateNameOnLogin        *bool   `json:"OAUTH_UPDATE_NAME_ON_LOGIN,omitempty"`
	OAuthUpdatePictureOnLogin     *bool   `json:"OAUTH_UPDATE_PICTURE_ON_LOGIN,omitempty"`
	OAuthRefreshTokenIncludeScope *bool   `json:"OAUTH_REFRESH_TOKEN_INCLUDE_SCOPE,omitempty"`
}

// MarshalJSON encodes the union fields: numeric timeout strings become JSON
// numbers (the empty string stays "", which the backend stores as "no
// timeout"), and "true"/"false" share values become JSON booleans, because a
// JSON string "true" would be stored as that string by Pydantic's bool|str
// smart-mode union rather than as a boolean.
func (f OAuthConfigForm) MarshalJSON() ([]byte, error) {
	type alias OAuthConfigForm
	aux := struct {
		alias
		GroupDefaultShare any `json:"OAUTH_GROUP_DEFAULT_SHARE,omitempty"`
		Timeout           any `json:"OAUTH_TIMEOUT,omitempty"`
		ClientTimeout     any `json:"OAUTH_CLIENT_TIMEOUT,omitempty"`
	}{alias: alias(f)}

	if f.OAuthGroupDefaultShare != nil {
		switch *f.OAuthGroupDefaultShare {
		case "true":
			aux.GroupDefaultShare = true
		case "false":
			// A boolean false is a real value, but omitempty drops a nil
			// interface only; false inside an interface is kept.
			aux.GroupDefaultShare = false
		default:
			aux.GroupDefaultShare = *f.OAuthGroupDefaultShare
		}
	}
	aux.Timeout = encodeIntOrString(f.OAuthTimeout)
	aux.ClientTimeout = encodeIntOrString(f.OAuthClientTimeout)
	return json.Marshal(aux)
}

// encodeIntOrString returns a JSON number for a numeric string, the string
// itself otherwise, and nil (omitted) for a nil pointer.
func encodeIntOrString(v *string) any {
	if v == nil {
		return nil
	}
	if n, err := strconv.ParseInt(*v, 10, 64); err == nil {
		return n
	}
	return *v
}

// UnmarshalJSON decodes the union fields into strings: JSON numbers and
// booleans are rendered in their canonical text form, JSON strings pass
// through, and null/absent stay nil.
func (f *OAuthConfigForm) UnmarshalJSON(data []byte) error {
	type alias OAuthConfigForm
	aux := struct {
		*alias
		GroupDefaultShare json.RawMessage `json:"OAUTH_GROUP_DEFAULT_SHARE"`
		Timeout           json.RawMessage `json:"OAUTH_TIMEOUT"`
		ClientTimeout     json.RawMessage `json:"OAUTH_CLIENT_TIMEOUT"`
	}{alias: (*alias)(f)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	var err error
	if f.OAuthGroupDefaultShare, err = decodeScalarAsString(aux.GroupDefaultShare); err != nil {
		return fmt.Errorf("OAUTH_GROUP_DEFAULT_SHARE: %w", err)
	}
	if f.OAuthTimeout, err = decodeScalarAsString(aux.Timeout); err != nil {
		return fmt.Errorf("OAUTH_TIMEOUT: %w", err)
	}
	if f.OAuthClientTimeout, err = decodeScalarAsString(aux.ClientTimeout); err != nil {
		return fmt.Errorf("OAUTH_CLIENT_TIMEOUT: %w", err)
	}
	return nil
}

// decodeScalarAsString decodes a JSON string, number or boolean as its text
// form; null or an absent key yields nil.
func decodeScalarAsString(raw json.RawMessage) (*string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return &s, nil
	}
	var b bool
	if err := json.Unmarshal(raw, &b); err == nil {
		s = strconv.FormatBool(b)
		return &s, nil
	}
	var n json.Number
	if err := json.Unmarshal(raw, &n); err == nil {
		s = n.String()
		return &s, nil
	}
	return nil, fmt.Errorf("expected a JSON string, number or boolean, got %s", string(raw))
}

// GetOAuthConfig retrieves the OIDC/OAuth configuration (Open WebUI v0.10.0+).
// OAUTH_CLIENT_SECRET is returned in plain text.
func (c *Client) GetOAuthConfig(ctx context.Context) (*OAuthConfigForm, error) {
	var resp OAuthConfigForm
	if err := c.do(ctx, http.MethodGet, "auths/admin/config/oauth", nil, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// SetOAuthConfig applies a partial update: only the non-nil fields of form are
// sent and every other stored value is left untouched by the server. The
// response is the full resulting configuration.
func (c *Client) SetOAuthConfig(ctx context.Context, form OAuthConfigForm) (*OAuthConfigForm, error) {
	var resp OAuthConfigForm
	if err := c.do(ctx, http.MethodPost, "auths/admin/config/oauth", nil, form, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ImagesConfigForm mirrors the ImagesConfig Pydantic model served by GET
// /images/config and accepted by POST /images/config/update
// (backend/open_webui/routers/images.py; identical shape from v0.9.6 through
// v0.11.0). Every one of the 33 fields is required by the model (none has a
// default), so the POST is a full-object write: the body must carry every key
// and nothing is omitted. Callers that only want to change some settings must
// read the current config first and send it back with their changes.
//
// Nullable fields (str | None, int | None) are pointers sent as explicit null
// when nil. IMAGE_SIZE and IMAGE_STEPS are nullable in the model, but the
// handler compares them with re.match / "< 0" before storing, so null makes it
// fail; send a value. The two dict | str | None params fields and the
// list[dict] node lists are kept as raw JSON so whatever the server stores
// (object, legacy JSON string, null) round-trips unchanged.
// AUTOMATIC1111_API_AUTH is also dict | str | None in the model, but the
// backend only ever uses it as a "user:password" string, so it is a *string
// here (an unexpected dict is decoded as its raw JSON text).
type ImagesConfigForm struct {
	EnableImageGeneration          bool            `json:"ENABLE_IMAGE_GENERATION"`
	EnableImagePromptGeneration    bool            `json:"ENABLE_IMAGE_PROMPT_GENERATION"`
	ImageGenerationEngine          string          `json:"IMAGE_GENERATION_ENGINE"`
	ImageGenerationModel           string          `json:"IMAGE_GENERATION_MODEL"`
	ImageSize                      *string         `json:"IMAGE_SIZE"`
	ImageSteps                     *int64          `json:"IMAGE_STEPS"`
	ImagesOpenAIAPIBaseURL         string          `json:"IMAGES_OPENAI_API_BASE_URL"`
	ImagesOpenAIAPIKey             string          `json:"IMAGES_OPENAI_API_KEY"`
	ImagesOpenAIAPIVersion         string          `json:"IMAGES_OPENAI_API_VERSION"`
	ImagesOpenAIAPIParams          json.RawMessage `json:"IMAGES_OPENAI_API_PARAMS"`
	Automatic1111BaseURL           string          `json:"AUTOMATIC1111_BASE_URL"`
	Automatic1111APIAuth           *string         `json:"AUTOMATIC1111_API_AUTH"`
	Automatic1111Params            json.RawMessage `json:"AUTOMATIC1111_PARAMS"`
	ComfyUIBaseURL                 string          `json:"COMFYUI_BASE_URL"`
	ComfyUIAPIKey                  string          `json:"COMFYUI_API_KEY"`
	ComfyUIWorkflow                string          `json:"COMFYUI_WORKFLOW"`
	ComfyUIWorkflowNodes           json.RawMessage `json:"COMFYUI_WORKFLOW_NODES"`
	ImagesGeminiAPIBaseURL         string          `json:"IMAGES_GEMINI_API_BASE_URL"`
	ImagesGeminiAPIKey             string          `json:"IMAGES_GEMINI_API_KEY"`
	ImagesGeminiEndpointMethod     string          `json:"IMAGES_GEMINI_ENDPOINT_METHOD"`
	EnableImageEdit                bool            `json:"ENABLE_IMAGE_EDIT"`
	ImageEditEngine                string          `json:"IMAGE_EDIT_ENGINE"`
	ImageEditModel                 string          `json:"IMAGE_EDIT_MODEL"`
	ImageEditSize                  *string         `json:"IMAGE_EDIT_SIZE"`
	ImagesEditOpenAIAPIBaseURL     string          `json:"IMAGES_EDIT_OPENAI_API_BASE_URL"`
	ImagesEditOpenAIAPIKey         string          `json:"IMAGES_EDIT_OPENAI_API_KEY"`
	ImagesEditOpenAIAPIVersion     string          `json:"IMAGES_EDIT_OPENAI_API_VERSION"`
	ImagesEditGeminiAPIBaseURL     string          `json:"IMAGES_EDIT_GEMINI_API_BASE_URL"`
	ImagesEditGeminiAPIKey         string          `json:"IMAGES_EDIT_GEMINI_API_KEY"`
	ImagesEditComfyUIBaseURL       string          `json:"IMAGES_EDIT_COMFYUI_BASE_URL"`
	ImagesEditComfyUIAPIKey        string          `json:"IMAGES_EDIT_COMFYUI_API_KEY"`
	ImagesEditComfyUIWorkflow      string          `json:"IMAGES_EDIT_COMFYUI_WORKFLOW"`
	ImagesEditComfyUIWorkflowNodes json.RawMessage `json:"IMAGES_EDIT_COMFYUI_WORKFLOW_NODES"`
}

// MarshalJSON sends every key. Nil raw node lists become [] because the
// list[dict] fields are not nullable; nil raw params become null.
func (f ImagesConfigForm) MarshalJSON() ([]byte, error) {
	type alias ImagesConfigForm
	a := alias(f)
	emptyList := json.RawMessage(`[]`)
	null := json.RawMessage(`null`)
	if len(a.ComfyUIWorkflowNodes) == 0 {
		a.ComfyUIWorkflowNodes = emptyList
	}
	if len(a.ImagesEditComfyUIWorkflowNodes) == 0 {
		a.ImagesEditComfyUIWorkflowNodes = emptyList
	}
	if len(a.ImagesOpenAIAPIParams) == 0 {
		a.ImagesOpenAIAPIParams = null
	}
	if len(a.Automatic1111Params) == 0 {
		a.Automatic1111Params = null
	}
	return json.Marshal(a)
}

// UnmarshalJSON decodes IMAGE_STEPS leniently (number, numeric string, or
// ""/null as nil) and AUTOMATIC1111_API_AUTH as a string (null as nil, any
// other JSON value as its raw text).
func (f *ImagesConfigForm) UnmarshalJSON(data []byte) error {
	type alias ImagesConfigForm
	aux := struct {
		*alias
		ImageSteps           json.RawMessage `json:"IMAGE_STEPS"`
		Automatic1111APIAuth json.RawMessage `json:"AUTOMATIC1111_API_AUTH"`
	}{alias: (*alias)(f)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	var err error
	if f.ImageSteps, err = decodeIntOrEmptyString(aux.ImageSteps); err != nil {
		return fmt.Errorf("IMAGE_STEPS: %w", err)
	}
	f.Automatic1111APIAuth = nil
	if raw := aux.Automatic1111APIAuth; len(raw) > 0 && string(raw) != "null" {
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			s = string(raw)
		}
		f.Automatic1111APIAuth = &s
	}
	return nil
}

// GetImagesConfig retrieves the image generation and editing configuration.
// API keys are returned in plain text.
func (c *Client) GetImagesConfig(ctx context.Context) (*ImagesConfigForm, error) {
	var resp ImagesConfigForm
	if err := c.do(ctx, http.MethodGet, "images/config", nil, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// SetImagesConfig replaces the whole image configuration (full-object write;
// see ImagesConfigForm) and returns the stored result.
func (c *Client) SetImagesConfig(ctx context.Context, form ImagesConfigForm) (*ImagesConfigForm, error) {
	var resp ImagesConfigForm
	if err := c.do(ctx, http.MethodPost, "images/config/update", nil, form, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
