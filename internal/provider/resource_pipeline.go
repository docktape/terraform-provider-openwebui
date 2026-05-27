package provider

import (
	"context"
	"path/filepath"
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

var _ resource.Resource = &pipelineResource{}
var _ resource.ResourceWithConfigure = &pipelineResource{}
var _ resource.ResourceWithImportState = &pipelineResource{}

// pipelineResource manages pipeline registrations.
type pipelineResource struct {
	client *client.Client
}

// pipelineResourceModel maps Terraform state for pipelines.
type pipelineResourceModel struct {
	ID          types.String `tfsdk:"id"`
	PipelineID  types.String `tfsdk:"pipeline_id"`
	URL         types.String `tfsdk:"url"`
	SourcePath  types.String `tfsdk:"source_path"`
	URLIdx      types.Int64  `tfsdk:"url_idx"`
	DetailsJSON types.String `tfsdk:"details_json"`
}

// NewPipelineResource constructs a new pipeline resource.
func NewPipelineResource() resource.Resource {
	return &pipelineResource{}
}

// Metadata sets the resource type name.
func (r *pipelineResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pipeline"
}

// Schema defines the pipeline resource schema.
func (r *pipelineResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Unique identifier assigned by Open WebUI.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"pipeline_id": schema.StringAttribute{
				Computed:    true,
				Description: "Pipeline identifier reported by the API.",
			},
			"url": schema.StringAttribute{
				Optional:      true,
				Description:   "Remote pipeline URL to register.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"source_path": schema.StringAttribute{
				Optional:      true,
				Description:   "Local pipeline bundle to upload.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"url_idx": schema.Int64Attribute{
				Optional:      true,
				Computed:      true,
				Description:   "Pipeline URL index (defaults to 0).",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"details_json": schema.StringAttribute{
				Computed:    true,
				Description: "Raw JSON describing the pipeline returned by the API.",
			},
		},
	}
}

// Configure attaches the API client.
func (r *pipelineResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	if client, ok := req.ProviderData.(*client.Client); ok {
		r.client = client
	}
}

// Create registers a pipeline.
func (r *pipelineResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing pipelines.")
		return
	}

	var plan pipelineResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	urlValue := strings.TrimSpace(plan.URL.ValueString())
	sourcePath := strings.TrimSpace(plan.SourcePath.ValueString())
	if urlValue == "" && sourcePath == "" {
		resp.Diagnostics.AddError("Missing pipeline source", "Either url or source_path must be provided to create a pipeline.")
		return
	}
	if urlValue != "" && sourcePath != "" {
		resp.Diagnostics.AddError("Conflicting pipeline sources", "Only one of url or source_path may be provided.")
		return
	}

	urlIdx := int(plan.URLIdx.ValueInt64())
	if plan.URLIdx.IsNull() || plan.URLIdx.IsUnknown() {
		urlIdx = 0
	}

	var response map[string]any
	var err error
	if sourcePath != "" {
		response, err = r.client.UploadPipeline(ctx, sourcePath, urlIdx)
	} else {
		response, err = r.client.AddPipeline(ctx, urlValue, urlIdx)
	}
	if err != nil {
		resp.Diagnostics.AddError("Create pipeline failed", err.Error())
		return
	}

	label := urlValue
	if label == "" {
		label = sourcePath
	}

	state, diags := pipelineStateFromResponse(ctx, r.client, response, label, urlIdx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.URL = plan.URL
	state.SourcePath = plan.SourcePath
	state.URLIdx = types.Int64Value(int64(urlIdx))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Read refreshes pipeline state.
func (r *pipelineResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing pipelines.")
		return
	}

	var state pipelineResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	urlIdx := int(state.URLIdx.ValueInt64())
	if state.URLIdx.IsNull() || state.URLIdx.IsUnknown() {
		urlIdx = 0
	}

	item, details, err := findPipeline(ctx, r.client, state.ID.ValueString(), state.URL.ValueString(), urlIdx)
	if err != nil {
		resp.Diagnostics.AddError("Read pipeline failed", err.Error())
		return
	}
	if item == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.ID = types.StringValue(pipelineString(item, "id", "pipeline_id", "pipelineId", "name"))
	state.PipelineID = state.ID
	state.DetailsJSON = details
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is a no-op; pipeline changes require replacement.
func (r *pipelineResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing pipelines.")
		return
	}

	var state pipelineResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete removes the pipeline.
func (r *pipelineResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing pipelines.")
		return
	}

	var state pipelineResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	urlIdx := int(state.URLIdx.ValueInt64())
	if state.URLIdx.IsNull() || state.URLIdx.IsUnknown() {
		urlIdx = 0
	}

	if state.ID.IsNull() || state.ID.IsUnknown() || state.ID.ValueString() == "" {
		resp.Diagnostics.AddError("Missing pipeline id", "Cannot delete a pipeline without a known ID.")
		return
	}

	if err := r.client.DeletePipeline(ctx, state.ID.ValueString(), urlIdx); err != nil {
		resp.Diagnostics.AddError("Delete pipeline failed", err.Error())
		return
	}
}

