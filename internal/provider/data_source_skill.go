package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

var _ datasource.DataSource = &skillDataSource{}
var _ datasource.DataSourceWithConfigure = &skillDataSource{}

// skillDataSource exposes skill details.
type skillDataSource struct {
	client *client.Client
}

// skillDataSourceModel maps data source inputs and outputs.
type skillDataSourceModel struct {
	SkillID     types.String `tfsdk:"skill_id"`
	ID          types.String `tfsdk:"id"`
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

// NewSkillDataSource constructs a new skill data source.
func NewSkillDataSource() datasource.DataSource {
	return &skillDataSource{}
}

// Metadata sets the data source type name.
func (d *skillDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_skill"
}

// Schema defines the skill data source schema.
func (d *skillDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up an existing skill by its `skill_id` or by its (instance-unique) `name`. Exactly one of the two must be set.\n\n" +
			"~> **Note:** lookup by `name` lists skills through `GET /skills/`, which, when `BYPASS_ADMIN_ACCESS_CONTROL` is disabled on the server, only returns skills the token owns or was granted access to. Lookup by `skill_id` always works for admins.",
		Attributes: map[string]schema.Attribute{
			"skill_id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Identifier of the skill to look up. Conflicts with `name`.",
				MarkdownDescription: "Identifier of the skill to look up. Conflicts with `name`.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.ExactlyOneOf(path.MatchRoot("name")),
				},
			},
			"name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Name of the skill to look up (skill names are unique). Conflicts with `skill_id`.",
				MarkdownDescription: "Name of the skill to look up (skill names are unique). Conflicts with `skill_id`.",
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Identifier of the skill as stored by Open WebUI. Equal to `skill_id`.",
				MarkdownDescription: "Identifier of the skill as stored by Open WebUI. Equal to `skill_id`.",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				Description:         "Short description of the skill.",
				MarkdownDescription: "Short description of the skill.",
			},
			"content": schema.StringAttribute{
				Computed:            true,
				Description:         "Instruction text of the skill.",
				MarkdownDescription: "Instruction text of the skill.",
			},
			"is_active": schema.BoolAttribute{
				Computed:            true,
				Description:         "Whether the skill is enabled.",
				MarkdownDescription: "Whether the skill is enabled.",
			},
			"tags": schema.ListAttribute{
				ElementType:         types.StringType,
				Computed:            true,
				Description:         "Tags assigned to the skill.",
				MarkdownDescription: "Tags assigned to the skill.",
			},
			"read_groups": schema.ListAttribute{
				ElementType:         types.StringType,
				Computed:            true,
				Description:         "Read-access group names currently applied to this skill.",
				MarkdownDescription: "Read-access group names currently applied to this skill.",
			},
			"write_groups": schema.ListAttribute{
				ElementType:         types.StringType,
				Computed:            true,
				Description:         "Write-access group names currently applied to this skill.",
				MarkdownDescription: "Write-access group names currently applied to this skill.",
			},
			"user_id": schema.StringAttribute{
				Computed:            true,
				Description:         "Owner user identifier.",
				MarkdownDescription: "Owner user identifier.",
			},
			"created_at": schema.Int64Attribute{
				Computed:            true,
				Description:         "Unix timestamp of creation.",
				MarkdownDescription: "Unix timestamp of creation.",
			},
			"updated_at": schema.Int64Attribute{
				Computed:            true,
				Description:         "Unix timestamp of last update.",
				MarkdownDescription: "Unix timestamp of last update.",
			},
			"write_access": schema.BoolAttribute{
				Computed:            true,
				Description:         "Whether the authenticated user has write access.",
				MarkdownDescription: "Whether the authenticated user has write access.",
			},
		},
	}
}

// Configure assigns the API client.
func (d *skillDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	if client, ok := req.ProviderData.(*client.Client); ok {
		d.client = client
	}
}

// Read retrieves skill details.
func (d *skillDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before using the skill data source.")
		return
	}

	var config skillDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasID := !config.SkillID.IsNull() && !config.SkillID.IsUnknown() && config.SkillID.ValueString() != ""
	hasName := !config.Name.IsNull() && !config.Name.IsUnknown() && config.Name.ValueString() != ""
	if hasID == hasName {
		resp.Diagnostics.AddError(
			"Missing skill identifier",
			"Exactly one of skill_id or name must be supplied to query an existing skill.",
		)
		return
	}

	skillID := config.SkillID.ValueString()
	lookupAttr := path.Root("skill_id")
	if hasName {
		lookupAttr = path.Root("name")
		skills, err := d.client.ListSkills(ctx)
		if err != nil {
			resp.Diagnostics.AddError("List skills failed", err.Error())
			return
		}

		name := config.Name.ValueString()
		found := false
		for _, skill := range skills {
			if skill.Name == name {
				skillID = skill.ID
				found = true
				break
			}
		}
		if !found {
			resp.Diagnostics.AddAttributeError(
				path.Root("name"),
				"Skill not found",
				fmt.Sprintf("No Open WebUI skill was found with the name %q.", name),
			)
			return
		}
	}

	access, err := d.client.GetSkill(ctx, skillID)
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.Diagnostics.AddAttributeError(
				lookupAttr,
				"Skill not found",
				fmt.Sprintf("No Open WebUI skill was found with the supplied %s.", lookupAttr),
			)
			return
		}
		resp.Diagnostics.AddError("Read skill failed", err.Error())
		return
	}

	content, fetchDiags := skillContentFor(ctx, d.client, access)
	resp.Diagnostics.Append(fetchDiags...)

	model, diags := skillResponseToModel(ctx, d.client, access, content, types.StringNull())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := skillDataSourceModel{
		SkillID:     model.SkillID,
		ID:          model.ID,
		Name:        model.Name,
		Description: model.Description,
		Content:     model.Content,
		IsActive:    model.IsActive,
		Tags:        model.Tags,
		ReadGroups:  model.ReadGroups,
		WriteGroups: model.WriteGroups,
		UserID:      model.UserID,
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
		WriteAccess: model.WriteAccess,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
