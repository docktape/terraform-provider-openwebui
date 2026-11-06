package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

var _ resource.Resource = &skillResource{}
var _ resource.ResourceWithConfigure = &skillResource{}
var _ resource.ResourceWithImportState = &skillResource{}
var _ resource.ResourceWithValidateConfig = &skillResource{}
var _ resource.ResourceWithModifyPlan = &skillResource{}

// skillResource manages Open WebUI skills.
type skillResource struct {
	client *client.Client
}

// skillResourceModel captures Terraform state for skills.
type skillResourceModel struct {
	ID          types.String `tfsdk:"id"`
	SkillID     types.String `tfsdk:"skill_id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Content     types.String `tfsdk:"content"`
	IsActive    types.Bool   `tfsdk:"is_active"`
	Tags        types.List   `tfsdk:"tags"`
	ReadGroups  types.List   `tfsdk:"read_groups"`
	WriteGroups types.List   `tfsdk:"write_groups"`
	UserID      types.String `tfsdk:"user_id"`
	CreatedAt   types.Int64  `tfsdk:"created_at"`
	UpdatedAt   types.Int64  `tfsdk:"updated_at"`
	WriteAccess types.Bool   `tfsdk:"write_access"`
}

// NewSkillResource constructs a new skill resource.
func NewSkillResource() resource.Resource {
	return &skillResource{}
}

// skillIDValidator rejects skill identifiers the backend would silently
// rewrite or that cannot be addressed afterwards. On create Open WebUI
// lowercases the id and replaces spaces with hyphens, which would make the
// stored id differ from the configured one ("inconsistent result after apply");
// an empty id or one containing "/" is accepted by create but can never be read
// back or deleted through the id routes.
type skillIDValidator struct{}

func (skillIDValidator) Description(_ context.Context) string {
	return "must be non-empty, lowercase, and contain no whitespace or `/`"
}

func (v skillIDValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (skillIDValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()
	var problem string
	switch {
	case value == "":
		problem = "The skill_id must not be empty."
	case value != strings.ToLower(value):
		problem = fmt.Sprintf("Open WebUI lowercases skill ids on create, so %q would be stored as %q. Use a lowercase id.", value, strings.ToLower(value))
	case strings.IndexFunc(value, unicode.IsSpace) >= 0:
		problem = "Open WebUI rewrites spaces in skill ids to hyphens on create. Use hyphens (or underscores) instead of whitespace."
	case value == "." || value == "..":
		problem = "The skill_id must not be \".\" or \"..\": such a skill cannot be addressed by the id routes."
	case strings.Contains(value, "/"):
		problem = "The skill_id must not contain `/`: such a skill cannot be addressed by the id routes."
	}

	if problem != "" {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid skill_id", problem)
	}
}

// Metadata sets the resource type name.
func (r *skillResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_skill"
}

// Schema defines the skill resource schema.
func (r *skillResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Skill in Open WebUI: a reusable instructional object exposed to language models (admins and users with the `workspace.skills` permission can create them), structurally similar to `openwebui_tool` but without executable Python code or valves. Skill names must be unique across the instance.\n\n" +
			"Updates replace every field of the skill: removing `description`, `is_active` or `tags` from the configuration clears the description, re-enables the skill and empties the tags. To revoke group access set `read_groups` / `write_groups` to `[]` explicitly; omitting them keeps the current grants. Every group in `write_groups` must also be listed in `read_groups`, because Open WebUI does not treat write access as read access.\n\n" +
			"Only group grants are managed. Public (`*`) and per-user grants added in the Open WebUI interface are dropped on the next update. A non-admin token without the `access_grants.allow_groups` permission has group grants silently stripped by the server.\n\n" +
			"~> **Note:** the skill `content` is read back from the skill itself, falling back to the export endpoint. A non-admin token without the `workspace.skills_export` permission cannot use that fallback and keeps the configured value instead.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Identifier of the skill as stored by Open WebUI. Always equal to `skill_id`.",
				MarkdownDescription: "Identifier of the skill as stored by Open WebUI. Always equal to `skill_id`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"skill_id": schema.StringAttribute{
				Required:            true,
				Description:         "Unique identifier for the skill, e.g. `customer-support-playbook`. Must be lowercase with no whitespace or `/`. Forces replacement.",
				MarkdownDescription: "Unique identifier for the skill, e.g. `customer-support-playbook`. Must be lowercase with no whitespace or `/`. Forces replacement.",
				Validators:          []validator.String{skillIDValidator{}},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				Description:         "Display name of the skill. Must be unique across the Open WebUI instance.",
				MarkdownDescription: "Display name of the skill. Must be unique across the Open WebUI instance.",
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"description": schema.StringAttribute{
				Optional:            true,
				Description:         "Short description shown in the Open WebUI interface.",
				MarkdownDescription: "Short description shown in the Open WebUI interface.",
			},
			"content": schema.StringAttribute{
				Required:            true,
				Description:         "Instruction text (Markdown) that the skill provides to language models.",
				MarkdownDescription: "Instruction text (Markdown) that the skill provides to language models.",
			},
			"is_active": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				Description:         "Whether the skill is enabled. Defaults to `true`.",
				MarkdownDescription: "Whether the skill is enabled. Defaults to `true`.",
			},
			"tags": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				Default:             listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
				Description:         "Tags used to categorise the skill (stored as `meta.tags`). Defaults to an empty list.",
				MarkdownDescription: "Tags used to categorise the skill (stored as `meta.tags`). Defaults to an empty list.",
			},
			"read_groups": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				Description:         accessReadGroupsDescription("skill"),
				MarkdownDescription: accessReadGroupsDescription("skill"),
				PlanModifiers:       []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			},
			"write_groups": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				Description:         accessWriteGroupsDescription("skill"),
				MarkdownDescription: accessWriteGroupsDescription("skill"),
				PlanModifiers:       []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			},
			"user_id": schema.StringAttribute{
				Computed:            true,
				Description:         "Owner user identifier. Set by Open WebUI.",
				MarkdownDescription: "Owner user identifier. Set by Open WebUI.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"created_at": schema.Int64Attribute{
				Computed:            true,
				Description:         "Unix timestamp of creation. Set by Open WebUI.",
				MarkdownDescription: "Unix timestamp of creation. Set by Open WebUI.",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.Int64Attribute{
				Computed:            true,
				Description:         "Unix timestamp of last update. Set by Open WebUI.",
				MarkdownDescription: "Unix timestamp of last update. Set by Open WebUI.",
			},
			"write_access": schema.BoolAttribute{
				Computed:            true,
				Description:         "Whether the authenticated user has write access to this skill. Read-only; set by Open WebUI.",
				MarkdownDescription: "Whether the authenticated user has write access to this skill. Read-only; set by Open WebUI.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

// Configure assigns the API client.
func (r *skillResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	if client, ok := req.ProviderData.(*client.Client); ok {
		r.client = client
	}
}

// Create provisions a skill.
func (r *skillResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing skills.")
		return
	}

	var plan skillResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	form, diags := skillFormFromPlan(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateSkill(ctx, form)
	if err != nil {
		resp.Diagnostics.AddError("Create skill failed", err.Error())
		return
	}

	access, err := r.client.GetSkill(ctx, created.ID)
	if err != nil {
		resp.Diagnostics.AddError("Read skill failed", err.Error())
		return
	}

	content, fetchDiags := skillContentFor(ctx, r.client, access)
	resp.Diagnostics.Append(fetchDiags...)

	state, stateDiags := skillResponseToModel(ctx, r.client, access, content, plan.Content)
	resp.Diagnostics.Append(stateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Read refreshes skill state.
func (r *skillResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing skills.")
		return
	}

	var state skillResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	access, err := r.client.GetSkill(ctx, state.ID.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Read skill failed", err.Error())
		return
	}

	content, fetchDiags := skillContentFor(ctx, r.client, access)
	resp.Diagnostics.Append(fetchDiags...)

	updated, diags := skillResponseToModel(ctx, r.client, access, content, state.Content)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &updated)...)
}

// Update replaces skill properties.
func (r *skillResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing skills.")
		return
	}

	var plan skillResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	form, diags := skillFormFromPlan(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpdateSkill(ctx, plan.ID.ValueString(), form)
	if err != nil {
		resp.Diagnostics.AddError("Update skill failed", err.Error())
		return
	}

	access, err := r.client.GetSkill(ctx, updated.ID)
	if err != nil {
		resp.Diagnostics.AddError("Read skill after update failed", err.Error())
		return
	}

	state, stateDiags := skillResponseToModel(ctx, r.client, access, &updated.Content, plan.Content)
	resp.Diagnostics.Append(stateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete removes a skill.
func (r *skillResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing skills.")
		return
	}

	var state skillResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteSkill(ctx, state.ID.ValueString()); err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Delete skill failed", err.Error())
		return
	}
}

// ImportState maps an import identifier onto the id and skill_id attributes.
func (r *skillResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("skill_id"), req.ID)...)
}

func skillFormFromPlan(ctx context.Context, apiClient *client.Client, plan skillResourceModel) (client.SkillForm, diag.Diagnostics) {
	var diags diag.Diagnostics

	var description *string
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		value := plan.Description.ValueString()
		description = &value
	}

	// Unknown or null tags (Terraform plans an unknown value for a computed list
	// on create) expand to nil; the client always sends a list, never null.
	tags := expandStringList(ctx, plan.Tags, path.Root("tags"), &diags)
	if tags == nil {
		tags = []string{}
	}

	// The backend defaults is_active to true.
	isActive := true
	if !plan.IsActive.IsNull() && !plan.IsActive.IsUnknown() {
		isActive = plan.IsActive.ValueBool()
	}

	readNames := expandStringList(ctx, plan.ReadGroups, path.Root("read_groups"), &diags)
	writeNames := expandStringList(ctx, plan.WriteGroups, path.Root("write_groups"), &diags)
	readIDs := resolveGroupNamesToIDs(ctx, apiClient, readNames, path.Root("read_groups"), &diags)
	writeIDs := resolveGroupNamesToIDs(ctx, apiClient, writeNames, path.Root("write_groups"), &diags)

	return client.SkillForm{
		ID:            plan.SkillID.ValueString(),
		Name:          plan.Name.ValueString(),
		Description:   description,
		Content:       plan.Content.ValueString(),
		Meta:          client.SkillMeta{Tags: tags},
		IsActive:      isActive,
		AccessControl: buildAccessControl(readIDs, writeIDs),
	}, diags
}

// skillContentFor returns the skill's content. The GET-by-id response includes
// it in practice (see client.SkillAccessResponse) but the declared response
// model does not, so the export listing is the documented fallback. A nil
// result means the content could not be determined and callers keep their
// prior value; a failing export only produces a warning.
func skillContentFor(ctx context.Context, apiClient *client.Client, access *client.SkillAccessResponse) (*string, diag.Diagnostics) {
	var diags diag.Diagnostics

	if access.Content != nil {
		return access.Content, diags
	}

	skills, err := apiClient.ExportSkills(ctx)
	if err != nil {
		diags.AddWarning("Export skills failed", err.Error())
		return nil, diags
	}

	for _, skill := range skills {
		if skill.ID == access.ID {
			content := skill.Content
			return &content, diags
		}
	}

	return nil, diags
}

func skillResponseToModel(ctx context.Context, apiClient *client.Client, access *client.SkillAccessResponse, content *string, fallbackContent types.String) (skillResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	if access == nil {
		diags.AddError("Missing skill response", "Skill details were not returned by the Open WebUI API.")
		return skillResourceModel{}, diags
	}

	readIDs := extractGroupIDsFromAccessControl(access.AccessControl, "read")
	writeIDs := extractGroupIDsFromAccessControl(access.AccessControl, "write")

	readNames, readDiags := fetchGroupNamesForIDs(ctx, apiClient, readIDs)
	diags.Append(readDiags...)
	writeNames, writeDiags := fetchGroupNamesForIDs(ctx, apiClient, writeIDs)
	diags.Append(writeDiags...)

	readList, readListDiags := flattenStringSlice(ctx, readNames)
	diags.Append(readListDiags...)
	writeList, writeListDiags := flattenStringSlice(ctx, writeNames)
	diags.Append(writeListDiags...)

	// The server stores null tags as null; flattenStringSlice maps nil to an empty
	// (non-null) list, matching the schema default.
	tagList, tagDiags := flattenStringSlice(ctx, access.Meta.Tags)
	diags.Append(tagDiags...)

	description := types.StringNull()
	if access.Description != nil {
		description = types.StringValue(*access.Description)
	}

	contentValue := types.StringNull()
	if content != nil {
		contentValue = types.StringValue(*content)
	} else if !fallbackContent.IsNull() && !fallbackContent.IsUnknown() {
		contentValue = types.StringValue(fallbackContent.ValueString())
	}

	state := skillResourceModel{
		ID:          types.StringValue(access.ID),
		SkillID:     types.StringValue(access.ID),
		Name:        types.StringValue(access.Name),
		Description: description,
		Content:     contentValue,
		IsActive:    types.BoolValue(access.IsActive),
		Tags:        tagList,
		ReadGroups:  readList,
		WriteGroups: writeList,
		UserID:      types.StringValue(access.UserID),
		CreatedAt:   types.Int64Value(access.CreatedAt),
		UpdatedAt:   types.Int64Value(access.UpdatedAt),
		WriteAccess: types.BoolNull(),
	}

	if access.WriteAccess != nil {
		state.WriteAccess = types.BoolValue(*access.WriteAccess)
	}

	return state, diags
}

// ValidateConfig requires every configured write group to be a configured read
// group (see validateAccessGroupsConfig).
func (r *skillResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	validateAccessGroupsConfig(ctx, req.Config, &resp.Diagnostics)
}

// ModifyPlan keeps the planned write_groups a subset of the planned read_groups
// (see modifyAccessGroupsPlan).
func (r *skillResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	modifyAccessGroupsPlan(ctx, req, resp)
}
