package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

var _ resource.Resource = &evaluationArenaConfigResource{}
var _ resource.ResourceWithConfigure = &evaluationArenaConfigResource{}
var _ resource.ResourceWithImportState = &evaluationArenaConfigResource{}

type evaluationArenaConfigResource struct {
	client *client.Client
}

type evaluationArenaConfigModel struct {
	ID                          types.String `tfsdk:"id"`
	EnableEvaluationArenaModels types.Bool   `tfsdk:"enable_evaluation_arena_models"`
	EvaluationArenaModels       types.List   `tfsdk:"evaluation_arena_models"`
}

func NewEvaluationArenaConfigResource() resource.Resource {
	return &evaluationArenaConfigResource{}
}

func (r *evaluationArenaConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_evaluation_arena_config"
}

func (r *evaluationArenaConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages Open WebUI's evaluation arena configuration for model comparison.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Singleton identifier.",
				MarkdownDescription: "Singleton identifier.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"enable_evaluation_arena_models": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Whether evaluation arena model comparison is enabled.",
				MarkdownDescription: "Whether evaluation arena model comparison is enabled.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"evaluation_arena_models": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				Description:         "List of model IDs enabled for evaluation arena comparison.",
				MarkdownDescription: "List of model IDs enabled for evaluation arena comparison.",
				PlanModifiers:       []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *evaluationArenaConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	if c, ok := req.ProviderData.(*client.Client); ok {
		r.client = c
	}
}

func (r *evaluationArenaConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing evaluation arena config.")
		return
	}

	var plan evaluationArenaConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applyEvaluationArenaConfig(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *evaluationArenaConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing evaluation arena config.")
		return
	}

	var plan evaluationArenaConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applyEvaluationArenaConfig(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func applyEvaluationArenaConfig(ctx context.Context, apiClient *client.Client, plan evaluationArenaConfigModel) (evaluationArenaConfigModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	var arenaModels []string
	if !plan.EvaluationArenaModels.IsNull() && !plan.EvaluationArenaModels.IsUnknown() {
		arenaModels = expandStringList(ctx, plan.EvaluationArenaModels, path.Root("evaluation_arena_models"), &diags)
	}

	form := client.EvaluationArenaConfigForm{
		EnableEvaluationArenaModels: plan.EnableEvaluationArenaModels.ValueBool(),
		EvaluationArenaModels:       arenaModels,
	}

	updated, err := apiClient.SetEvaluationArenaConfig(ctx, form)
	if err != nil {
		diags.AddError("Update evaluation arena config failed", err.Error())
		return evaluationArenaConfigModel{}, diags
	}

	modelList, listDiags := flattenStringSlice(ctx, updated.EvaluationArenaModels)
	diags.Append(listDiags...)

	state := evaluationArenaConfigModel{
		ID:                          types.StringValue("evaluation_arena"),
		EnableEvaluationArenaModels: types.BoolValue(updated.EnableEvaluationArenaModels),
		EvaluationArenaModels:       modelList,
	}

	return state, diags
}

func (r *evaluationArenaConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing evaluation arena config.")
		return
	}

	current, err := r.client.GetEvaluationArenaConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Read evaluation arena config failed", err.Error())
		return
	}

	modelList, listDiags := flattenStringSlice(ctx, current.EvaluationArenaModels)
	resp.Diagnostics.Append(listDiags...)

	state := evaluationArenaConfigModel{
		ID:                          types.StringValue("evaluation_arena"),
		EnableEvaluationArenaModels: types.BoolValue(current.EnableEvaluationArenaModels),
		EvaluationArenaModels:       modelList,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *evaluationArenaConfigResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// Singleton config: destroying the resource leaves the remote config untouched.
}

func (r *evaluationArenaConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
