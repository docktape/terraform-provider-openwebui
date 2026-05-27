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

var _ resource.Resource = &functionResource{}
var _ resource.ResourceWithConfigure = &functionResource{}
var _ resource.ResourceWithImportState = &functionResource{}

// functionResource manages Open WebUI functions (pipe/filter/action plugins).
type functionResource struct {
	client *client.Client
}

// functionResourceModel captures Terraform state for functions.
type functionResourceModel struct {
	ID           types.String `tfsdk:"id"`
	FunctionID   types.String `tfsdk:"function_id"`
	Name         types.String `tfsdk:"name"`
	Content      types.String `tfsdk:"content"`
	Description  types.String `tfsdk:"description"`
	IsActive     types.Bool   `tfsdk:"is_active"`
	IsGlobal     types.Bool   `tfsdk:"is_global"`
	Type         types.String `tfsdk:"type"`
	ManifestJSON types.String `tfsdk:"manifest_json"`
	UserID       types.String `tfsdk:"user_id"`
	CreatedAt    types.Int64  `tfsdk:"created_at"`
	UpdatedAt    types.Int64  `tfsdk:"updated_at"`
}

// NewFunctionResource constructs a new function resource.
func NewFunctionResource() resource.Resource {
	return &functionResource{}
}

// Metadata sets the resource type name.
func (r *functionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_function"
}

// Schema defines the function resource schema.
func (r *functionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Identifier of the function (mirrors function_id).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"function_id": schema.StringAttribute{
				Required:    true,
				Description: "Function identifier. Must be a valid identifier (letters, digits, underscores); lowercased by Open WebUI. Changing it forces replacement.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Display name for the function.",
			},
			"content": schema.StringAttribute{
				Required:    true,
				Description: "Python source for the function. The defined class (Pipe/Filter/Action) determines the function type.",
			},
			"description": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Human-readable function description (meta.description).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"is_active": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Whether the function is enabled. Defaults to false (Open WebUI default).",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"is_global": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Whether the function is applied globally to every chat. Defaults to false (Open WebUI default).",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"type": schema.StringAttribute{
				Computed:    true,
				Description: "Function type derived from the source: pipe, filter, or action.",
			},
			"manifest_json": schema.StringAttribute{
				Computed:    true,
				Description: "JSON manifest derived from the source frontmatter.",
			},
			"user_id": schema.StringAttribute{
				Computed:    true,
				Description: "Identifier of the user who owns the function.",
			},
			"created_at": schema.Int64Attribute{
				Computed:      true,
				Description:   "Unix timestamp of function creation.",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.Int64Attribute{
				Computed:    true,
				Description: "Unix timestamp of the last function update.",
			},
		},
	}
}

// Configure assigns the API client.
func (r *functionResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	if c, ok := req.ProviderData.(*client.Client); ok {
		r.client = c
	}
}

// Create provisions a function and converges its activation flags.
func (r *functionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing functions.")
		return
	}

	var plan functionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateFunction(ctx, functionFormFromPlan(plan))
	if err != nil {
		resp.Diagnostics.AddError("Create function failed", err.Error())
		return
	}

	if diags := r.convergeToggles(ctx, created.ID, created.IsActive, created.IsGlobal, plan); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	fn, err := r.client.GetFunction(ctx, created.ID)
	if err != nil {
		resp.Diagnostics.AddError("Read function failed", err.Error())
		return
	}

	state, diags := functionModelToState(fn)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Content = plan.Content // content is config-authoritative

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Read refreshes function state.
func (r *functionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing functions.")
		return
	}

	var state functionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	fn, err := r.client.GetFunction(ctx, state.ID.ValueString())
	if err != nil {
		if err == client.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Read function failed", err.Error())
		return
	}

	updated, diags := functionModelToState(fn)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Preserve configured content to avoid diffs from server-side replace_imports.
	if !state.Content.IsNull() {
		updated.Content = state.Content
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &updated)...)
}

