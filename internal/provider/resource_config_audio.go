package provider

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

var _ resource.Resource = &audioConfigResource{}
var _ resource.ResourceWithConfigure = &audioConfigResource{}
var _ resource.ResourceWithImportState = &audioConfigResource{}

type audioConfigResource struct {
	client *client.Client
}

// audioConfigModel holds the two nested sub-objects as types.Object so that a
// sub-object the user omits can be planned as unknown (and filled from the
// server) rather than failing to decode into a Go struct.
type audioConfigModel struct {
	ID  types.String `tfsdk:"id"`
	TTS types.Object `tfsdk:"tts"`
	STT types.Object `tfsdk:"stt"`
}

type audioTTSModel struct {
	OpenAIAPIBaseURL        types.String `tfsdk:"openai_api_base_url"`
	OpenAIAPIKey            types.String `tfsdk:"openai_api_key"`
	OpenAIParamsJSON        types.String `tfsdk:"openai_params_json"`
	APIKey                  types.String `tfsdk:"api_key"`
	Engine                  types.String `tfsdk:"engine"`
	Model                   types.String `tfsdk:"model"`
	Voice                   types.String `tfsdk:"voice"`
	SplitOn                 types.String `tfsdk:"split_on"`
	AzureSpeechRegion       types.String `tfsdk:"azure_speech_region"`
	AzureSpeechBaseURL      types.String `tfsdk:"azure_speech_base_url"`
	AzureSpeechOutputFormat types.String `tfsdk:"azure_speech_output_format"`
	MistralAPIKey           types.String `tfsdk:"mistral_api_key"`
	MistralAPIBaseURL       types.String `tfsdk:"mistral_api_base_url"`
}

type audioSTTModel struct {
	OpenAIAPIBaseURL          types.String `tfsdk:"openai_api_base_url"`
	OpenAIAPIKey              types.String `tfsdk:"openai_api_key"`
	OpenAIAPIRequestFormat    types.String `tfsdk:"openai_api_request_format"`
	Engine                    types.String `tfsdk:"engine"`
	Model                     types.String `tfsdk:"model"`
	SupportedContentTypes     types.List   `tfsdk:"supported_content_types"`
	AllowedExtensions         types.List   `tfsdk:"allowed_extensions"`
	WhisperModel              types.String `tfsdk:"whisper_model"`
	DeepgramAPIKey            types.String `tfsdk:"deepgram_api_key"`
	AzureAPIKey               types.String `tfsdk:"azure_api_key"`
	AzureRegion               types.String `tfsdk:"azure_region"`
	AzureLocales              types.String `tfsdk:"azure_locales"`
	AzureBaseURL              types.String `tfsdk:"azure_base_url"`
	AzureMaxSpeakers          types.String `tfsdk:"azure_max_speakers"`
	MistralAPIKey             types.String `tfsdk:"mistral_api_key"`
	MistralAPIBaseURL         types.String `tfsdk:"mistral_api_base_url"`
	MistralUseChatCompletions types.Bool   `tfsdk:"mistral_use_chat_completions"`
}

var audioTTSAttrTypes = map[string]attr.Type{
	"openai_api_base_url":        types.StringType,
	"openai_api_key":             types.StringType,
	"openai_params_json":         types.StringType,
	"api_key":                    types.StringType,
	"engine":                     types.StringType,
	"model":                      types.StringType,
	"voice":                      types.StringType,
	"split_on":                   types.StringType,
	"azure_speech_region":        types.StringType,
	"azure_speech_base_url":      types.StringType,
	"azure_speech_output_format": types.StringType,
	"mistral_api_key":            types.StringType,
	"mistral_api_base_url":       types.StringType,
}

