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
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Server-assigned prompt identifier (UUID).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"command": schema.StringAttribute{
				Required:    true,
				Description: "Unique command string used to invoke the prompt.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Prompt name displayed in Open WebUI.",
			},
			"content": schema.StringAttribute{
				Required:    true,
				Description: "Prompt content text.",
			},
			"read_groups": schema.ListAttribute{
				ElementType:   types.StringType,
				Optional:      true,
				Computed:      true,
				Description:   "Group names or IDs granted read access to the prompt.",
				PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			},
			"write_groups": schema.ListAttribute{
				ElementType:   types.StringType,
				Optional:      true,
				Computed:      true,
				Description:   "Group names or IDs granted write access to the prompt (also receive read access).",
				PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			},
			"created_at": schema.StringAttribute{
				Computed:      true,
				Description:   "Prompt creation date formatted as YYYY-MM-DD.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "Prompt last-update date formatted as YYYY-MM-DD.",
			},
			"user_id": schema.StringAttribute{
				Computed:    true,
				Description: "Identifier of the user who owns the prompt.",
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
