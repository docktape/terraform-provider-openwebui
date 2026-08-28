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

var _ resource.Resource = &tasksConfigResource{}
var _ resource.ResourceWithConfigure = &tasksConfigResource{}
var _ resource.ResourceWithImportState = &tasksConfigResource{}

type tasksConfigResource struct {
	client *client.Client
}

type tasksConfigModel struct {
	ID                                   types.String `tfsdk:"id"`
	TaskModel                            types.String `tfsdk:"task_model"`
	TaskModelExternal                    types.String `tfsdk:"task_model_external"`
	EnableTitleGeneration                types.Bool   `tfsdk:"enable_title_generation"`
	TitleGenerationPromptTemplate        types.String `tfsdk:"title_generation_prompt_template"`
	ImagePromptGenerationPromptTemplate  types.String `tfsdk:"image_prompt_generation_prompt_template"`
	EnableAutocompleteGeneration         types.Bool   `tfsdk:"enable_autocomplete_generation"`
	AutocompleteGenerationInputMaxLength types.Int64  `tfsdk:"autocomplete_generation_input_max_length"`
	AutocompleteGenerationPromptTemplate types.String `tfsdk:"autocomplete_generation_prompt_template"`
	TagsGenerationPromptTemplate         types.String `tfsdk:"tags_generation_prompt_template"`
	FollowUpGenerationPromptTemplate     types.String `tfsdk:"follow_up_generation_prompt_template"`
	EnableFollowUpGeneration             types.Bool   `tfsdk:"enable_follow_up_generation"`
	EnableTagsGeneration                 types.Bool   `tfsdk:"enable_tags_generation"`
	EnableSearchQueryGeneration          types.Bool   `tfsdk:"enable_search_query_generation"`
	EnableRetrievalQueryGeneration       types.Bool   `tfsdk:"enable_retrieval_query_generation"`
	QueryGenerationPromptTemplate        types.String `tfsdk:"query_generation_prompt_template"`
	ToolsFunctionCallingPromptTemplate   types.String `tfsdk:"tools_function_calling_prompt_template"`
	EnableVoiceModePrompt                types.Bool   `tfsdk:"enable_voice_mode_prompt"`
	VoiceModePromptTemplate              types.String `tfsdk:"voice_mode_prompt_template"`
}

func NewTasksConfigResource() resource.Resource {
	return &tasksConfigResource{}
}

func (r *tasksConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tasks_config"
}

func (r *tasksConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	str := func(desc string) schema.StringAttribute {
		return schema.StringAttribute{Optional: true, Computed: true, Description: desc, MarkdownDescription: desc, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}}
	}
	b := func(desc string) schema.BoolAttribute {
		return schema.BoolAttribute{Optional: true, Computed: true, Description: desc, MarkdownDescription: desc, PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()}}
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages Open WebUI's title/tag/autocomplete/follow-up task-model generation settings. This is a singleton resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Singleton identifier.",
				MarkdownDescription: "Singleton identifier.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"task_model":                               str("Model ID used for internal generation tasks (title, tags, etc.) when not using the chat model."),
			"task_model_external":                      str("Model ID used for internal generation tasks when the chat backend is an external API."),
			"enable_title_generation":                  b("Whether chat titles are auto-generated."),
			"title_generation_prompt_template":         str("Prompt template used to generate chat titles."),
			"image_prompt_generation_prompt_template":  str("Prompt template used to generate image-generation prompts."),
			"enable_autocomplete_generation":           b("Whether inline autocomplete suggestions are generated."),
			"autocomplete_generation_input_max_length": schema.Int64Attribute{Optional: true, Computed: true, Description: "Maximum input length considered for autocomplete generation.", MarkdownDescription: "Maximum input length considered for autocomplete generation.", PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}},
			"autocomplete_generation_prompt_template":  str("Prompt template used for autocomplete generation. Always sent on write, including as an empty string when unset. On an Open WebUI release at or before v0.9.6, the read response may omit this key entirely, which is read back as an empty string."),
			"tags_generation_prompt_template":          str("Prompt template used to generate chat tags."),
			"follow_up_generation_prompt_template":     str("Prompt template used to generate follow-up suggestions."),
			"enable_follow_up_generation":              b("Whether follow-up suggestions are generated."),
			"enable_tags_generation":                   b("Whether chat tags are auto-generated."),
			"enable_search_query_generation":           b("Whether web-search queries are auto-generated from chat context."),
			"enable_retrieval_query_generation":        b("Whether knowledge-retrieval queries are auto-generated from chat context."),
			"query_generation_prompt_template":         str("Prompt template used for search/retrieval query generation."),
			"tools_function_calling_prompt_template":   str("Prompt template used for tools function-calling generation."),
			"enable_voice_mode_prompt":                 b("Whether a dedicated voice-mode system prompt is used."),
			"voice_mode_prompt_template":               str("Prompt template used in voice mode."),
		},
	}
}

func (r *tasksConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	if c, ok := req.ProviderData.(*client.Client); ok {
		r.client = c
	}
}