var audioSTTAttrTypes = map[string]attr.Type{
	"openai_api_base_url":          types.StringType,
	"openai_api_key":               types.StringType,
	"openai_api_request_format":    types.StringType,
	"engine":                       types.StringType,
	"model":                        types.StringType,
	"supported_content_types":      types.ListType{ElemType: types.StringType},
	"allowed_extensions":           types.ListType{ElemType: types.StringType},
	"whisper_model":                types.StringType,
	"deepgram_api_key":             types.StringType,
	"azure_api_key":                types.StringType,
	"azure_region":                 types.StringType,
	"azure_locales":                types.StringType,
	"azure_base_url":               types.StringType,
	"azure_max_speakers":           types.StringType,
	"mistral_api_key":              types.StringType,
	"mistral_api_base_url":         types.StringType,
	"mistral_use_chat_completions": types.BoolType,
}

// NewAudioConfigResource constructs the text-to-speech / speech-to-text config resource.
func NewAudioConfigResource() resource.Resource {
	return &audioConfigResource{}
}

func (r *audioConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_audio_config"
}

func (r *audioConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	str := func(desc string, validators ...validator.String) schema.StringAttribute {
		return schema.StringAttribute{Optional: true, Computed: true, Description: desc, MarkdownDescription: desc, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}, Validators: validators}
	}
	secret := func(desc string) schema.StringAttribute {
		return schema.StringAttribute{Optional: true, Computed: true, Sensitive: true, Description: desc, MarkdownDescription: desc, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}}
	}
	list := func(desc string) schema.ListAttribute {
		return schema.ListAttribute{ElementType: types.StringType, Optional: true, Computed: true, Description: desc, MarkdownDescription: desc, PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()}}
	}
	nested := func(desc string, attrs map[string]schema.Attribute) schema.SingleNestedAttribute {
		return schema.SingleNestedAttribute{Optional: true, Computed: true, Description: desc, MarkdownDescription: desc, PlanModifiers: []planmodifier.Object{objectplanmodifier.UseStateForUnknown()}, Attributes: attrs}
	}
	const whisperNote = "When `stt.engine` is empty (local Whisper), every apply of this resource makes Open WebUI load `stt.whisper_model` before it responds, and download it from Hugging Face if it is not in the server's model cache (this can take minutes). An unknown model name makes the update fail with an HTTP 500 after the new settings have already been stored. Setting any other engine unloads the local model."

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages Open WebUI's audio settings: text-to-speech (`tts`) and speech-to-text (`stt`) engines, models, voices and per-engine connection details.\n\n" +
			"~> Open WebUI's update endpoint replaces the whole audio configuration (both `tts` and `stt`) at once. This resource reads the current configuration first and sends it back with only the attributes set in your configuration changed, so omitted attributes (and an omitted `tts` or `stt` block) keep their last-read (state) values (the current server values on first apply). Removing an attribute from your configuration therefore does not reset it; set it to the value you want instead.\n\n" +
			"~> The API keys are stored in Terraform state because Open WebUI returns them in plain text. Protect your state accordingly.\n\n" +
			"-> " + whisperNote,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Singleton identifier.",
				MarkdownDescription: "Singleton identifier.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"tts": nested("Text-to-speech settings. Omit the block, or any attribute in it, to keep the last-read (state) values.", map[string]schema.Attribute{
				"openai_api_base_url":        str("Base URL of the OpenAI-compatible speech API. Server default: the `OPENAI_API_BASE_URL` setting."),
				"openai_api_key":             secret("API key for the OpenAI-compatible speech API."),
				"openai_params_json":         str("JSON object of extra parameters merged into OpenAI speech requests, for example `jsonencode({ speed = 1.25 })`. Use `{}` for none. Server default: `{}`. Compared semantically: key order and whitespace differences are not drift.", jsonObjectValidator()),
				"api_key":                    secret("API key for the ElevenLabs and Azure AI Speech engines."),
				"engine":                     str("Text-to-speech engine: empty for the browser's Web API, `transformers` (local), `openai`, `elevenlabs`, `azure` or `mistral`. Server default: empty."),
				"model":                      str("Text-to-speech model. Server default: `tts-1`."),
				"voice":                      str("Voice used for speech output. Server default: `alloy`."),
				"split_on":                   str("How responses are split into chunks for speech: `punctuation`, `paragraphs` or `none`. Server default: `punctuation`."),
				"azure_speech_region":        str("Azure AI Speech region, for example `eastus`."),
				"azure_speech_base_url":      str("Azure AI Speech base URL; overrides the region-derived endpoint when set."),
				"azure_speech_output_format": str("Azure AI Speech output format. Server default: `audio-24khz-160kbitrate-mono-mp3`."),
				"mistral_api_key":            secret("API key for the Mistral text-to-speech engine."),
				"mistral_api_base_url":       str("Base URL of the Mistral API for text-to-speech. Server default: `https://api.mistral.ai/v1`."),
			}),
			"stt": nested("Speech-to-text settings. Omit the block, or any attribute in it, to keep the last-read (state) values.", map[string]schema.Attribute{
				"openai_api_base_url":       str("Base URL of the OpenAI-compatible transcription API. Server default: the `OPENAI_API_BASE_URL` setting."),
				"openai_api_key":            secret("API key for the OpenAI-compatible transcription API."),
				"openai_api_request_format": str("How audio is sent to the OpenAI-compatible transcription API: `multipart` (file upload) or `json` (base64 in a JSON body). Server default: `multipart`. Added in Open WebUI v0.10.2; setting it against an older server is an error."),
				"engine":                    str("Speech-to-text engine: empty for local Whisper, `openai`, `web` (browser), `deepgram`, `azure` or `mistral`. Server default: empty. Leaving it empty makes every apply reload `whisper_model` (see the resource notes)."),
				"model":                     str("Speech-to-text model for the remote engines, for example `whisper-1`."),
				"supported_content_types":   list("MIME types accepted for transcription, for example `[\"audio/*\"]`. An empty list means Open WebUI's built-in default (`audio/*` and `video/webm`). Server default: `[]`."),
				"allowed_extensions":        list("File extensions accepted for transcription. An empty list allows any extension. Server default: `[\"mp3\", \"wav\", \"m4a\", \"webm\", \"ogg\", \"flac\", \"mp4\", \"mpga\", \"mpeg\"]`. Open WebUI's admin UI does not send this field, so saving the audio settings there resets it to `[]`."),
				"whisper_model":             str("Local faster-whisper model (for example `base` or `small`) used when `engine` is empty. Server default: `base`. Loaded (and downloaded if not cached) on every apply while `engine` is empty; see the resource notes."),
				"deepgram_api_key":          secret("API key for the Deepgram engine."),
				"azure_api_key":             secret("API key for the Azure AI Speech transcription engine."),
				"azure_region":              str("Azure AI Speech region for transcription, for example `eastus`."),
				"azure_locales":             str("Comma-separated Azure AI Speech locales for transcription, for example `en-US,de-DE`. Empty lets Azure detect the language."),
				"azure_base_url":            str("Azure AI Speech base URL for transcription; overrides the region-derived endpoint when set."),
				"azure_max_speakers":        str("Maximum number of speakers for Azure diarization, as a string (for example `3`). Empty means Open WebUI's default of 3."),
				"mistral_api_key":           secret("API key for the Mistral transcription engine."),
				"mistral_api_base_url":      str("Base URL of the Mistral API for transcription. Server default: `https://api.mistral.ai/v1`."),
				"mistral_use_chat_completions": schema.BoolAttribute{
					Optional:            true,
					Computed:            true,
					Description:         "Whether Mistral transcription uses the chat completions endpoint instead of the dedicated transcription endpoint. Server default: `false`.",
					MarkdownDescription: "Whether Mistral transcription uses the chat completions endpoint instead of the dedicated transcription endpoint. Server default: `false`.",
					PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
				},
			}),
		},
	}
}

