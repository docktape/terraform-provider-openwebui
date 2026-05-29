package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

var _ resource.Resource = &promptResource{}
var _ resource.ResourceWithConfigure = &promptResource{}
var _ resource.ResourceWithImportState = &promptResource{}

// promptResource implements Terraform management for prompts.
type promptResource struct {
	client *client.Client
}

// promptResourceModel describes Terraform state.
type promptResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Command     types.String `tfsdk:"command"`
	Name        types.String `tfsdk:"name"`
	Content     types.String `tfsdk:"content"`
	ReadGroups  types.List   `tfsdk:"read_groups"`
	WriteGroups types.List   `tfsdk:"write_groups"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
	UserID      types.String `tfsdk:"user_id"`
}

// NewPromptResource returns a configured resource instance.
func NewPromptResource() resource.Resource {
	return &promptResource{}
}

// Metadata implements resource.Resource.
func (r *promptResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_prompt"
}

// Schema defines the prompt resource schema.
func (r *promptResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a reusable prompt command in Open WebUI. The `command` is the slash-command users type to invoke the prompt.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Server-assigned UUID for the prompt.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"command": schema.StringAttribute{
				Required:    true,
				Description: "Slash-command identifier, e.g. `summarize`. The leading `/` is normalised automatically.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Display name of the prompt.",
			},
			"content": schema.StringAttribute{
				Required:    true,
				Description: "Prompt template text. Use `{{variable}}` for user-fillable placeholders.",
			},
			"read_groups": schema.ListAttribute{
				ElementType:   types.StringType,
				Optional:      true,
				Computed:      true,
				Description:   "List of group names or IDs granted read access. Leave unset or empty for public access.",
				PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			},
			"write_groups": schema.ListAttribute{
				ElementType:   types.StringType,
				Optional:      true,
				Computed:      true,
				Description:   "List of group names or IDs granted write access.",
				PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			},
			"created_at": schema.StringAttribute{
				Computed:      true,
				Description:   "Creation date in `YYYY-MM-DD` format. Set by Open WebUI.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "Last-updated date in `YYYY-MM-DD` format. Set by Open WebUI.",
			},
			"user_id": schema.StringAttribute{
				Computed:    true,
				Description: "Owner user identifier. Set by Open WebUI.",
			},
		},
	}
}

// Configure stores the API client for subsequent operations.
func (r *promptResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	if client, ok := req.ProviderData.(*client.Client); ok {
		r.client = client
	}
}

// Create provisions a prompt.
func (r *promptResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing prompts.")
		return
	}

	var plan promptResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	form := client.PromptForm{
		Command: plan.Command.ValueString(),
		Name:    plan.Name.ValueString(),
		Content: plan.Content.ValueString(),
	}

	readNames := expandStringList(ctx, plan.ReadGroups, path.Root("read_groups"), &resp.Diagnostics)
	writeNames := expandStringList(ctx, plan.WriteGroups, path.Root("write_groups"), &resp.Diagnostics)
	readIDs := resolveGroupNamesToIDs(ctx, r.client, readNames, path.Root("read_groups"), &resp.Diagnostics)
	writeIDs := resolveGroupNamesToIDs(ctx, r.client, writeNames, path.Root("write_groups"), &resp.Diagnostics)

	form.AccessControl = buildAccessControl(readIDs, writeIDs)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreatePrompt(ctx, form)
	if err != nil {
		resp.Diagnostics.AddError("Create prompt failed", err.Error())
		return
	}

	state, diags := promptResponseToModel(ctx, r.client, created)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Read refreshes state from the API.
func (r *promptResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing prompts.")
		return
	}

	var state promptResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	current, err := r.client.GetPrompt(ctx, state.ID.ValueString())
	if err != nil {
		if err == client.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Read prompt failed", err.Error())
		return
	}

	updated, diags := promptResponseToModel(ctx, r.client, current)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &updated)...)
}

// Update mutates the prompt definition.
func (r *promptResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing prompts.")
		return
	}

	var state promptResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var plan promptResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	form := client.PromptForm{
		Command: plan.Command.ValueString(),
		Name:    plan.Name.ValueString(),
		Content: plan.Content.ValueString(),
	}

	readNames := expandStringList(ctx, plan.ReadGroups, path.Root("read_groups"), &resp.Diagnostics)
	writeNames := expandStringList(ctx, plan.WriteGroups, path.Root("write_groups"), &resp.Diagnostics)
	readIDs := resolveGroupNamesToIDs(ctx, r.client, readNames, path.Root("read_groups"), &resp.Diagnostics)
	writeIDs := resolveGroupNamesToIDs(ctx, r.client, writeNames, path.Root("write_groups"), &resp.Diagnostics)

	form.AccessControl = buildAccessControl(readIDs, writeIDs)
	if resp.Diagnostics.HasError() {
		return
	}

	updatedPrompt, err := r.client.UpdatePrompt(ctx, state.ID.ValueString(), form)
	if err != nil {
		resp.Diagnostics.AddError("Update prompt failed", err.Error())
		return
	}

	newState, diags := promptResponseToModel(ctx, r.client, updatedPrompt)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

// Delete removes the prompt.
func (r *promptResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing prompts.")
		return
	}

	var state promptResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeletePrompt(ctx, state.ID.ValueString()); err != nil {
		if err == client.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Delete prompt failed", err.Error())
		return
	}
}

// ImportState allows importing by the server UUID.
func (r *promptResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// promptResponseToModel maps API objects into Terraform state structures.
func promptResponseToModel(ctx context.Context, apiClient *client.Client, resp *client.PromptModel) (promptResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	readIDs := extractGroupIDsFromAccessControl(resp.AccessControl, "read")
	writeIDs := extractGroupIDsFromAccessControl(resp.AccessControl, "write")

	readNames, readDiags := fetchGroupNamesForIDs(ctx, apiClient, readIDs)
	diags.Append(readDiags...)
	writeNames, writeDiags := fetchGroupNamesForIDs(ctx, apiClient, writeIDs)
	diags.Append(writeDiags...)

	readList := types.ListNull(types.StringType)
	if len(readNames) > 0 {
		l, listDiags := types.ListValueFrom(ctx, types.StringType, readNames)
		diags.Append(listDiags...)
		if !listDiags.HasError() {
			readList = l
		}
	}

	writeList := types.ListNull(types.StringType)
	if len(writeNames) > 0 {
		l, listDiags := types.ListValueFrom(ctx, types.StringType, writeNames)
		diags.Append(listDiags...)
		if !listDiags.HasError() {
			writeList = l
		}
	}

	state := promptResourceModel{
		ID:          types.StringValue(resp.ID),
		Command:     types.StringValue(resp.Command),
		Name:        types.StringValue(resp.Name),
		Content:     types.StringValue(resp.Content),
		ReadGroups:  readList,
		WriteGroups: writeList,
		CreatedAt:   formatDateValue(resp.CreatedAt),
		UpdatedAt:   formatDateValue(resp.UpdatedAt),
		UserID:      types.StringValue(resp.UserID),
	}

	return state, diags
}
