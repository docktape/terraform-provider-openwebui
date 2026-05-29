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

var _ resource.Resource = &toolValvesResource{}
var _ resource.ResourceWithConfigure = &toolValvesResource{}
var _ resource.ResourceWithImportState = &toolValvesResource{}

// toolValvesResource manages tool valve configuration.
type toolValvesResource struct {
	client *client.Client
}

// toolValvesResourceModel maps Terraform state for tool valves.
type toolValvesResourceModel struct {
	ID         types.String `tfsdk:"id"`
	ToolID     types.String `tfsdk:"tool_id"`
	ValvesJSON types.String `tfsdk:"valves_json"`
	SpecJSON   types.String `tfsdk:"spec_json"`
}

// NewToolValvesResource constructs a new tool valves resource.
func NewToolValvesResource() resource.Resource {
	return &toolValvesResource{}
}

// Metadata sets the resource type name.
func (r *toolValvesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tool_valves"
}

// Schema defines the tool valves schema.
func (r *toolValvesResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Configures the Valves (settings) of an Open WebUI tool. Valves are the tool's user-configurable parameters.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Mirrors `tool_id`.",
				MarkdownDescription: "Mirrors `tool_id`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"tool_id": schema.StringAttribute{
				Required:            true,
				Description:         "Identifier of the tool whose valves to configure. Forces replacement.",
				MarkdownDescription: "Identifier of the tool whose valves to configure. Forces replacement.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"valves_json": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "JSON object of valve values to apply. Keys must match the tool's Valves schema. e.g. `jsonencode({ max_pages = 5, timeout = 30 })`.",
				MarkdownDescription: "JSON object of valve values to apply. Keys must match the tool's Valves schema. e.g. `jsonencode({ max_pages = 5, timeout = 30 })`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"spec_json": schema.StringAttribute{
				Computed:            true,
				Description:         "JSON schema specification of the tool's Valves class. Read-only; returned by Open WebUI.",
				MarkdownDescription: "JSON schema specification of the tool's Valves class. Read-only; returned by Open WebUI.",
			},
		},
	}
}

// Configure assigns the API client.
func (r *toolValvesResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	if client, ok := req.ProviderData.(*client.Client); ok {
		r.client = client
	}
}

// Create updates tool valves when provided.
func (r *toolValvesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing tool valves.")
		return
	}

	var plan toolValvesResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !plan.ValvesJSON.IsNull() && !plan.ValvesJSON.IsUnknown() {
		valves := decodeOptionalJSON(plan.ValvesJSON, path.Root("valves_json"), &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		if valves == nil {
			valves = map[string]any{}
		}
		if _, err := r.client.UpdateToolValves(ctx, plan.ToolID.ValueString(), valves); err != nil {
			resp.Diagnostics.AddError("Update tool valves failed", err.Error())
			return
		}
	}

	state, diags := readToolValvesState(ctx, r.client, plan.ToolID.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Read refreshes tool valve state.
func (r *toolValvesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing tool valves.")
		return
	}

	var state toolValvesResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, diags := readToolValvesState(ctx, r.client, state.ToolID.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &updated)...)
}

// Update applies new valve settings.
func (r *toolValvesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing tool valves.")
		return
	}

	var plan toolValvesResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	valves := decodeOptionalJSON(plan.ValvesJSON, path.Root("valves_json"), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if valves == nil {
		valves = map[string]any{}
	}
	if _, err := r.client.UpdateToolValves(ctx, plan.ToolID.ValueString(), valves); err != nil {
		resp.Diagnostics.AddError("Update tool valves failed", err.Error())
		return
	}

	state, diags := readToolValvesState(ctx, r.client, plan.ToolID.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete removes the resource from state without changing remote configuration.
func (r *toolValvesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing tool valves.")
		return
	}
}

// ImportState maps import identifiers to tool_id.
func (r *toolValvesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("tool_id"), req, resp)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func readToolValvesState(ctx context.Context, apiClient *client.Client, toolID string) (toolValvesResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	valves, err := apiClient.GetToolValves(ctx, toolID)
	if err != nil {
		if err == client.ErrNotFound {
			diags.AddError("Tool not found", "Open WebUI did not return valves for the requested tool.")
			return toolValvesResourceModel{}, diags
		}
		diags.AddError("Read tool valves failed", err.Error())
		return toolValvesResourceModel{}, diags
	}

	spec, err := apiClient.GetToolValvesSpec(ctx, toolID)
	if err != nil {
		if err != client.ErrNotFound {
			diags.AddError("Read tool valves spec failed", err.Error())
		}
	}

	valvesJSON, err := encodeOptionalJSONValue(valves)
	if err != nil {
		diags.AddError("Serialize valves", err.Error())
	}
	specJSON, err := encodeOptionalJSONValue(spec)
	if err != nil {
		diags.AddError("Serialize valves spec", err.Error())
	}

	state := toolValvesResourceModel{
		ID:         types.StringValue(toolID),
		ToolID:     types.StringValue(toolID),
		ValvesJSON: valvesJSON,
		SpecJSON:   specJSON,
	}

	return state, diags
}
