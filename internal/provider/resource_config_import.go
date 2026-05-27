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

var _ resource.Resource = &configImportResource{}
var _ resource.ResourceWithConfigure = &configImportResource{}
var _ resource.ResourceWithImportState = &configImportResource{}

// configImportResource applies configuration exports.
type configImportResource struct {
	client *client.Client
}

type configImportModel struct {
	ID         types.String `tfsdk:"id"`
	ConfigJSON types.String `tfsdk:"config_json"`
}

// NewConfigImportResource constructs a new config import resource.
func NewConfigImportResource() resource.Resource {
	return &configImportResource{}
}

// Metadata sets the resource type name.
func (r *configImportResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_config_import"
}

// Schema defines the config import schema.
func (r *configImportResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Singleton identifier for config import.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"config_json": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "Full configuration export payload as JSON.",
			},
		},
	}
}

// Configure assigns the API client.
func (r *configImportResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	if client, ok := req.ProviderData.(*client.Client); ok {
		r.client = client
	}
}

// Create applies the configuration import.
func (r *configImportResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing config import.")
		return
	}

	var plan configImportModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applyConfigImport(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Read refreshes the configuration state from export.
func (r *configImportResource) Read(ctx context.Context, _ resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing config import.")
		return
	}

	config, err := r.client.ExportConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Read config export failed", err.Error())
		return
	}

	configJSON, err := encodeOptionalJSON(config)
	if err != nil {
		resp.Diagnostics.AddError("Serialize config export", err.Error())
		return
	}

	state := configImportModel{
		ID:         types.StringValue("config_import"),
		ConfigJSON: configJSON,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update reapplies the configuration import.
func (r *configImportResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing config import.")
		return
	}

	var plan configImportModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applyConfigImport(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete removes the resource from state without changing remote configuration.
func (r *configImportResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing config import.")
		return
	}
}

// ImportState maps import identifiers onto the id attribute.
func (r *configImportResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func applyConfigImport(ctx context.Context, apiClient *client.Client, plan configImportModel) (configImportModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	config := decodeOptionalJSON(plan.ConfigJSON, path.Root("config_json"), &diags)
	if diags.HasError() {
		return configImportModel{}, diags
	}
	if config == nil {
		diags.AddAttributeError(
			path.Root("config_json"),
			"Missing config JSON",
			"config_json must contain a JSON object describing the full configuration export.",
		)
		return configImportModel{}, diags
	}

	updated, err := apiClient.ImportConfig(ctx, config)
	if err != nil {
		diags.AddError("Import config failed", err.Error())
		return configImportModel{}, diags
	}

	stateConfig := plan.ConfigJSON
	if updated != nil {
		encoded, err := encodeOptionalJSON(updated)
		if err != nil {
			diags.AddError("Serialize config import", err.Error())
			return configImportModel{}, diags
		}
		stateConfig = encoded
	}

	state := configImportModel{
		ID:         types.StringValue("config_import"),
		ConfigJSON: stateConfig,
	}

	return state, diags
}