func (r *audioConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	if c, ok := req.ProviderData.(*client.Client); ok {
		r.client = c
	}
}

func (r *audioConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing audio config.")
		return
	}

	var plan audioConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applyAudioConfig(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *audioConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing audio config.")
		return
	}

	var plan audioConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applyAudioConfig(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// applyAudioConfig performs a fetch-then-merge write. AudioConfigUpdateForm
// requires both sub-objects and nearly every field in them, and the handler
// upserts every field it receives, so the POST replaces the whole audio
// configuration. The current configuration is therefore read first and every
// unknown attribute (or wholly unknown sub-object) keeps its server value.
func applyAudioConfig(ctx context.Context, apiClient *client.Client, plan audioConfigModel) (audioConfigModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	current, err := apiClient.GetAudioConfig(ctx)
	if err != nil {
		diags.AddError("Read current audio config failed", err.Error())
		return audioConfigModel{}, diags
	}

	form, formDiags := overlayAudioConfigPlan(ctx, *current, plan)
	diags.Append(formDiags...)
	if diags.HasError() {
		return audioConfigModel{}, diags
	}

	updated, err := apiClient.SetAudioConfig(ctx, form)
	if err != nil {
		diags.AddError("Update audio config failed", err.Error())
		return audioConfigModel{}, diags
	}

	state, stateDiags := audioConfigModelFromForm(ctx, *updated)
	diags.Append(stateDiags...)
	if diags.HasError() {
		return audioConfigModel{}, diags
	}
	return preserveAudioConfigJSON(ctx, plan, state), diags
}

func (r *audioConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing audio config.")
		return
	}

	var prior audioConfigModel
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	current, err := r.client.GetAudioConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Read audio config failed", err.Error())
		return
	}

	state, diags := audioConfigModelFromForm(ctx, *current)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state = preserveAudioConfigJSON(ctx, prior, state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *audioConfigResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing audio config.")
		return
	}
	// Singleton config: destroying the resource leaves the remote config untouched.
}

