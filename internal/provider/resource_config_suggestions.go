package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

var _ resource.Resource = &suggestionsConfigResource{}
var _ resource.ResourceWithConfigure = &suggestionsConfigResource{}
var _ resource.ResourceWithImportState = &suggestionsConfigResource{}

// suggestionsConfigResource manages default suggestion configuration.
type suggestionsConfigResource struct {
	client *client.Client
}

type suggestionItemModel struct {
	Title   types.List   `tfsdk:"title"`
	Content types.String `tfsdk:"content"`
}

type suggestionsConfigModel struct {
	ID          types.String          `tfsdk:"id"`
	Suggestions []suggestionItemModel `tfsdk:"suggestions"`
}

// NewSuggestionsConfigResource constructs a new suggestions config resource.
func NewSuggestionsConfigResource() resource.Resource {
	return &suggestionsConfigResource{}
}

// Metadata sets the resource type name.
func (r *suggestionsConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_suggestions_config"
}

// Schema defines the suggestions config schema.
func (r *suggestionsConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Singleton identifier for the suggestions config.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"suggestions": schema.ListNestedAttribute{
				Required:    true,
				Description: "Default prompt suggestions.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"title":   schema.ListAttribute{ElementType: types.StringType, Required: true},
						"content": schema.StringAttribute{Required: true},
					},
				},
			},
		},
	}
}

// Configure assigns the API client.
func (r *suggestionsConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	if client, ok := req.ProviderData.(*client.Client); ok {
		r.client = client
	}
}

// Create updates the suggestions config.
func (r *suggestionsConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing suggestions config.")
		return
	}

	var plan suggestionsConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applySuggestionsConfig(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Read preserves current state (no read endpoint available).
func (r *suggestionsConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state suggestionsConfigModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update applies new suggestions config.
func (r *suggestionsConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing suggestions config.")
		return
	}

	var plan suggestionsConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applySuggestionsConfig(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete removes the resource from state without changing remote configuration.
func (r *suggestionsConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing suggestions config.")
		return
	}
}

// ImportState maps import identifiers onto the id attribute.
func (r *suggestionsConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func applySuggestionsConfig(ctx context.Context, apiClient *client.Client, plan suggestionsConfigModel) (suggestionsConfigModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	items := make([]client.PromptSuggestion, 0, len(plan.Suggestions))
	for i, suggestion := range plan.Suggestions {
		titles := expandStringList(ctx, suggestion.Title, path.Root("suggestions").AtListIndex(i).AtName("title"), &diags)
		items = append(items, client.PromptSuggestion{
			Title:   titles,
			Content: suggestion.Content.ValueString(),
		})
	}

	if diags.HasError() {
		return suggestionsConfigModel{}, diags
	}

	updated, err := apiClient.SetDefaultSuggestions(ctx, items)
	if err != nil {
		diags.AddError("Update suggestions config failed", err.Error())
		return suggestionsConfigModel{}, diags
	}

	state := suggestionsConfigModel{
		ID:          types.StringValue("suggestions"),
		Suggestions: flattenPromptSuggestions(updated),
	}

	return state, diags
}

func flattenPromptSuggestions(items []client.PromptSuggestion) []suggestionItemModel {
	result := make([]suggestionItemModel, 0, len(items))
	for _, item := range items {
		list, _ := types.ListValueFrom(context.Background(), types.StringType, item.Title)
		result = append(result, suggestionItemModel{
			Title:   list,
			Content: types.StringValue(item.Content),
		})
	}
	return result
}
