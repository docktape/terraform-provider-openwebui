package provider

import (
	"context"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

var _ resource.Resource = &pipelineValvesResource{}
var _ resource.ResourceWithConfigure = &pipelineValvesResource{}
var _ resource.ResourceWithImportState = &pipelineValvesResource{}

// pipelineValvesResource manages pipeline valve configuration.
type pipelineValvesResource struct {
	client *client.Client
}

// pipelineValvesResourceModel maps Terraform state for pipeline valves.
type pipelineValvesResourceModel struct {
	ID         types.String `tfsdk:"id"`
	PipelineID types.String `tfsdk:"pipeline_id"`
	URLIdx     types.Int64  `tfsdk:"url_idx"`
	ValvesJSON types.String `tfsdk:"valves_json"`
	SpecJSON   types.String `tfsdk:"spec_json"`
}

// NewPipelineValvesResource constructs a new pipeline valves resource.
func NewPipelineValvesResource() resource.Resource {
	return &pipelineValvesResource{}
}

// Metadata sets the resource type name.
func (r *pipelineValvesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pipeline_valves"
}

// Schema defines the pipeline valves schema.
func (r *pipelineValvesResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Configures the Valves (settings) of an Open WebUI pipeline.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Composite identifier in the form `pipeline_id:url_idx`.",
				MarkdownDescription: "Composite identifier in the form `pipeline_id:url_idx`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"pipeline_id": schema.StringAttribute{
				Required:            true,
				Description:         "Pipeline identifier to configure valves for. Forces replacement.",
				MarkdownDescription: "Pipeline identifier to configure valves for. Forces replacement.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"url_idx": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Description:         "URL index of the pipeline server. Defaults to 0.",
				MarkdownDescription: "URL index of the pipeline server. Defaults to 0.",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"valves_json": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "JSON object of valve values to apply to the pipeline. e.g. `jsonencode({ temperature = 0.7 })`.",
				MarkdownDescription: "JSON object of valve values to apply to the pipeline. e.g. `jsonencode({ temperature = 0.7 })`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"spec_json": schema.StringAttribute{
				Computed:            true,
				Description:         "JSON schema of the pipeline's Valves class. Read-only; returned by Open WebUI.",
				MarkdownDescription: "JSON schema of the pipeline's Valves class. Read-only; returned by Open WebUI.",
			},
		},
	}
}

// Configure assigns the API client.
func (r *pipelineValvesResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	if client, ok := req.ProviderData.(*client.Client); ok {
		r.client = client
	}
}

// Create updates pipeline valves when provided.
func (r *pipelineValvesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing pipeline valves.")
		return
	}

	var plan pipelineValvesResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	urlIdx := int(plan.URLIdx.ValueInt64())
	if plan.URLIdx.IsNull() || plan.URLIdx.IsUnknown() {
		urlIdx = 0
	}

	if !plan.ValvesJSON.IsNull() && !plan.ValvesJSON.IsUnknown() {
		valves := decodeOptionalJSON(plan.ValvesJSON, path.Root("valves_json"), &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		if valves == nil {
			valves = map[string]any{}
		}
		if _, err := r.client.UpdatePipelineValves(ctx, plan.PipelineID.ValueString(), urlIdx, valves); err != nil {
			resp.Diagnostics.AddError("Update pipeline valves failed", err.Error())
			return
		}
	}

	state, diags := readPipelineValvesState(ctx, r.client, plan.PipelineID.ValueString(), urlIdx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Read refreshes pipeline valve state.
func (r *pipelineValvesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing pipeline valves.")
		return
	}

	var state pipelineValvesResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	urlIdx := int(state.URLIdx.ValueInt64())
	if state.URLIdx.IsNull() || state.URLIdx.IsUnknown() {
		urlIdx = 0
	}

	updated, diags := readPipelineValvesState(ctx, r.client, state.PipelineID.ValueString(), urlIdx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &updated)...)
}

// Update applies new valve settings.
func (r *pipelineValvesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing pipeline valves.")
		return
	}

	var plan pipelineValvesResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	urlIdx := int(plan.URLIdx.ValueInt64())
	if plan.URLIdx.IsNull() || plan.URLIdx.IsUnknown() {
		urlIdx = 0
	}

	valves := decodeOptionalJSON(plan.ValvesJSON, path.Root("valves_json"), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if valves == nil {
		valves = map[string]any{}
	}
	if _, err := r.client.UpdatePipelineValves(ctx, plan.PipelineID.ValueString(), urlIdx, valves); err != nil {
		resp.Diagnostics.AddError("Update pipeline valves failed", err.Error())
		return
	}

	state, diags := readPipelineValvesState(ctx, r.client, plan.PipelineID.ValueString(), urlIdx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete removes the resource from state without changing remote configuration.
func (r *pipelineValvesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing pipeline valves.")
		return
	}
}

// ImportState maps import identifiers to pipeline_id.
func (r *pipelineValvesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")
	if len(parts) == 2 {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("pipeline_id"), parts[0])...)
		if value, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("url_idx"), value)...)
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[0])...)
		return
	}

	resource.ImportStatePassthroughID(ctx, path.Root("pipeline_id"), req, resp)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func readPipelineValvesState(ctx context.Context, apiClient *client.Client, pipelineID string, urlIdx int) (pipelineValvesResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	valves, err := apiClient.GetPipelineValves(ctx, pipelineID, urlIdx)
	if err != nil {
		if err == client.ErrNotFound {
			diags.AddError("Pipeline not found", "Open WebUI did not return valves for the requested pipeline.")
			return pipelineValvesResourceModel{}, diags
		}
		diags.AddError("Read pipeline valves failed", err.Error())
		return pipelineValvesResourceModel{}, diags
	}

	spec, err := apiClient.GetPipelineValvesSpec(ctx, pipelineID, urlIdx)
	if err != nil {
		if err != client.ErrNotFound {
			diags.AddError("Read pipeline valves spec failed", err.Error())
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

	state := pipelineValvesResourceModel{
		ID:         types.StringValue(pipelineID),
		PipelineID: types.StringValue(pipelineID),
		URLIdx:     types.Int64Value(int64(urlIdx)),
		ValvesJSON: valvesJSON,
		SpecJSON:   specJSON,
	}

	return state, diags
}
