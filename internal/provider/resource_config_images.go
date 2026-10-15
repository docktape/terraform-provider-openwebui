package provider

import (
	"context"
	"encoding/json"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

var _ resource.Resource = &imagesConfigResource{}
var _ resource.ResourceWithConfigure = &imagesConfigResource{}
var _ resource.ResourceWithImportState = &imagesConfigResource{}

// imageSizeValidator mirrors the backend check in update_config: "", "auto"
// or WIDTHxHEIGHT. Anything else is rejected with HTTP 400.
var imageSizeValidator = stringvalidator.RegexMatches(
	regexp.MustCompile(`^(|auto|[0-9]+x[0-9]+)$`),
	"must be WIDTHxHEIGHT (for example 512x512), auto, or an empty string",
)

// noEdgeSlashValidator rejects values with a leading or trailing slash: the
// backend stores ComfyUI base URLs with .strip('/'), so such a value would not
// round-trip.
var noEdgeSlashValidator = stringvalidator.RegexMatches(
	regexp.MustCompile(`^$|^[^/](.*[^/])?$`),
	"must not start or end with a slash (Open WebUI strips leading and trailing slashes from ComfyUI base URLs)",
)

type imagesConfigResource struct {
	client *client.Client
}

type imagesConfigModel struct {
	ID                                 types.String `tfsdk:"id"`
	EnableImageGeneration              types.Bool   `tfsdk:"enable_image_generation"`
	EnableImagePromptGeneration        types.Bool   `tfsdk:"enable_image_prompt_generation"`
	ImageGenerationEngine              types.String `tfsdk:"image_generation_engine"`
	ImageGenerationModel               types.String `tfsdk:"image_generation_model"`
	ImageSize                          types.String `tfsdk:"image_size"`
	ImageSteps                         types.Int64  `tfsdk:"image_steps"`
	ImagesOpenAIAPIBaseURL             types.String `tfsdk:"images_openai_api_base_url"`
	ImagesOpenAIAPIKey                 types.String `tfsdk:"images_openai_api_key"`
	ImagesOpenAIAPIVersion             types.String `tfsdk:"images_openai_api_version"`
	ImagesOpenAIAPIParamsJSON          types.String `tfsdk:"images_openai_api_params_json"`
	Automatic1111BaseURL               types.String `tfsdk:"automatic1111_base_url"`
	Automatic1111APIAuth               types.String `tfsdk:"automatic1111_api_auth"`
	Automatic1111ParamsJSON            types.String `tfsdk:"automatic1111_params_json"`
	ComfyUIBaseURL                     types.String `tfsdk:"comfyui_base_url"`
	ComfyUIAPIKey                      types.String `tfsdk:"comfyui_api_key"`
	ComfyUIWorkflow                    types.String `tfsdk:"comfyui_workflow"`
	ComfyUIWorkflowNodesJSON           types.String `tfsdk:"comfyui_workflow_nodes_json"`
	ImagesGeminiAPIBaseURL             types.String `tfsdk:"images_gemini_api_base_url"`
	ImagesGeminiAPIKey                 types.String `tfsdk:"images_gemini_api_key"`
	ImagesGeminiEndpointMethod         types.String `tfsdk:"images_gemini_endpoint_method"`
	EnableImageEdit                    types.Bool   `tfsdk:"enable_image_edit"`
	ImageEditEngine                    types.String `tfsdk:"image_edit_engine"`
	ImageEditModel                     types.String `tfsdk:"image_edit_model"`
	ImageEditSize                      types.String `tfsdk:"image_edit_size"`
	ImagesEditOpenAIAPIBaseURL         types.String `tfsdk:"images_edit_openai_api_base_url"`
	ImagesEditOpenAIAPIKey             types.String `tfsdk:"images_edit_openai_api_key"`
	ImagesEditOpenAIAPIVersion         types.String `tfsdk:"images_edit_openai_api_version"`
	ImagesEditGeminiAPIBaseURL         types.String `tfsdk:"images_edit_gemini_api_base_url"`
	ImagesEditGeminiAPIKey             types.String `tfsdk:"images_edit_gemini_api_key"`
	ImagesEditComfyUIBaseURL           types.String `tfsdk:"images_edit_comfyui_base_url"`
	ImagesEditComfyUIAPIKey            types.String `tfsdk:"images_edit_comfyui_api_key"`
	ImagesEditComfyUIWorkflow          types.String `tfsdk:"images_edit_comfyui_workflow"`
	ImagesEditComfyUIWorkflowNodesJSON types.String `tfsdk:"images_edit_comfyui_workflow_nodes_json"`
}

// NewImagesConfigResource constructs the image generation/editing config resource.
func NewImagesConfigResource() resource.Resource {
	return &imagesConfigResource{}
}

func (r *imagesConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_images_config"
}

func (r *imagesConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	str := func(desc string, validators ...validator.String) schema.StringAttribute {
		return schema.StringAttribute{Optional: true, Computed: true, Description: desc, MarkdownDescription: desc, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}, Validators: validators}
	}
	secret := func(desc string) schema.StringAttribute {
		return schema.StringAttribute{Optional: true, Computed: true, Sensitive: true, Description: desc, MarkdownDescription: desc, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}}
	}
	b := func(desc string) schema.BoolAttribute {
		return schema.BoolAttribute{Optional: true, Computed: true, Description: desc, MarkdownDescription: desc, PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()}}
	}
	const jsonNote = " Compared semantically: key order and whitespace differences are not drift."
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages Open WebUI's image generation and image editing settings (engines, models, sizes and per-engine connection details for OpenAI, Gemini, ComfyUI and AUTOMATIC1111).\n\n" +
			"~> Open WebUI's update endpoint replaces the whole image configuration at once. This resource reads the current configuration first and sends it back with only the attributes set in your configuration changed, so attributes you leave out keep their last-read (state) values (the current server values on first apply). Removing an attribute from your configuration therefore does not reset it; set it to the value you want instead.\n\n" +
			"~> The API keys and `automatic1111_api_auth` are stored in Terraform state because Open WebUI returns them in plain text. Protect your state accordingly.\n\n" +
			"-> Saving the configuration with `image_generation_engine` set to `automatic1111` (or empty) makes Open WebUI contact `automatic1111_base_url` to switch the checkpoint to `image_generation_model`; connection errors are ignored by the server.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Singleton identifier.",
				MarkdownDescription: "Singleton identifier.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"enable_image_generation":        b("Whether image generation is enabled. Server default: `false`."),
			"enable_image_prompt_generation": b("Whether a task model rewrites the user's request into an image prompt before generation. Server default: `true`."),
			"image_generation_engine":        str("Image generation engine: `openai`, `gemini`, `comfyui` or `automatic1111` (an empty string is treated as `automatic1111`). Server default: `openai`."),
			"image_generation_model":         str("Model used for image generation. Empty means the engine's default."),
			"image_size":                     str("Generated image size as `WIDTHxHEIGHT` (for example `512x512`), `auto`, or an empty string. Open WebUI only accepts `auto` when `image_generation_model` matches its `IMAGE_AUTO_SIZE_MODELS_REGEX_PATTERN` (default `^gpt-image`). On Open WebUI v0.9.6 the server writes the generation toggles, engine and model before it rejects `auto` for a non-matching model with HTTP 400, so such a failed apply can still leave those values changed. Server default: `512x512`.", imageSizeValidator),
			"image_steps": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Description:         "Number of sampling steps (ComfyUI and AUTOMATIC1111). Must be 0 or greater. Server default: `50`.",
				MarkdownDescription: "Number of sampling steps (ComfyUI and AUTOMATIC1111). Must be 0 or greater. Server default: `50`.",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
				Validators:          []validator.Int64{int64validator.AtLeast(0)},
			},
			"images_openai_api_base_url":              str("Base URL of the OpenAI-compatible image generation API. Server default: the `OPENAI_API_BASE_URL` setting."),
			"images_openai_api_key":                   secret("API key for the OpenAI-compatible image generation API."),
			"images_openai_api_version":               str("API version sent to the OpenAI-compatible endpoint (for example for Azure OpenAI). Empty means none."),
			"images_openai_api_params_json":           str("JSON object of extra parameters merged into OpenAI image generation requests, for example `jsonencode({ quality = \"hd\" })`. Use `{}` for none. Server default: `{}`."+jsonNote, jsonObjectValidator()),
			"automatic1111_base_url":                  str("Base URL of the AUTOMATIC1111 API."),
			"automatic1111_api_auth":                  secret("AUTOMATIC1111 API credentials as `username:password`, sent as HTTP basic auth."),
			"automatic1111_params_json":               str("JSON object of extra parameters merged into AUTOMATIC1111 txt2img requests, for example `jsonencode({ cfg_scale = 7 })`. Use `{}` for none. Server default: `{}`."+jsonNote, jsonObjectValidator()),
			"comfyui_base_url":                        str("Base URL of the ComfyUI API, without leading or trailing slashes (Open WebUI strips them).", noEdgeSlashValidator),
			"comfyui_api_key":                         secret("Bearer token for the ComfyUI API."),
			"comfyui_workflow":                        str("ComfyUI workflow (API format) as JSON text. Stored verbatim as a string by Open WebUI, so it is compared exactly. Server default: Open WebUI's built-in text-to-image workflow."),
			"comfyui_workflow_nodes_json":             str("JSON array of objects mapping workflow inputs to node IDs, for example `jsonencode([{ type = \"prompt\", key = \"text\", node_ids = [\"6\"] }])`. Use `[]` for none. Server default: `[]`."+jsonNote, jsonArrayOfObjectsValidator()),
			"images_gemini_api_base_url":              str("Base URL of the Gemini image generation API. Server default: the `GEMINI_API_BASE_URL` setting."),
			"images_gemini_api_key":                   secret("API key for the Gemini image generation API."),
			"images_gemini_endpoint_method":           str("Gemini endpoint method, for example `predict` or `generateContent`. Empty lets Open WebUI choose."),
			"enable_image_edit":                       b("Whether image editing is enabled. Server default: `false`."),
			"image_edit_engine":                       str("Image editing engine: `openai`, `gemini` or `comfyui`. Server default: `openai`."),
			"image_edit_model":                        str("Model used for image editing. Empty means the engine's default."),
			"image_edit_size":                         str("Edited image size, for example `1024x1024`. Empty keeps the input size. Server default: empty string."),
			"images_edit_openai_api_base_url":         str("Base URL of the OpenAI-compatible image editing API. Server default: the `OPENAI_API_BASE_URL` setting."),
			"images_edit_openai_api_key":              secret("API key for the OpenAI-compatible image editing API."),
			"images_edit_openai_api_version":          str("API version sent to the OpenAI-compatible image editing endpoint. Empty means none."),
			"images_edit_gemini_api_base_url":         str("Base URL of the Gemini image editing API. Server default: the `GEMINI_API_BASE_URL` setting."),
			"images_edit_gemini_api_key":              secret("API key for the Gemini image editing API."),
			"images_edit_comfyui_base_url":            str("Base URL of the ComfyUI API used for image editing, without leading or trailing slashes (Open WebUI strips them).", noEdgeSlashValidator),
			"images_edit_comfyui_api_key":             secret("Bearer token for the ComfyUI image editing API."),
			"images_edit_comfyui_workflow":            str("ComfyUI image editing workflow (API format) as JSON text, stored verbatim. Server default: empty string."),
			"images_edit_comfyui_workflow_nodes_json": str("JSON array of objects mapping image editing workflow inputs to node IDs. Use `[]` for none. Server default: `[]`."+jsonNote, jsonArrayOfObjectsValidator()),
		},
	}
}

