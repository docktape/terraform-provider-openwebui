package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

var _ resource.Resource = &chatConfigResource{}
var _ resource.ResourceWithConfigure = &chatConfigResource{}
var _ resource.ResourceWithImportState = &chatConfigResource{}

type chatConfigResource struct {
	client *client.Client
}

type chatConfigModel struct {
	ID                                   types.String `tfsdk:"id"`
	EnableContextCompaction              types.Bool   `tfsdk:"enable_context_compaction"`
	ContextCompactionTokenThreshold      types.Int64  `tfsdk:"context_compaction_token_threshold"`
	ContextCompactionTokenCap            types.Int64  `tfsdk:"context_compaction_token_cap"`
	ContextCompactionRetentionPercentage types.Int64  `tfsdk:"context_compaction_retention_percentage"`
	ContextCompactionPromptTemplate      types.String `tfsdk:"context_compaction_prompt_template"`
	ContextCompactionModel               types.String `tfsdk:"context_compaction_model"`
}

func NewChatConfigResource() resource.Resource {
	return &chatConfigResource{}
}

func (r *chatConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_chat_config"
}

func (r *chatConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	b := func(desc string) schema.BoolAttribute {
		return schema.BoolAttribute{Optional: true, Computed: true, Description: desc, MarkdownDescription: desc, PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()}}
	}
	i := func(desc string, validators ...validator.Int64) schema.Int64Attribute {
		return schema.Int64Attribute{Optional: true, Computed: true, Description: desc, MarkdownDescription: desc, PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}, Validators: validators}
	}
	s := func(desc string) schema.StringAttribute {
		return schema.StringAttribute{Optional: true, Computed: true, Description: desc, MarkdownDescription: desc, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}}
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "~> **Requires Open WebUI v0.11.0 or later.** The chat config endpoint first appeared in v0.10.0 with only three of this resource's six attributes; `context_compaction_model`, `context_compaction_token_cap` and `context_compaction_retention_percentage` were added in v0.11.0, and this resource reads and writes all six.\n\nManages Open WebUI's chat context-compaction settings. This is a singleton resource.\n\n" +
			"~> This resource does a full-object update: any attribute not explicitly set in your configuration will be reset to its zero value (false/0/empty string) on every apply.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Singleton identifier.",
				MarkdownDescription: "Singleton identifier.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"enable_context_compaction":               b("Whether context compaction is enabled for chat processing."),
			"context_compaction_token_threshold":      i("Minimum token count that triggers context compaction. Must be at least 1 (Open WebUI raises smaller values to 1).", int64validator.AtLeast(1)),
			"context_compaction_token_cap":            i("Maximum tokens to retain after compaction. Must be at least 1 (Open WebUI raises smaller values to 1; 0 is replaced by the token threshold).", int64validator.AtLeast(1)),
			"context_compaction_retention_percentage": i("Percentage of tokens to retain from the original context. Must be between 10 and 50 inclusive (Open WebUI clamps values outside this range).", int64validator.Between(10, 50)),
			"context_compaction_prompt_template":      s("Prompt template used for context compaction operations."),
			"context_compaction_model":                s("Model ID used when compacting/summarizing chat context, if different from the chat's regular model."),
		},
	}
}

func (r *chatConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	if c, ok := req.ProviderData.(*client.Client); ok {
		r.client = c
	}
}

func (r *chatConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing chat config.")
		return
	}

	var plan chatConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applyChatConfig(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *chatConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing chat config.")
		return
	}

	var plan chatConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applyChatConfig(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func applyChatConfig(ctx context.Context, apiClient *client.Client, plan chatConfigModel) (chatConfigModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	form := chatConfigFormFromModel(plan)
	updated, err := apiClient.SetChatConfig(ctx, form)
	if err != nil {
		diags.AddError("Update chat config failed", err.Error())
		return chatConfigModel{}, diags
	}

	state := chatConfigModelFromForm(*updated)
	return state, diags
}

func (r *chatConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing chat config.")
		return
	}

	current, err := r.client.GetChatConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Read chat config failed", err.Error())
		return
	}

	state := chatConfigModelFromForm(*current)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *chatConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing chat config.")
		return
	}
	// Singleton config: destroying the resource leaves the remote config untouched.
}

func (r *chatConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func chatConfigFormFromModel(m chatConfigModel) client.ChatConfigForm {
	return client.ChatConfigForm{
		EnableContextCompaction:              m.EnableContextCompaction.ValueBool(),
		ContextCompactionTokenThreshold:      m.ContextCompactionTokenThreshold.ValueInt64(),
		ContextCompactionTokenCap:            int64Ptr(m.ContextCompactionTokenCap),
		ContextCompactionRetentionPercentage: m.ContextCompactionRetentionPercentage.ValueInt64(),
		ContextCompactionPromptTemplate:      m.ContextCompactionPromptTemplate.ValueString(),
		ContextCompactionModel:               m.ContextCompactionModel.ValueString(),
	}
}

func chatConfigModelFromForm(f client.ChatConfigForm) chatConfigModel {
	var tokenCapValue types.Int64
	if f.ContextCompactionTokenCap != nil {
		tokenCapValue = types.Int64Value(*f.ContextCompactionTokenCap)
	} else {
		tokenCapValue = types.Int64Null()
	}
	return chatConfigModel{
		ID:                                   types.StringValue("chat"),
		EnableContextCompaction:              types.BoolValue(f.EnableContextCompaction),
		ContextCompactionTokenThreshold:      types.Int64Value(f.ContextCompactionTokenThreshold),
		ContextCompactionTokenCap:            tokenCapValue,
		ContextCompactionRetentionPercentage: types.Int64Value(f.ContextCompactionRetentionPercentage),
		ContextCompactionPromptTemplate:      types.StringValue(f.ContextCompactionPromptTemplate),
		ContextCompactionModel:               types.StringValue(f.ContextCompactionModel),
	}
}
