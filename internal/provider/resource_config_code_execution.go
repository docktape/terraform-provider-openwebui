package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

var _ resource.Resource = &codeExecutionConfigResource{}
var _ resource.ResourceWithConfigure = &codeExecutionConfigResource{}
var _ resource.ResourceWithImportState = &codeExecutionConfigResource{}

// codeExecutionConfigResource manages code execution settings.
type codeExecutionConfigResource struct {
	client *client.Client
}

type codeExecutionConfigModel struct {
	ID                                 types.String `tfsdk:"id"`
	EnableCodeExecution                types.Bool   `tfsdk:"enable_code_execution"`
	CodeExecutionEngine                types.String `tfsdk:"code_execution_engine"`
	CodeExecutionJupyterURL            types.String `tfsdk:"code_execution_jupyter_url"`
	CodeExecutionJupyterAuth           types.String `tfsdk:"code_execution_jupyter_auth"`
	CodeExecutionJupyterAuthToken      types.String `tfsdk:"code_execution_jupyter_auth_token"`
	CodeExecutionJupyterAuthPassword   types.String `tfsdk:"code_execution_jupyter_auth_password"`
	CodeExecutionJupyterTimeout        types.Int64  `tfsdk:"code_execution_jupyter_timeout"`
	EnableCodeInterpreter              types.Bool   `tfsdk:"enable_code_interpreter"`
	CodeInterpreterEngine              types.String `tfsdk:"code_interpreter_engine"`
	CodeInterpreterPromptTemplate      types.String `tfsdk:"code_interpreter_prompt_template"`
	CodeInterpreterJupyterURL          types.String `tfsdk:"code_interpreter_jupyter_url"`
	CodeInterpreterJupyterAuth         types.String `tfsdk:"code_interpreter_jupyter_auth"`
	CodeInterpreterJupyterAuthToken    types.String `tfsdk:"code_interpreter_jupyter_auth_token"`
	CodeInterpreterJupyterAuthPassword types.String `tfsdk:"code_interpreter_jupyter_auth_password"`
	CodeInterpreterJupyterTimeout      types.Int64  `tfsdk:"code_interpreter_jupyter_timeout"`
}

// NewCodeExecutionConfigResource constructs a new code execution config resource.
func NewCodeExecutionConfigResource() resource.Resource {
	return &codeExecutionConfigResource{}
}

// Metadata sets the resource type name.
func (r *codeExecutionConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_code_execution_config"
}

// Schema defines the code execution config schema.
func (r *codeExecutionConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages code execution and code interpreter settings for Open WebUI.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Singleton identifier. Set by Open WebUI.",
				MarkdownDescription: "Singleton identifier. Set by Open WebUI.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"enable_code_execution": schema.BoolAttribute{
				Required:            true,
				Description:         "Whether code execution via Jupyter is enabled.",
				MarkdownDescription: "Whether code execution via Jupyter is enabled.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"code_execution_engine": schema.StringAttribute{
				Required:            true,
				Description:         "Code execution engine type, e.g. `jupyter`.",
				MarkdownDescription: "Code execution engine type, e.g. `jupyter`.",
			},
			"code_execution_jupyter_url": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "URL of the Jupyter server used for code execution.",
				MarkdownDescription: "URL of the Jupyter server used for code execution.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"code_execution_jupyter_auth": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Authentication type for the Jupyter server, e.g. `token` or `password`.",
				MarkdownDescription: "Authentication type for the Jupyter server, e.g. `token` or `password`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"code_execution_jupyter_auth_token": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
				Description:         "Auth token for the Jupyter server. Sensitive.",
				MarkdownDescription: "Auth token for the Jupyter server. Sensitive.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"code_execution_jupyter_auth_password": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
				Description:         "Password for the Jupyter server. Sensitive.",
				MarkdownDescription: "Password for the Jupyter server. Sensitive.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"code_execution_jupyter_timeout": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Description:         "Request timeout in seconds for the Jupyter server.",
				MarkdownDescription: "Request timeout in seconds for the Jupyter server.",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"enable_code_interpreter": schema.BoolAttribute{
				Required:            true,
				Description:         "Whether the AI code interpreter feature is enabled.",
				MarkdownDescription: "Whether the AI code interpreter feature is enabled.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"code_interpreter_engine": schema.StringAttribute{
				Required:            true,
				Description:         "Code interpreter engine type.",
				MarkdownDescription: "Code interpreter engine type.",
			},
			"code_interpreter_prompt_template": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "System prompt template used with the code interpreter.",
				MarkdownDescription: "System prompt template used with the code interpreter.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"code_interpreter_jupyter_url": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "URL of the Jupyter server used for the code interpreter.",
				MarkdownDescription: "URL of the Jupyter server used for the code interpreter.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"code_interpreter_jupyter_auth": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Authentication type for the code interpreter Jupyter server.",
				MarkdownDescription: "Authentication type for the code interpreter Jupyter server.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"code_interpreter_jupyter_auth_token": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
				Description:         "Auth token for the code interpreter Jupyter server. Sensitive.",
				MarkdownDescription: "Auth token for the code interpreter Jupyter server. Sensitive.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"code_interpreter_jupyter_auth_password": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
				Description:         "Password for the code interpreter Jupyter server. Sensitive.",
				MarkdownDescription: "Password for the code interpreter Jupyter server. Sensitive.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"code_interpreter_jupyter_timeout": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Description:         "Request timeout in seconds for the code interpreter Jupyter server.",
				MarkdownDescription: "Request timeout in seconds for the code interpreter Jupyter server.",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
		},
	}
}