func (r *imagesConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	if c, ok := req.ProviderData.(*client.Client); ok {
		r.client = c
	}
}

func (r *imagesConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing images config.")
		return
	}

	var plan imagesConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applyImagesConfig(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *imagesConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing images config.")
		return
	}

	var plan imagesConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applyImagesConfig(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// applyImagesConfig performs a fetch-then-merge write. The backend's
// ImagesConfig model has no defaults, so the POST must carry all 33 fields and
// replaces the whole configuration; sending only the planned values would
// fail validation (422), and sending zero values would wipe settings. The
// current configuration is therefore read first and every unknown attribute
// keeps its server value.
func applyImagesConfig(ctx context.Context, apiClient *client.Client, plan imagesConfigModel) (imagesConfigModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	current, err := apiClient.GetImagesConfig(ctx)
	if err != nil {
		diags.AddError("Read current images config failed", err.Error())
		return imagesConfigModel{}, diags
	}

	form, formDiags := overlayImagesConfigPlan(*current, plan)
	diags.Append(formDiags...)
	if diags.HasError() {
		return imagesConfigModel{}, diags
	}

	updated, err := apiClient.SetImagesConfig(ctx, form)
	if err != nil {
		diags.AddError("Update images config failed", err.Error())
		return imagesConfigModel{}, diags
	}

	return preserveImagesConfigJSON(plan, imagesConfigModelFromForm(*updated)), diags
}

func (r *imagesConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing images config.")
		return
	}

	var prior imagesConfigModel
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	current, err := r.client.GetImagesConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Read images config failed", err.Error())
		return
	}

	state := preserveImagesConfigJSON(prior, imagesConfigModelFromForm(*current))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *imagesConfigResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing images config.")
		return
	}
	// Singleton config: destroying the resource leaves the remote config untouched.
}