func (r *audioConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// overlayAudioConfigPlan returns current with every known plan value applied.
// Null or unknown sub-objects and attributes keep the current server value.
func overlayAudioConfigPlan(ctx context.Context, current client.AudioConfigForm, m audioConfigModel) (client.AudioConfigForm, diag.Diagnostics) {
	var diags diag.Diagnostics
	f := current
	// OPENAI_PARAMS is Optional[dict] on write, but the GET has no response
	// model, so a stored non-object value (e.g. a string) would be echoed back
	// and rejected with HTTP 422. Send null instead.
	if !isJSONObject(f.TTS.OpenAIParams) {
		f.TTS.OpenAIParams = nil
	}

	setString := func(dst *string, v types.String) {
		if !v.IsNull() && !v.IsUnknown() {
			*dst = v.ValueString()
		}
	}

	if knownObject(m.TTS) {
		var tts audioTTSModel
		diags.Append(m.TTS.As(ctx, &tts, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return f, diags
		}
		setString(&f.TTS.OpenAIAPIBaseURL, tts.OpenAIAPIBaseURL)
		setString(&f.TTS.OpenAIAPIKey, tts.OpenAIAPIKey)
		// A non-object value can only come from prior state (the validator
		// rejects it in configuration); it is not sent back.
		if v := tts.OpenAIParamsJSON; !v.IsNull() && !v.IsUnknown() && isJSONObject(json.RawMessage(v.ValueString())) {
			// Keep the server's raw value when it already means the same thing.
			if existing := rawJSONToState(f.TTS.OpenAIParams); existing.IsNull() || !equivalentJSON(existing.ValueString(), v.ValueString()) {
				canonical, err := canonicalJSON([]byte(v.ValueString()))
				if err != nil {
					diags.AddAttributeError(path.Root("tts").AtName("openai_params_json"), "Invalid JSON value", err.Error())
					return f, diags
				}
				f.TTS.OpenAIParams = json.RawMessage(canonical)
			}
		}
		setString(&f.TTS.APIKey, tts.APIKey)
		setString(&f.TTS.Engine, tts.Engine)
		setString(&f.TTS.Model, tts.Model)
		setString(&f.TTS.Voice, tts.Voice)
		setString(&f.TTS.SplitOn, tts.SplitOn)
		setString(&f.TTS.AzureSpeechRegion, tts.AzureSpeechRegion)
		setString(&f.TTS.AzureSpeechBaseURL, tts.AzureSpeechBaseURL)
		setString(&f.TTS.AzureSpeechOutputFormat, tts.AzureSpeechOutputFormat)
		setString(&f.TTS.MistralAPIKey, tts.MistralAPIKey)
		setString(&f.TTS.MistralAPIBaseURL, tts.MistralAPIBaseURL)
	}

	if knownObject(m.STT) {
		var stt audioSTTModel
		diags.Append(m.STT.As(ctx, &stt, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return f, diags
		}
		setList := func(dst *[]string, v types.List, name string) {
			if v.IsNull() || v.IsUnknown() {
				return
			}
			values := expandStringList(ctx, v, path.Root("stt").AtName(name), &diags)
			if values == nil {
				values = []string{}
			}
			*dst = values
		}
		setString(&f.STT.OpenAIAPIBaseURL, stt.OpenAIAPIBaseURL)
		setString(&f.STT.OpenAIAPIKey, stt.OpenAIAPIKey)
		if p := stringPtr(stt.OpenAIAPIRequestFormat); p != nil {
			if f.STT.OpenAIAPIRequestFormat == nil {
				diags.AddAttributeError(
					path.Root("stt").AtName("openai_api_request_format"),
					"Unsupported by this Open WebUI version",
					"The server does not report OPENAI_API_REQUEST_FORMAT, so it would ignore this value. The setting was added in Open WebUI v0.10.2; remove it from the configuration or upgrade Open WebUI.",
				)
				return f, diags
			}
			f.STT.OpenAIAPIRequestFormat = p
		}
		setString(&f.STT.Engine, stt.Engine)
		setString(&f.STT.Model, stt.Model)
		setList(&f.STT.SupportedContentTypes, stt.SupportedContentTypes, "supported_content_types")
		setList(&f.STT.AllowedExtensions, stt.AllowedExtensions, "allowed_extensions")
		setString(&f.STT.WhisperModel, stt.WhisperModel)
		setString(&f.STT.DeepgramAPIKey, stt.DeepgramAPIKey)
		setString(&f.STT.AzureAPIKey, stt.AzureAPIKey)
		setString(&f.STT.AzureRegion, stt.AzureRegion)
		setString(&f.STT.AzureLocales, stt.AzureLocales)
		setString(&f.STT.AzureBaseURL, stt.AzureBaseURL)
		setString(&f.STT.AzureMaxSpeakers, stt.AzureMaxSpeakers)
		setString(&f.STT.MistralAPIKey, stt.MistralAPIKey)
		setString(&f.STT.MistralAPIBaseURL, stt.MistralAPIBaseURL)
		if v := stt.MistralUseChatCompletions; !v.IsNull() && !v.IsUnknown() {
			f.STT.MistralUseChatCompletions = v.ValueBool()
		}
	}

	return f, diags
}

// audioConfigModelFromForm flattens the server config into state. Lists are
// never null (the backend's list[str] fields are not nullable), so [] stays [].
func audioConfigModelFromForm(ctx context.Context, f client.AudioConfigForm) (audioConfigModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	tts := audioTTSModel{
		OpenAIAPIBaseURL:        types.StringValue(f.TTS.OpenAIAPIBaseURL),
		OpenAIAPIKey:            types.StringValue(f.TTS.OpenAIAPIKey),
		OpenAIParamsJSON:        rawJSONToState(f.TTS.OpenAIParams),
		APIKey:                  types.StringValue(f.TTS.APIKey),
		Engine:                  types.StringValue(f.TTS.Engine),
		Model:                   types.StringValue(f.TTS.Model),
		Voice:                   types.StringValue(f.TTS.Voice),
		SplitOn:                 types.StringValue(f.TTS.SplitOn),
		AzureSpeechRegion:       types.StringValue(f.TTS.AzureSpeechRegion),
		AzureSpeechBaseURL:      types.StringValue(f.TTS.AzureSpeechBaseURL),
		AzureSpeechOutputFormat: types.StringValue(f.TTS.AzureSpeechOutputFormat),
		MistralAPIKey:           types.StringValue(f.TTS.MistralAPIKey),
		MistralAPIBaseURL:       types.StringValue(f.TTS.MistralAPIBaseURL),
	}

	contentTypes, d := flattenStringSlice(ctx, f.STT.SupportedContentTypes)
	diags.Append(d...)
	extensions, d := flattenStringSlice(ctx, f.STT.AllowedExtensions)
	diags.Append(d...)

	stt := audioSTTModel{
		OpenAIAPIBaseURL:          types.StringValue(f.STT.OpenAIAPIBaseURL),
		OpenAIAPIKey:              types.StringValue(f.STT.OpenAIAPIKey),
		OpenAIAPIRequestFormat:    stringValueOrNull(f.STT.OpenAIAPIRequestFormat),
		Engine:                    types.StringValue(f.STT.Engine),
		Model:                     types.StringValue(f.STT.Model),
		SupportedContentTypes:     contentTypes,
		AllowedExtensions:         extensions,
		WhisperModel:              types.StringValue(f.STT.WhisperModel),
		DeepgramAPIKey:            types.StringValue(f.STT.DeepgramAPIKey),
		AzureAPIKey:               types.StringValue(f.STT.AzureAPIKey),
		AzureRegion:               types.StringValue(f.STT.AzureRegion),
		AzureLocales:              types.StringValue(f.STT.AzureLocales),
		AzureBaseURL:              types.StringValue(f.STT.AzureBaseURL),
		AzureMaxSpeakers:          types.StringValue(f.STT.AzureMaxSpeakers),
		MistralAPIKey:             types.StringValue(f.STT.MistralAPIKey),
		MistralAPIBaseURL:         types.StringValue(f.STT.MistralAPIBaseURL),
		MistralUseChatCompletions: types.BoolValue(f.STT.MistralUseChatCompletions),
	}

	ttsObj, d := types.ObjectValueFrom(ctx, audioTTSAttrTypes, tts)
	diags.Append(d...)
	sttObj, d := types.ObjectValueFrom(ctx, audioSTTAttrTypes, stt)
	diags.Append(d...)

	return audioConfigModel{ID: types.StringValue("audio"), TTS: ttsObj, STT: sttObj}, diags
}

// preserveAudioConfigJSON keeps the preferred (plan or prior state) text of
// tts.openai_params_json when it is semantically equal to the server's value,
// so key order and whitespace never show up as drift.
func preserveAudioConfigJSON(ctx context.Context, preferred, fresh audioConfigModel) audioConfigModel {
	if !knownObject(preferred.TTS) || !knownObject(fresh.TTS) {
		return fresh
	}
	var pref, cur audioTTSModel
	if preferred.TTS.As(ctx, &pref, basetypes.ObjectAsOptions{}).HasError() || fresh.TTS.As(ctx, &cur, basetypes.ObjectAsOptions{}).HasError() {
		return fresh
	}
	kept := preserveEquivalentJSON(pref.OpenAIParamsJSON, cur.OpenAIParamsJSON)
	if kept.Equal(cur.OpenAIParamsJSON) {
		return fresh
	}
	cur.OpenAIParamsJSON = kept
	obj, diags := types.ObjectValueFrom(ctx, audioTTSAttrTypes, cur)
	if diags.HasError() {
		return fresh
	}
	fresh.TTS = obj
	return fresh
}
