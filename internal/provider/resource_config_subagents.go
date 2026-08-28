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

var _ resource.Resource = &subagentsConfigResource{}
var _ resource.ResourceWithConfigure = &subagentsConfigResource{}
var _ resource.ResourceWithImportState = &subagentsConfigResource{}

type subagentsConfigResource struct {
	client *client.Client
}

type subagentsConfigModel struct {
	ID                         types.String `tfsdk:"id"`
	EnableSubagents            types.Bool   `tfsdk:"enable_subagents"`
	SubagentsBackgroundEnabled types.Bool   `tfsdk:"subagents_background_enabled"`
	SubagentsMaxConcurrent     types.Int64  `tfsdk:"subagents_max_concurrent"`
	SubagentsMaxAsync          types.Int64  `tfsdk:"subagents_max_async"`
	SubagentsMaxIterations     types.Int64  `tfsdk:"subagents_max_iterations"`
	SubagentsMaxOutput         types.Int64  `tfsdk:"subagents_max_output"`
	SubagentsSystemPrompt      types.String `tfsdk:"subagents_system_prompt"`
}

func NewSubagentsConfigResource() resource.Resource {
	return &subagentsConfigResource{}
}

func (r *subagentsConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_subagents_config"
}

func (r *subagentsConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	b := func(desc string) schema.BoolAttribute {
		return schema.BoolAttribute{Optional: true, Computed: true, Description: desc, MarkdownDescription: desc, PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()}}
	}
	i := func(desc string) schema.Int64Attribute {
		return schema.Int64Attribute{Optional: true, Computed: true, Description: desc, MarkdownDescription: desc, PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}}
	}
	s := func(desc string) schema.StringAttribute {
		return schema.StringAttribute{Optional: true, Computed: true, Description: desc, MarkdownDescription: desc, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}}
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "~> **Requires Open WebUI v0.11.0+** — the Subagents feature and this config endpoint did not exist in earlier releases.\n\nManages Open WebUI's sub-agent orchestration settings. This is a singleton resource.\n\n" +
			"~> This resource does a full-object update: any attribute not explicitly set in your configuration will be reset to its zero value (false/0/empty string) on every apply.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Singleton identifier.",
				MarkdownDescription: "Singleton identifier.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"enable_subagents":             b("Whether sub-agents are enabled."),
			"subagents_background_enabled": b("Whether background execution for sub-agents is enabled."),
			"subagents_max_concurrent":     i("Maximum number of concurrent sub-agent tasks."),
			"subagents_max_async":          i("Maximum number of asynchronous sub-agent tasks."),
			"subagents_max_iterations":     i("Maximum number of iterations for sub-agent loops."),
			"subagents_max_output":         i("Maximum output size in characters for sub-agent results."),
			"subagents_system_prompt":      s("System prompt used to guide sub-agent behavior."),
		},
	}
}

func (r *subagentsConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	if c, ok := req.ProviderData.(*client.Client); ok {
		r.client = c
	}
}

func (r *subagentsConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing subagents config.")
		return
	}

	var plan subagentsConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applySubagentsConfig(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *subagentsConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing subagents config.")
		return
	}

	var plan subagentsConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applySubagentsConfig(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func applySubagentsConfig(ctx context.Context, apiClient *client.Client, plan subagentsConfigModel) (subagentsConfigModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	form := subagentsConfigFormFromModel(plan)
	updated, err := apiClient.SetSubagentsConfig(ctx, form)
	if err != nil {
		diags.AddError("Update subagents config failed", err.Error())
		return subagentsConfigModel{}, diags
	}

	state := subagentsConfigModelFromForm(*updated)
	return state, diags
}

func (r *subagentsConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing subagents config.")
		return
	}

	current, err := r.client.GetSubagentsConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Read subagents config failed", err.Error())
		return
	}

	state := subagentsConfigModelFromForm(*current)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *subagentsConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing subagents config.")
		return
	}
	// Singleton config: destroying the resource leaves the remote config untouched.
}

func (r *subagentsConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func subagentsConfigFormFromModel(m subagentsConfigModel) client.SubagentsConfigForm {
	return client.SubagentsConfigForm{
		EnableSubagents:            m.EnableSubagents.ValueBool(),
		SubagentsBackgroundEnabled: m.SubagentsBackgroundEnabled.ValueBool(),
		SubagentsMaxConcurrent:     m.SubagentsMaxConcurrent.ValueInt64(),
		SubagentsMaxAsync:          m.SubagentsMaxAsync.ValueInt64(),
		SubagentsMaxIterations:     m.SubagentsMaxIterations.ValueInt64(),
		SubagentsMaxOutput:         m.SubagentsMaxOutput.ValueInt64(),
		SubagentsSystemPrompt:      m.SubagentsSystemPrompt.ValueString(),
	}
}

func subagentsConfigModelFromForm(f client.SubagentsConfigForm) subagentsConfigModel {
	return subagentsConfigModel{
		ID:                         types.StringValue("subagents"),
		EnableSubagents:            types.BoolValue(f.EnableSubagents),
		SubagentsBackgroundEnabled: types.BoolValue(f.SubagentsBackgroundEnabled),
		SubagentsMaxConcurrent:     types.Int64Value(f.SubagentsMaxConcurrent),
		SubagentsMaxAsync:          types.Int64Value(f.SubagentsMaxAsync),
		SubagentsMaxIterations:     types.Int64Value(f.SubagentsMaxIterations),
		SubagentsMaxOutput:         types.Int64Value(f.SubagentsMaxOutput),
		SubagentsSystemPrompt:      types.StringValue(f.SubagentsSystemPrompt),
	}
}