func (r *imagesConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// overlayImagesConfigPlan returns current with every known plan value applied.
// Null and unknown attributes keep the current server value.
func overlayImagesConfigPlan(current client.ImagesConfigForm, m imagesConfigModel) (client.ImagesConfigForm, diag.Diagnostics) {
	var diags diag.Diagnostics
	f := current

	setBool := func(dst *bool, v types.Bool) {
		if !v.IsNull() && !v.IsUnknown() {
			*dst = v.ValueBool()
		}
	}
	setString := func(dst *string, v types.String) {
		if !v.IsNull() && !v.IsUnknown() {
			*dst = v.ValueString()
		}
	}
	setStringPtr := func(dst **string, v types.String) {
		if p := stringPtr(v); p != nil {
			*dst = p
		}
	}
	setJSON := func(dst *json.RawMessage, v types.String, attr string) {
		if v.IsNull() || v.IsUnknown() {
			return
		}
		// Keep the server's raw value when it already means the same thing, so a
		// legacy JSON-string value is not rewritten by an unrelated apply.
		if existing := rawJSONToState(*dst); !existing.IsNull() && equivalentJSON(existing.ValueString(), v.ValueString()) {
			return
		}
		canonical, err := canonicalJSON([]byte(v.ValueString()))
		if err != nil {
			diags.AddAttributeError(path.Root(attr), "Invalid JSON value", err.Error())
			return
		}
		*dst = json.RawMessage(canonical)
	}

	setBool(&f.EnableImageGeneration, m.EnableImageGeneration)
	setBool(&f.EnableImagePromptGeneration, m.EnableImagePromptGeneration)
	setString(&f.ImageGenerationEngine, m.ImageGenerationEngine)
	setString(&f.ImageGenerationModel, m.ImageGenerationModel)
	setStringPtr(&f.ImageSize, m.ImageSize)
	if p := int64Ptr(m.ImageSteps); p != nil {
		f.ImageSteps = p
	}
	setString(&f.ImagesOpenAIAPIBaseURL, m.ImagesOpenAIAPIBaseURL)
	setString(&f.ImagesOpenAIAPIKey, m.ImagesOpenAIAPIKey)
	setString(&f.ImagesOpenAIAPIVersion, m.ImagesOpenAIAPIVersion)
	setJSON(&f.ImagesOpenAIAPIParams, m.ImagesOpenAIAPIParamsJSON, "images_openai_api_params_json")
	setString(&f.Automatic1111BaseURL, m.Automatic1111BaseURL)
	setStringPtr(&f.Automatic1111APIAuth, m.Automatic1111APIAuth)
	setJSON(&f.Automatic1111Params, m.Automatic1111ParamsJSON, "automatic1111_params_json")
	setString(&f.ComfyUIBaseURL, m.ComfyUIBaseURL)
	setString(&f.ComfyUIAPIKey, m.ComfyUIAPIKey)
	setString(&f.ComfyUIWorkflow, m.ComfyUIWorkflow)
	setJSON(&f.ComfyUIWorkflowNodes, m.ComfyUIWorkflowNodesJSON, "comfyui_workflow_nodes_json")
	setString(&f.ImagesGeminiAPIBaseURL, m.ImagesGeminiAPIBaseURL)
	setString(&f.ImagesGeminiAPIKey, m.ImagesGeminiAPIKey)
	setString(&f.ImagesGeminiEndpointMethod, m.ImagesGeminiEndpointMethod)
	setBool(&f.EnableImageEdit, m.EnableImageEdit)
	setString(&f.ImageEditEngine, m.ImageEditEngine)
	setString(&f.ImageEditModel, m.ImageEditModel)
	setStringPtr(&f.ImageEditSize, m.ImageEditSize)
	setString(&f.ImagesEditOpenAIAPIBaseURL, m.ImagesEditOpenAIAPIBaseURL)
	setString(&f.ImagesEditOpenAIAPIKey, m.ImagesEditOpenAIAPIKey)
	setString(&f.ImagesEditOpenAIAPIVersion, m.ImagesEditOpenAIAPIVersion)
	setString(&f.ImagesEditGeminiAPIBaseURL, m.ImagesEditGeminiAPIBaseURL)
	setString(&f.ImagesEditGeminiAPIKey, m.ImagesEditGeminiAPIKey)
	setString(&f.ImagesEditComfyUIBaseURL, m.ImagesEditComfyUIBaseURL)
	setString(&f.ImagesEditComfyUIAPIKey, m.ImagesEditComfyUIAPIKey)
	setString(&f.ImagesEditComfyUIWorkflow, m.ImagesEditComfyUIWorkflow)
	setJSON(&f.ImagesEditComfyUIWorkflowNodes, m.ImagesEditComfyUIWorkflowNodesJSON, "images_edit_comfyui_workflow_nodes_json")

	return f, diags
}

func imagesConfigModelFromForm(f client.ImagesConfigForm) imagesConfigModel {
	return imagesConfigModel{
		ID:                                 types.StringValue("images"),
		EnableImageGeneration:              types.BoolValue(f.EnableImageGeneration),
		EnableImagePromptGeneration:        types.BoolValue(f.EnableImagePromptGeneration),
		ImageGenerationEngine:              types.StringValue(f.ImageGenerationEngine),
		ImageGenerationModel:               types.StringValue(f.ImageGenerationModel),
		ImageSize:                          stringValueOrNull(f.ImageSize),
		ImageSteps:                         int64ValueOrNull(f.ImageSteps),
		ImagesOpenAIAPIBaseURL:             types.StringValue(f.ImagesOpenAIAPIBaseURL),
		ImagesOpenAIAPIKey:                 types.StringValue(f.ImagesOpenAIAPIKey),
		ImagesOpenAIAPIVersion:             types.StringValue(f.ImagesOpenAIAPIVersion),
		ImagesOpenAIAPIParamsJSON:          rawJSONToState(f.ImagesOpenAIAPIParams),
		Automatic1111BaseURL:               types.StringValue(f.Automatic1111BaseURL),
		Automatic1111APIAuth:               stringValueOrNull(f.Automatic1111APIAuth),
		Automatic1111ParamsJSON:            rawJSONToState(f.Automatic1111Params),
		ComfyUIBaseURL:                     types.StringValue(f.ComfyUIBaseURL),
		ComfyUIAPIKey:                      types.StringValue(f.ComfyUIAPIKey),
		ComfyUIWorkflow:                    types.StringValue(f.ComfyUIWorkflow),
		ComfyUIWorkflowNodesJSON:           rawJSONToState(f.ComfyUIWorkflowNodes),
		ImagesGeminiAPIBaseURL:             types.StringValue(f.ImagesGeminiAPIBaseURL),
		ImagesGeminiAPIKey:                 types.StringValue(f.ImagesGeminiAPIKey),
		ImagesGeminiEndpointMethod:         types.StringValue(f.ImagesGeminiEndpointMethod),
		EnableImageEdit:                    types.BoolValue(f.EnableImageEdit),
		ImageEditEngine:                    types.StringValue(f.ImageEditEngine),
		ImageEditModel:                     types.StringValue(f.ImageEditModel),
		ImageEditSize:                      stringValueOrNull(f.ImageEditSize),
		ImagesEditOpenAIAPIBaseURL:         types.StringValue(f.ImagesEditOpenAIAPIBaseURL),
		ImagesEditOpenAIAPIKey:             types.StringValue(f.ImagesEditOpenAIAPIKey),
		ImagesEditOpenAIAPIVersion:         types.StringValue(f.ImagesEditOpenAIAPIVersion),
		ImagesEditGeminiAPIBaseURL:         types.StringValue(f.ImagesEditGeminiAPIBaseURL),
		ImagesEditGeminiAPIKey:             types.StringValue(f.ImagesEditGeminiAPIKey),
		ImagesEditComfyUIBaseURL:           types.StringValue(f.ImagesEditComfyUIBaseURL),
		ImagesEditComfyUIAPIKey:            types.StringValue(f.ImagesEditComfyUIAPIKey),
		ImagesEditComfyUIWorkflow:          types.StringValue(f.ImagesEditComfyUIWorkflow),
		ImagesEditComfyUIWorkflowNodesJSON: rawJSONToState(f.ImagesEditComfyUIWorkflowNodes),
	}
}

// preserveImagesConfigJSON keeps the preferred (plan or prior state) text of
// each JSON attribute when it is semantically equal to the server's value.
func preserveImagesConfigJSON(preferred, fresh imagesConfigModel) imagesConfigModel {
	fresh.ImagesOpenAIAPIParamsJSON = preserveEquivalentJSON(preferred.ImagesOpenAIAPIParamsJSON, fresh.ImagesOpenAIAPIParamsJSON)
	fresh.Automatic1111ParamsJSON = preserveEquivalentJSON(preferred.Automatic1111ParamsJSON, fresh.Automatic1111ParamsJSON)
	fresh.ComfyUIWorkflowNodesJSON = preserveEquivalentJSON(preferred.ComfyUIWorkflowNodesJSON, fresh.ComfyUIWorkflowNodesJSON)
	fresh.ImagesEditComfyUIWorkflowNodesJSON = preserveEquivalentJSON(preferred.ImagesEditComfyUIWorkflowNodesJSON, fresh.ImagesEditComfyUIWorkflowNodesJSON)
	return fresh
}
