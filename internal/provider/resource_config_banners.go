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

var _ resource.Resource = &bannersConfigResource{}
var _ resource.ResourceWithConfigure = &bannersConfigResource{}
var _ resource.ResourceWithImportState = &bannersConfigResource{}

// bannersConfigResource manages banner configuration.
type bannersConfigResource struct {
	client *client.Client
}

type bannerItemModel struct {
	ID          types.String `tfsdk:"id"`
	Type        types.String `tfsdk:"type"`
	Title       types.String `tfsdk:"title"`
	Content     types.String `tfsdk:"content"`
	Dismissible types.Bool   `tfsdk:"dismissible"`
	Timestamp   types.Int64  `tfsdk:"timestamp"`
}

type bannersConfigModel struct {
	ID      types.String      `tfsdk:"id"`
	Banners []bannerItemModel `tfsdk:"banners"`
}

// NewBannersConfigResource constructs a new banners config resource.
func NewBannersConfigResource() resource.Resource {
	return &bannersConfigResource{}
}

// Metadata sets the resource type name.
func (r *bannersConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_banners_config"
}

// Schema defines the banners config schema.
func (r *bannersConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages UI announcement banners displayed to all Open WebUI users.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Singleton identifier. Set by Open WebUI.",
				MarkdownDescription: "Singleton identifier. Set by Open WebUI.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"banners": schema.ListNestedAttribute{
				Required:            true,
				Description:         "List of banners to display.",
				MarkdownDescription: "List of banners to display.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Required:            true,
							Description:         "Unique identifier for this banner.",
							MarkdownDescription: "Unique identifier for this banner.",
						},
						"type": schema.StringAttribute{
							Required:            true,
							Description:         "Visual style of the banner: `info`, `warning`, or `error`.",
							MarkdownDescription: "Visual style of the banner: `info`, `warning`, or `error`.",
						},
						"title": schema.StringAttribute{
							Optional:            true,
							Description:         "Short headline shown in the banner.",
							MarkdownDescription: "Short headline shown in the banner.",
						},
						"content": schema.StringAttribute{
							Required:            true,
							Description:         "Full body text of the banner.",
							MarkdownDescription: "Full body text of the banner.",
						},
						"dismissible": schema.BoolAttribute{
							Required:            true,
							Description:         "Whether users can dismiss the banner.",
							MarkdownDescription: "Whether users can dismiss the banner.",
							PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
						},
						"timestamp": schema.Int64Attribute{
							Required:            true,
							Description:         "Unix timestamp used for ordering banners.",
							MarkdownDescription: "Unix timestamp used for ordering banners.",
							PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
						},
					},
				},
			},
		},
	}
}

// Configure assigns the API client.
func (r *bannersConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	if client, ok := req.ProviderData.(*client.Client); ok {
		r.client = client
	}
}

// Create updates the banners config.
func (r *bannersConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing banners config.")
		return
	}

	var plan bannersConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applyBannersConfig(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Read refreshes the banners config.
func (r *bannersConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing banners config.")
		return
	}

	banners, err := r.client.GetBanners(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Read banners config failed", err.Error())
		return
	}

	state := bannersConfigModel{
		ID:      types.StringValue("banners"),
		Banners: flattenBanners(banners),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update applies new banners config.
func (r *bannersConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing banners config.")
		return
	}

	var plan bannersConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applyBannersConfig(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete removes the resource from state without changing remote configuration.
func (r *bannersConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing banners config.")
		return
	}
}

// ImportState maps import identifiers onto the id attribute.
func (r *bannersConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func applyBannersConfig(ctx context.Context, apiClient *client.Client, plan bannersConfigModel) (bannersConfigModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	items := make([]client.BannerModel, 0, len(plan.Banners))
	for i, banner := range plan.Banners {
		if banner.ID.IsNull() || banner.ID.IsUnknown() || banner.ID.ValueString() == "" {
			diags.AddAttributeError(
				path.Root("banners").AtListIndex(i).AtName("id"),
				"Missing banner id",
				"Each banner must include an id.",
			)
			continue
		}
		if banner.Type.IsNull() || banner.Type.IsUnknown() || banner.Type.ValueString() == "" {
			diags.AddAttributeError(
				path.Root("banners").AtListIndex(i).AtName("type"),
				"Missing banner type",
				"Each banner must include a type.",
			)
			continue
		}

		var title *string
		if !banner.Title.IsNull() && !banner.Title.IsUnknown() {
			value := banner.Title.ValueString()
			title = &value
		}

		items = append(items, client.BannerModel{
			ID:          banner.ID.ValueString(),
			Type:        banner.Type.ValueString(),
			Title:       title,
			Content:     banner.Content.ValueString(),
			Dismissible: banner.Dismissible.ValueBool(),
			Timestamp:   banner.Timestamp.ValueInt64(),
		})
	}

	if diags.HasError() {
		return bannersConfigModel{}, diags
	}

	updated, err := apiClient.SetBanners(ctx, items)
	if err != nil {
		diags.AddError("Update banners config failed", err.Error())
		return bannersConfigModel{}, diags
	}

	state := bannersConfigModel{
		ID:      types.StringValue("banners"),
		Banners: flattenBanners(updated),
	}

	return state, diags
}

func flattenBanners(items []client.BannerModel) []bannerItemModel {
	result := make([]bannerItemModel, 0, len(items))
	for _, banner := range items {
		model := bannerItemModel{
			ID:          types.StringValue(banner.ID),
			Type:        types.StringValue(banner.Type),
			Title:       types.StringNull(),
			Content:     types.StringValue(banner.Content),
			Dismissible: types.BoolValue(banner.Dismissible),
			Timestamp:   types.Int64Value(banner.Timestamp),
		}
		if banner.Title != nil {
			model.Title = types.StringValue(*banner.Title)
		}
		result = append(result, model)
	}
	return result
}