// ImportState maps import identifiers onto the id attribute.
func (r *pipelineResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")
	if len(parts) == 2 {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[0])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("pipeline_id"), parts[0])...)
		if value, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("url_idx"), value)...)
		}
		return
	}

	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func pipelineStateFromResponse(ctx context.Context, apiClient *client.Client, response map[string]any, urlValue string, urlIdx int) (pipelineResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	item := response
	id := pipelineString(item, "id", "pipeline_id", "pipelineId", "name")
	if id == "" {
		id = derivePipelineID(urlValue, item)
	}

	detailsJSON, err := encodeOptionalJSONValue(item)
	if err != nil {
		diags.AddError("Serialize pipeline", err.Error())
	}

	state := pipelineResourceModel{
		ID:          types.StringValue(id),
		PipelineID:  types.StringValue(id),
		URL:         types.StringValue(urlValue),
		URLIdx:      types.Int64Value(int64(urlIdx)),
		DetailsJSON: detailsJSON,
	}

	if id == "" {
		item, details, err := findPipeline(ctx, apiClient, id, urlValue, urlIdx)
		if err != nil {
			diags.AddError("Locate pipeline failed", err.Error())
			return state, diags
		}
		if item != nil {
			resolvedID := pipelineString(item, "id", "pipeline_id", "pipelineId", "name")
			state.ID = types.StringValue(resolvedID)
			state.PipelineID = state.ID
			state.DetailsJSON = details
		}
	}

	if id == "" {
		diags.AddError("Pipeline id missing", "Open WebUI did not return a pipeline id in the create response.")
	}

	return state, diags
}

func findPipeline(ctx context.Context, apiClient *client.Client, pipelineID string, urlValue string, urlIdx int) (map[string]any, types.String, error) {
	items, err := apiClient.GetPipelines(ctx, &urlIdx)
	if err != nil {
		return nil, types.StringNull(), err
	}

	for _, item := range items {
		id := pipelineString(item, "id", "pipeline_id", "pipelineId", "name")
		if pipelineID != "" && id == pipelineID {
			jsonVal, _ := encodeOptionalJSONValue(item)
			return item, jsonVal, nil
		}
		candidateURL := pipelineString(item, "url", "pipeline_url", "pipelineUrl", "source")
		if urlValue != "" && candidateURL == urlValue {
			jsonVal, _ := encodeOptionalJSONValue(item)
			return item, jsonVal, nil
		}
	}

	return nil, types.StringNull(), nil
}

func pipelineString(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if raw, ok := values[key]; ok {
			if str, ok := raw.(string); ok && str != "" {
				return str
			}
		}
	}
	return ""
}

func derivePipelineID(urlValue string, values map[string]any) string {
	if urlValue != "" {
		base := filepath.Base(urlValue)
		if base != "." && base != "/" {
			return base
		}
	}
	if name := pipelineString(values, "name", "title"); name != "" {
		return name
	}
	return ""
}
