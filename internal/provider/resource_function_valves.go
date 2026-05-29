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

var _ resource.Resource = &functionValvesResource{}
var _ resource.ResourceWithConfigure = &functionValvesResource{}
var _ resource.ResourceWithImportState = &functionValvesResource{}

// functionValvesResource manages function valve configuration.
type functionValvesResource struct {
	client *client.Client
}

// functionValvesResourceModel maps Terraform state for function valves.
type functionValvesResourceModel struct {
	ID         types.String `tfsdk:"id"`
	FunctionID types.String `tfsdk:"function_id"`
	ValvesJSON types.String `tfsdk:"valves_json"`
	SpecJSON   types.String `tfsdk:"spec_json"`
}

// NewFunctionValvesResource constructs a new function valves resource.
func NewFunctionValvesResource() resource.Resource {
	return &functionValvesResource{}
}

// Metadata sets the resource type name.
func (r *functionValvesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_function_valves"
}

// Schema defines the function valves schema.
func (r *functionValvesResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Configures the Valves (settings) of an Open WebUI function.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Mirrors `function_id`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"function_id": schema.StringAttribute{
				Required:    true,
				Description: "Identifier of the function whose valves to configure. Forces replacement.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"valves_json": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "JSON object of valve values to apply. Keys must match the function's Valves schema. e.g. `jsonencode({ priority = 5 })`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"spec_json": schema.StringAttribute{
				Computed:    true,
				Description: "JSON schema of the function's Valves class. Read-only; returned by Open WebUI.",
			},
		},
	}
}

// Configure assigns the API client.
func (r *functionValvesResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	if c, ok := req.ProviderData.(*client.Client); ok {
		r.client = c
	}
}

// Create updates function valves when provided.
func (r *functionValvesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing function valves.")
		return
	}

	var plan functionValvesResourceModel
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
		if _, err := r.client.UpdateFunctionValves(ctx, plan.FunctionID.ValueString(), valves); err != nil {
			resp.Diagnostics.AddError("Update function valves failed", err.Error())
			return
		}
	}

	state, diags := readFunctionValvesState(ctx, r.client, plan.FunctionID.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Read refreshes function valve state.
func (r *functionValvesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing function valves.")
		return
	}

	var state functionValvesResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, diags := readFunctionValvesState(ctx, r.client, state.FunctionID.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &updated)...)
}

// Update applies new valve settings.
func (r *functionValvesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing function valves.")
		return
	}

	var plan functionValvesResourceModel
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
	if _, err := r.client.UpdateFunctionValves(ctx, plan.FunctionID.ValueString(), valves); err != nil {
		resp.Diagnostics.AddError("Update function valves failed", err.Error())
		return
	}

	state, diags := readFunctionValvesState(ctx, r.client, plan.FunctionID.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete removes the resource from state without changing remote configuration.
func (r *functionValvesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing function valves.")
		return
	}
}

// ImportState maps import identifiers to function_id.
func (r *functionValvesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("function_id"), req, resp)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func readFunctionValvesState(ctx context.Context, apiClient *client.Client, functionID string) (functionValvesResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	valves, err := apiClient.GetFunctionValves(ctx, functionID)
	if err != nil {
		if err == client.ErrNotFound {
			diags.AddError("Function not found", "Open WebUI did not return valves for the requested function.")
			return functionValvesResourceModel{}, diags
		}
		diags.AddError("Read function valves failed", err.Error())
		return functionValvesResourceModel{}, diags
	}

	spec, err := apiClient.GetFunctionValvesSpec(ctx, functionID)
	if err != nil {
		if err != client.ErrNotFound {
			diags.AddError("Read function valves spec failed", err.Error())
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

	state := functionValvesResourceModel{
		ID:         types.StringValue(functionID),
		FunctionID: types.StringValue(functionID),
		ValvesJSON: valvesJSON,
		SpecJSON:   specJSON,
	}

	return state, diags
}