// Configure assigns the API client.
func (r *codeExecutionConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	if client, ok := req.ProviderData.(*client.Client); ok {
		r.client = client
	}
}

// Create updates the code execution config.
func (r *codeExecutionConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing code execution config.")
		return
	}

	var plan codeExecutionConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applyCodeExecutionConfig(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Read refreshes the code execution config.
func (r *codeExecutionConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing code execution config.")
		return
	}

	config, err := r.client.GetCodeExecutionConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Read code execution config failed", err.Error())
		return
	}

	state := codeExecutionConfigModel{
		ID:                                 types.StringValue("code_execution"),
		EnableCodeExecution:                types.BoolValue(config.EnableCodeExecution),
		CodeExecutionEngine:                types.StringValue(config.CodeExecutionEngine),
		CodeExecutionJupyterURL:            stringValueOrNull(config.CodeExecutionJupyterURL),
		CodeExecutionJupyterAuth:           stringValueOrNull(config.CodeExecutionJupyterAuth),
		CodeExecutionJupyterAuthToken:      stringValueOrNull(config.CodeExecutionJupyterAuthToken),
		CodeExecutionJupyterAuthPassword:   stringValueOrNull(config.CodeExecutionJupyterAuthPassword),
		CodeExecutionJupyterTimeout:        int64ValueOrNull(config.CodeExecutionJupyterTimeout),
		EnableCodeInterpreter:              types.BoolValue(config.EnableCodeInterpreter),
		CodeInterpreterEngine:              types.StringValue(config.CodeInterpreterEngine),
		CodeInterpreterPromptTemplate:      stringValueOrNull(config.CodeInterpreterPromptTemplate),
		CodeInterpreterJupyterURL:          stringValueOrNull(config.CodeInterpreterJupyterURL),
		CodeInterpreterJupyterAuth:         stringValueOrNull(config.CodeInterpreterJupyterAuth),
		CodeInterpreterJupyterAuthToken:    stringValueOrNull(config.CodeInterpreterJupyterAuthToken),
		CodeInterpreterJupyterAuthPassword: stringValueOrNull(config.CodeInterpreterJupyterAuthPassword),
		CodeInterpreterJupyterTimeout:      int64ValueOrNull(config.CodeInterpreterJupyterTimeout),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update applies new code execution config.
func (r *codeExecutionConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing code execution config.")
		return
	}

	var plan codeExecutionConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applyCodeExecutionConfig(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete removes the resource from state without changing remote configuration.
func (r *codeExecutionConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing code execution config.")
		return
	}
}

// ImportState maps import identifiers onto the id attribute.
func (r *codeExecutionConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func applyCodeExecutionConfig(ctx context.Context, apiClient *client.Client, plan codeExecutionConfigModel) (codeExecutionConfigModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	form := client.CodeInterpreterConfigForm{
		EnableCodeExecution:                plan.EnableCodeExecution.ValueBool(),
		CodeExecutionEngine:                plan.CodeExecutionEngine.ValueString(),
		CodeExecutionJupyterURL:            stringPtr(plan.CodeExecutionJupyterURL),
		CodeExecutionJupyterAuth:           stringPtr(plan.CodeExecutionJupyterAuth),
		CodeExecutionJupyterAuthToken:      stringPtr(plan.CodeExecutionJupyterAuthToken),
		CodeExecutionJupyterAuthPassword:   stringPtr(plan.CodeExecutionJupyterAuthPassword),
		CodeExecutionJupyterTimeout:        int64Ptr(plan.CodeExecutionJupyterTimeout),
		EnableCodeInterpreter:              plan.EnableCodeInterpreter.ValueBool(),
		CodeInterpreterEngine:              plan.CodeInterpreterEngine.ValueString(),
		CodeInterpreterPromptTemplate:      stringPtr(plan.CodeInterpreterPromptTemplate),
		CodeInterpreterJupyterURL:          stringPtr(plan.CodeInterpreterJupyterURL),
		CodeInterpreterJupyterAuth:         stringPtr(plan.CodeInterpreterJupyterAuth),
		CodeInterpreterJupyterAuthToken:    stringPtr(plan.CodeInterpreterJupyterAuthToken),
		CodeInterpreterJupyterAuthPassword: stringPtr(plan.CodeInterpreterJupyterAuthPassword),
		CodeInterpreterJupyterTimeout:      int64Ptr(plan.CodeInterpreterJupyterTimeout),
	}

	updated, err := apiClient.SetCodeExecutionConfig(ctx, form)
	if err != nil {
		diags.AddError("Update code execution config failed", err.Error())
		return codeExecutionConfigModel{}, diags
	}

	state := codeExecutionConfigModel{
		ID:                                 types.StringValue("code_execution"),
		EnableCodeExecution:                types.BoolValue(updated.EnableCodeExecution),
		CodeExecutionEngine:                types.StringValue(updated.CodeExecutionEngine),
		CodeExecutionJupyterURL:            stringValueOrNull(updated.CodeExecutionJupyterURL),
		CodeExecutionJupyterAuth:           stringValueOrNull(updated.CodeExecutionJupyterAuth),
		CodeExecutionJupyterAuthToken:      stringValueOrNull(updated.CodeExecutionJupyterAuthToken),
		CodeExecutionJupyterAuthPassword:   stringValueOrNull(updated.CodeExecutionJupyterAuthPassword),
		CodeExecutionJupyterTimeout:        int64ValueOrNull(updated.CodeExecutionJupyterTimeout),
		EnableCodeInterpreter:              types.BoolValue(updated.EnableCodeInterpreter),
		CodeInterpreterEngine:              types.StringValue(updated.CodeInterpreterEngine),
		CodeInterpreterPromptTemplate:      stringValueOrNull(updated.CodeInterpreterPromptTemplate),
		CodeInterpreterJupyterURL:          stringValueOrNull(updated.CodeInterpreterJupyterURL),
		CodeInterpreterJupyterAuth:         stringValueOrNull(updated.CodeInterpreterJupyterAuth),
		CodeInterpreterJupyterAuthToken:    stringValueOrNull(updated.CodeInterpreterJupyterAuthToken),
		CodeInterpreterJupyterAuthPassword: stringValueOrNull(updated.CodeInterpreterJupyterAuthPassword),
		CodeInterpreterJupyterTimeout:      int64ValueOrNull(updated.CodeInterpreterJupyterTimeout),
	}

	return state, diags
}

func stringPtr(value types.String) *string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	result := value.ValueString()
	return &result
}

func int64Ptr(value types.Int64) *int64 {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	result := value.ValueInt64()
	return &result
}

func stringValueOrNull(value *string) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(*value)
}

func int64ValueOrNull(value *int64) types.Int64 {
	if value == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*value)
}