func (r *tasksConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing tasks config.")
		return
	}

	var plan tasksConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applyTasksConfig(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *tasksConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing tasks config.")
		return
	}

	var plan tasksConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applyTasksConfig(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func applyTasksConfig(ctx context.Context, apiClient *client.Client, plan tasksConfigModel) (tasksConfigModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	form := tasksConfigFormFromModel(plan)
	updated, err := apiClient.SetTasksConfig(ctx, form)
	if err != nil {
		diags.AddError("Update tasks config failed", err.Error())
		return tasksConfigModel{}, diags
	}

	state := tasksConfigModelFromForm(*updated)
	return state, diags
}

func (r *tasksConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing tasks config.")
		return
	}

	current, err := r.client.GetTasksConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Read tasks config failed", err.Error())
		return
	}

	state := tasksConfigModelFromForm(*current)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *tasksConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing tasks config.")
		return
	}
	// Singleton config: destroying the resource leaves the remote config untouched.
}

func (r *tasksConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func tasksConfigFormFromModel(m tasksConfigModel) client.TaskConfigForm {
	return client.TaskConfigForm{
		TaskModel:                            stringPtr(m.TaskModel),
		TaskModelExternal:                    stringPtr(m.TaskModelExternal),
		EnableTitleGeneration:                m.EnableTitleGeneration.ValueBool(),
		TitleGenerationPromptTemplate:        m.TitleGenerationPromptTemplate.ValueString(),
		ImagePromptGenerationPromptTemplate:  m.ImagePromptGenerationPromptTemplate.ValueString(),
		EnableAutocompleteGeneration:         m.EnableAutocompleteGeneration.ValueBool(),
		AutocompleteGenerationInputMaxLength: m.AutocompleteGenerationInputMaxLength.ValueInt64(),
		AutocompleteGenerationPromptTemplate: m.AutocompleteGenerationPromptTemplate.ValueString(),
		TagsGenerationPromptTemplate:         m.TagsGenerationPromptTemplate.ValueString(),
		FollowUpGenerationPromptTemplate:     m.FollowUpGenerationPromptTemplate.ValueString(),
		EnableFollowUpGeneration:             m.EnableFollowUpGeneration.ValueBool(),
		EnableTagsGeneration:                 m.EnableTagsGeneration.ValueBool(),
		EnableSearchQueryGeneration:          m.EnableSearchQueryGeneration.ValueBool(),
		EnableRetrievalQueryGeneration:       m.EnableRetrievalQueryGeneration.ValueBool(),
		QueryGenerationPromptTemplate:        m.QueryGenerationPromptTemplate.ValueString(),
		ToolsFunctionCallingPromptTemplate:   m.ToolsFunctionCallingPromptTemplate.ValueString(),
		EnableVoiceModePrompt:                m.EnableVoiceModePrompt.ValueBool(),
		VoiceModePromptTemplate:              stringPtr(m.VoiceModePromptTemplate),
	}
}

func tasksConfigModelFromForm(f client.TaskConfigForm) tasksConfigModel {
	return tasksConfigModel{
		ID:                                   types.StringValue("tasks"),
		TaskModel:                            stringValueOrNull(f.TaskModel),
		TaskModelExternal:                    stringValueOrNull(f.TaskModelExternal),
		EnableTitleGeneration:                types.BoolValue(f.EnableTitleGeneration),
		TitleGenerationPromptTemplate:        types.StringValue(f.TitleGenerationPromptTemplate),
		ImagePromptGenerationPromptTemplate:  types.StringValue(f.ImagePromptGenerationPromptTemplate),
		EnableAutocompleteGeneration:         types.BoolValue(f.EnableAutocompleteGeneration),
		AutocompleteGenerationInputMaxLength: types.Int64Value(f.AutocompleteGenerationInputMaxLength),
		AutocompleteGenerationPromptTemplate: types.StringValue(f.AutocompleteGenerationPromptTemplate),
		TagsGenerationPromptTemplate:         types.StringValue(f.TagsGenerationPromptTemplate),
		FollowUpGenerationPromptTemplate:     types.StringValue(f.FollowUpGenerationPromptTemplate),
		EnableFollowUpGeneration:             types.BoolValue(f.EnableFollowUpGeneration),
		EnableTagsGeneration:                 types.BoolValue(f.EnableTagsGeneration),
		EnableSearchQueryGeneration:          types.BoolValue(f.EnableSearchQueryGeneration),
		EnableRetrievalQueryGeneration:       types.BoolValue(f.EnableRetrievalQueryGeneration),
		QueryGenerationPromptTemplate:        types.StringValue(f.QueryGenerationPromptTemplate),
		ToolsFunctionCallingPromptTemplate:   types.StringValue(f.ToolsFunctionCallingPromptTemplate),
		EnableVoiceModePrompt:                types.BoolValue(f.EnableVoiceModePrompt),
		VoiceModePromptTemplate:              stringValueOrNull(f.VoiceModePromptTemplate),
	}
}