// Update mutates function properties and converges activation flags.
func (r *functionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing functions.")
		return
	}

	var plan functionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpdateFunction(ctx, plan.ID.ValueString(), functionFormFromPlan(plan))
	if err != nil {
		resp.Diagnostics.AddError("Update function failed", err.Error())
		return
	}

	if diags := r.convergeToggles(ctx, updated.ID, updated.IsActive, updated.IsGlobal, plan); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	fn, err := r.client.GetFunction(ctx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read function failed", err.Error())
		return
	}

	state, diags := functionModelToState(fn)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Content = plan.Content

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete removes a function.
func (r *functionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing functions.")
		return
	}

	var state functionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteFunction(ctx, state.ID.ValueString()); err != nil {
		if err == client.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Delete function failed", err.Error())
		return
	}
}

// ImportState maps an import identifier onto the id and function_id attributes.
func (r *functionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("function_id"), req.ID)...)
}

// convergeToggles flips is_active / is_global to match the plan, given the
// function's current flags. Omitted (null/unknown) plan values keep the current value.
func (r *functionResource) convergeToggles(ctx context.Context, id string, currentActive, currentGlobal bool, plan functionResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	desiredActive := currentActive
	if !plan.IsActive.IsNull() && !plan.IsActive.IsUnknown() {
		desiredActive = plan.IsActive.ValueBool()
	}
	desiredGlobal := currentGlobal
	if !plan.IsGlobal.IsNull() && !plan.IsGlobal.IsUnknown() {
		desiredGlobal = plan.IsGlobal.ValueBool()
	}

	toggleActive, toggleGlobal := functionTogglesNeeded(currentActive, currentGlobal, desiredActive, desiredGlobal)
	if toggleActive {
		if _, err := r.client.ToggleFunction(ctx, id); err != nil {
			diags.AddError("Toggle function is_active failed", err.Error())
			return diags
		}
	}
	if toggleGlobal {
		if _, err := r.client.ToggleFunctionGlobal(ctx, id); err != nil {
			diags.AddError("Toggle function is_global failed", err.Error())
			return diags
		}
	}
	return diags
}

func functionFormFromPlan(plan functionResourceModel) client.FunctionForm {
	var description *string
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		value := plan.Description.ValueString()
		description = &value
	}
	return client.FunctionForm{
		ID:      plan.FunctionID.ValueString(),
		Name:    plan.Name.ValueString(),
		Content: plan.Content.ValueString(),
		Meta:    client.FunctionMeta{Description: description},
	}
}

func functionModelToState(fn *client.FunctionModel) (functionResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	manifestJSON, err := encodeOptionalJSON(fn.Meta.Manifest)
	if err != nil {
		diags.AddError("Serialize manifest", err.Error())
	}

	description := types.StringNull()
	if fn.Meta.Description != nil {
		description = types.StringValue(*fn.Meta.Description)
	}

	contentValue := types.StringNull()
	if fn.Content != "" {
		contentValue = types.StringValue(fn.Content)
	}

	return functionResourceModel{
		ID:           types.StringValue(fn.ID),
		FunctionID:   types.StringValue(fn.ID),
		Name:         types.StringValue(fn.Name),
		Content:      contentValue,
		Description:  description,
		IsActive:     types.BoolValue(fn.IsActive),
		IsGlobal:     types.BoolValue(fn.IsGlobal),
		Type:         types.StringValue(fn.Type),
		ManifestJSON: manifestJSON,
		UserID:       types.StringValue(fn.UserID),
		CreatedAt:    types.Int64Value(fn.CreatedAt),
		UpdatedAt:    types.Int64Value(fn.UpdatedAt),
	}, diags
}

// functionTogglesNeeded reports whether the is_active / is_global toggle
// endpoints must be called to move a function from its current state to the
// desired state. The toggle endpoints flip the value rather than set it.
func functionTogglesNeeded(currentActive, currentGlobal, desiredActive, desiredGlobal bool) (toggleActive, toggleGlobal bool) {
	return currentActive != desiredActive, currentGlobal != desiredGlobal
}
