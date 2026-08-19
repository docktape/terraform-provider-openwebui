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

// arenaModelModel is the flat Terraform-facing shape of an evaluation arena
// model entry. The wire format nests profile_image_url/description/model_ids
// under a "meta" object; here they're flattened onto the entry directly, the
// same way toolServerConnectionModel flattens headers/config onto the
// connection rather than mirroring the wire shape 1:1.
type arenaModelModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	ProfileImageURL types.String `tfsdk:"profile_image_url"`
	Description     types.String `tfsdk:"description"`
	ModelIDs        types.List   `tfsdk:"model_ids"`
}

type evaluationArenaConfigModel struct {
	ID                          types.String      `tfsdk:"id"`
	EnableEvaluationArenaModels types.Bool        `tfsdk:"enable_evaluation_arena_models"`
	EvaluationArenaModels       []arenaModelModel `tfsdk:"evaluation_arena_models"`
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
			"evaluation_arena_models": schema.ListNestedAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "List of arena model entries used for evaluation arena comparison.",
				MarkdownDescription: "List of arena model entries used for evaluation arena comparison.",
				PlanModifiers:       []planmodifier.List{listplanmodifier.UseStateForUnknown()},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Required:    true,
							Description: "Unique identifier for the arena model entry.",
						},
						"name": schema.StringAttribute{
							Required:    true,
							Description: "Display name for the arena model entry.",
						},
						"profile_image_url": schema.StringAttribute{
							Optional:    true,
							Description: "Profile image URL shown for the arena model entry, e.g. `/favicon.png`.",
						},
						"description": schema.StringAttribute{
							Optional:    true,
							Description: "Description of the arena model entry.",
						},
						"model_ids": schema.ListAttribute{
							ElementType: types.StringType,
							Optional:    true,
							Description: "IDs of the real models this arena model entry draws from.",
						},
					},
				},
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

	arenaModels := expandArenaModels(ctx, plan.EvaluationArenaModels, &diags)
	if diags.HasError() {
		return evaluationArenaConfigModel{}, diags
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

	modelList, listDiags := flattenArenaModels(ctx, updated.EvaluationArenaModels)
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

	modelList, listDiags := flattenArenaModels(ctx, current.EvaluationArenaModels)
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

// expandArenaModels converts the Terraform-facing flat arena model entries
// into the client's wire-shaped ArenaModel structs, folding
// profile_image_url/description/model_ids back into a nested Meta object.
func expandArenaModels(ctx context.Context, items []arenaModelModel, diags *diag.Diagnostics) []client.ArenaModel {
	if items == nil {
		return nil
	}

	result := make([]client.ArenaModel, 0, len(items))
	for i, item := range items {
		modelIDs := expandStringList(ctx, item.ModelIDs, path.Root("evaluation_arena_models").AtListIndex(i).AtName("model_ids"), diags)

		meta := &client.ArenaModelMeta{
			ModelIDs: modelIDs,
		}
		if !item.ProfileImageURL.IsNull() && !item.ProfileImageURL.IsUnknown() {
			value := item.ProfileImageURL.ValueString()
			meta.ProfileImageURL = &value
		}
		if !item.Description.IsNull() && !item.Description.IsUnknown() {
			value := item.Description.ValueString()
			meta.Description = &value
		}

		result = append(result, client.ArenaModel{
			ID:   item.ID.ValueString(),
			Name: item.Name.ValueString(),
			Meta: meta,
		})
	}

	return result
}

// flattenArenaModels converts the client's wire-shaped ArenaModel structs
// into the flat Terraform-facing arena model entries.
func flattenArenaModels(ctx context.Context, models []client.ArenaModel) ([]arenaModelModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	result := make([]arenaModelModel, 0, len(models))
	for _, m := range models {
		var modelIDs []string
		var profileImageURL, description *string
		if m.Meta != nil {
			modelIDs = m.Meta.ModelIDs
			profileImageURL = m.Meta.ProfileImageURL
			description = m.Meta.Description
		}

		// model_ids is Optional (not Computed), so Terraform requires the
		// provider return exactly null when the config omits it. The real
		// Open WebUI backend's DEFAULT_ARENA_MODEL.meta.model_ids defaults to
		// None, which json.Unmarshal leaves as a nil []string - distinct from
		// an explicit "model_ids": [] (a non-nil, zero-length slice). Preserve
		// that distinction instead of unconditionally calling
		// flattenStringSlice, which always coerces nil into an empty list.
		modelIDList := types.ListNull(types.StringType)
		if modelIDs != nil {
			var listDiags diag.Diagnostics
			modelIDList, listDiags = flattenStringSlice(ctx, modelIDs)
			diags.Append(listDiags...)
		}

		item := arenaModelModel{
			ID:              types.StringValue(m.ID),
			Name:            types.StringValue(m.Name),
			ProfileImageURL: types.StringNull(),
			Description:     types.StringNull(),
			ModelIDs:        modelIDList,
		}
		if profileImageURL != nil {
			item.ProfileImageURL = types.StringValue(*profileImageURL)
		}
		if description != nil {
			item.Description = types.StringValue(*description)
		}

		result = append(result, item)
	}

	return result, diags
}
