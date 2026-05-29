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

var _ resource.Resource = &fileResource{}
var _ resource.ResourceWithConfigure = &fileResource{}
var _ resource.ResourceWithImportState = &fileResource{}

// fileResource manages uploaded files.
type fileResource struct {
	client *client.Client
}

// fileResourceModel maps Terraform state for files.
type fileResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	SourcePath          types.String `tfsdk:"source_path"`
	MetadataJSON        types.String `tfsdk:"metadata_json"`
	Process             types.Bool   `tfsdk:"process"`
	ProcessInBackground types.Bool   `tfsdk:"process_in_background"`
	Filename            types.String `tfsdk:"filename"`
	Hash                types.String `tfsdk:"hash"`
	UserID              types.String `tfsdk:"user_id"`
	DataJSON            types.String `tfsdk:"data_json"`
	MetaJSON            types.String `tfsdk:"meta_json"`
	CreatedAt           types.Int64  `tfsdk:"created_at"`
	UpdatedAt           types.Int64  `tfsdk:"updated_at"`
}

// NewFileResource constructs a new file resource.
func NewFileResource() resource.Resource {
	return &fileResource{}
}

// Metadata sets the resource type name.
func (r *fileResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_file"
}

// Schema defines the file resource schema.
func (r *fileResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Uploads a local file to Open WebUI. Replacement is forced on any attribute change because files are immutable after upload.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "UUID assigned by Open WebUI on upload.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"source_path": schema.StringAttribute{
				Required:      true,
				Description:   "Absolute or module-relative path to the file to upload. Forces replacement.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"metadata_json": schema.StringAttribute{
				Optional:      true,
				Description:   "Optional JSON metadata to attach to the file on upload. e.g. `jsonencode({ category = \"docs\" })`. Forces replacement.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"process": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Whether Open WebUI should process the file for RAG (chunk and embed it). Defaults to `false`.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown(), boolplanmodifier.RequiresReplace()},
			},
			"process_in_background": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Whether RAG processing runs asynchronously. Only relevant when `process = true`. Defaults to `false`.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown(), boolplanmodifier.RequiresReplace()},
			},
			"filename": schema.StringAttribute{
				Computed:    true,
				Description: "Original filename as stored by Open WebUI. Set by Open WebUI.",
			},
			"hash": schema.StringAttribute{
				Computed:    true,
				Description: "SHA-256 hash of the uploaded file content. Set by Open WebUI.",
			},
			"user_id": schema.StringAttribute{
				Computed:    true,
				Description: "Owner user identifier. Set by Open WebUI.",
			},
			"data_json": schema.StringAttribute{
				Computed:    true,
				Description: "JSON data blob associated with the file as returned by Open WebUI.",
			},
			"meta_json": schema.StringAttribute{
				Computed:    true,
				Description: "JSON metadata blob associated with the file as returned by Open WebUI.",
			},
			"created_at": schema.Int64Attribute{
				Computed:      true,
				Description:   "Unix timestamp of when the file was uploaded. Set by Open WebUI.",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.Int64Attribute{
				Computed:      true,
				Description:   "Unix timestamp of when the file record was last updated. Set by Open WebUI.",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
		},
	}
}

// Configure assigns the API client.
func (r *fileResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	if client, ok := req.ProviderData.(*client.Client); ok {
		r.client = client
	}
}

// Create uploads a new file.
func (r *fileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing files.")
		return
	}

	var plan fileResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	process := true
	if !plan.Process.IsNull() && !plan.Process.IsUnknown() {
		process = plan.Process.ValueBool()
	}
	processInBackground := true
	if !plan.ProcessInBackground.IsNull() && !plan.ProcessInBackground.IsUnknown() {
		processInBackground = plan.ProcessInBackground.ValueBool()
	}

	metadata := ""
	if !plan.MetadataJSON.IsNull() && !plan.MetadataJSON.IsUnknown() {
		metadata = plan.MetadataJSON.ValueString()
	}

	uploaded, err := r.client.UploadFile(ctx, plan.SourcePath.ValueString(), metadata, process, processInBackground)
	if err != nil {
		resp.Diagnostics.AddError("Upload file failed", err.Error())
		return
	}

	state, diags := fileStateFromAPI(ctx, r.client, uploaded.ID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.SourcePath = plan.SourcePath
	state.MetadataJSON = plan.MetadataJSON
	state.Process = types.BoolValue(process)
	state.ProcessInBackground = types.BoolValue(processInBackground)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Read refreshes file state.
func (r *fileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing files.")
		return
	}

	var state fileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, diags := fileStateFromAPI(ctx, r.client, state.ID.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if updated.ID.IsNull() || updated.ID.IsUnknown() || updated.ID.ValueString() == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	updated.SourcePath = state.SourcePath
	updated.MetadataJSON = state.MetadataJSON
	updated.Process = state.Process
	updated.ProcessInBackground = state.ProcessInBackground
	resp.Diagnostics.Append(resp.State.Set(ctx, &updated)...)
}

// Update is a no-op; file changes require replacement.
func (r *fileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing files.")
		return
	}

	var state fileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete removes the file.
func (r *fileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing files.")
		return
	}

	var state fileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteFile(ctx, state.ID.ValueString()); err != nil {
		if err == client.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Delete file failed", err.Error())
		return
	}
}

// ImportState maps import identifiers to the id attribute.
func (r *fileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func fileStateFromAPI(ctx context.Context, apiClient *client.Client, id string) (fileResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	file, err := apiClient.GetFile(ctx, id)
	if err != nil {
		if err == client.ErrNotFound {
			return fileResourceModel{}, diags
		}
		diags.AddError("Read file failed", err.Error())
		return fileResourceModel{}, diags
	}

	dataJSON, err := encodeOptionalJSON(file.Data)
	if err != nil {
		diags.AddError("Serialize file data", err.Error())
	}
	metaJSON, err := encodeOptionalJSON(file.Meta)
	if err != nil {
		diags.AddError("Serialize file metadata", err.Error())
	}

	createdAt := types.Int64Value(file.CreatedAt)
	updatedAt := types.Int64Value(file.UpdatedAt)

	hash := types.StringNull()
	if file.Hash != nil {
		hash = types.StringValue(*file.Hash)
	}

	state := fileResourceModel{
		ID:        types.StringValue(file.ID),
		Filename:  types.StringValue(file.Filename),
		Hash:      hash,
		UserID:    types.StringValue(file.UserID),
		DataJSON:  dataJSON,
		MetaJSON:  metaJSON,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	return state, diags
}
