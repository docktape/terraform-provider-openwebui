package provider

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

func audioHCL(attrs string) string {
	return `
resource "openwebui_audio_config" "test" {
` + attrs + `
}
`
}

func stockAudioForm() client.AudioConfigForm {
	format := "multipart"
	return client.AudioConfigForm{
		TTS: client.AudioTTSConfigForm{
			OpenAIAPIBaseURL:        "https://api.openai.com/v1",
			OpenAIAPIKey:            "existing-tts-key",
			OpenAIParams:            json.RawMessage(`{}`),
			Model:                   "tts-1",
			Voice:                   "alloy",
			SplitOn:                 "punctuation",
			AzureSpeechOutputFormat: "audio-24khz-160kbitrate-mono-mp3",
			MistralAPIBaseURL:       "https://api.mistral.ai/v1",
		},
		STT: client.AudioSTTConfigForm{
			OpenAIAPIBaseURL:       "https://api.openai.com/v1",
			OpenAIAPIRequestFormat: &format,
			SupportedContentTypes:  []string{},
			AllowedExtensions:      []string{"mp3", "wav"},
			WhisperModel:           "base",
			DeepgramAPIKey:         "existing-dg-key",
			MistralAPIBaseURL:      "https://api.mistral.ai/v1",
		},
	}
}

// unknownTTS returns a tts model with every attribute unknown, as planned for a
// configured `tts = {}` block on create.
func unknownTTS() audioTTSModel {
	u := types.StringUnknown()
	return audioTTSModel{u, u, u, u, u, u, u, u, u, u, u, u, u}
}

func unknownSTT() audioSTTModel {
	u := types.StringUnknown()
	l := types.ListUnknown(types.StringType)
	return audioSTTModel{
		OpenAIAPIBaseURL: u, OpenAIAPIKey: u, OpenAIAPIRequestFormat: u, Engine: u, Model: u,
		SupportedContentTypes: l, AllowedExtensions: l, WhisperModel: u, DeepgramAPIKey: u, AzureAPIKey: u,
		AzureRegion: u, AzureLocales: u, AzureBaseURL: u, AzureMaxSpeakers: u, MistralAPIKey: u,
		MistralAPIBaseURL: u, MistralUseChatCompletions: types.BoolUnknown(),
	}
}

func audioObjects(t *testing.T, tts *audioTTSModel, stt *audioSTTModel) audioConfigModel {
	t.Helper()
	m := audioConfigModel{
		ID:  types.StringUnknown(),
		TTS: types.ObjectUnknown(audioTTSAttrTypes),
		STT: types.ObjectUnknown(audioSTTAttrTypes),
	}
	if tts != nil {
		obj, diags := types.ObjectValueFrom(context.Background(), audioTTSAttrTypes, tts)
		if diags.HasError() {
			t.Fatalf("tts object: %v", diags)
		}
		m.TTS = obj
	}
	if stt != nil {
		obj, diags := types.ObjectValueFrom(context.Background(), audioSTTAttrTypes, stt)
		if diags.HasError() {
			t.Fatalf("stt object: %v", diags)
		}
		m.STT = obj
	}
	return m
}

func ttsOf(t *testing.T, m audioConfigModel) audioTTSModel {
	t.Helper()
	var out audioTTSModel
	if diags := m.TTS.As(context.Background(), &out, basetypes.ObjectAsOptions{}); diags.HasError() {
		t.Fatalf("tts as: %v", diags)
	}
	return out
}

func sttOf(t *testing.T, m audioConfigModel) audioSTTModel {
	t.Helper()
	var out audioSTTModel
	if diags := m.STT.As(context.Background(), &out, basetypes.ObjectAsOptions{}); diags.HasError() {
		t.Fatalf("stt as: %v", diags)
	}
	return out
}

// Both sub-objects omitted on create (planned as wholly unknown): the POST must
// carry the current server config unchanged.
func TestAudioOverlayUnknownObjectsKeepCurrent(t *testing.T) {
	current := stockAudioForm()
	form, diags := overlayAudioConfigPlan(context.Background(), current, audioObjects(t, nil, nil))
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if !reflect.DeepEqual(form, current) {
		t.Fatalf("expected the current config unchanged\nwant %+v\ngot  %+v", current, form)
	}

	// Null objects (e.g. after import) behave the same.
	m := audioConfigModel{ID: types.StringNull(), TTS: types.ObjectNull(audioTTSAttrTypes), STT: types.ObjectNull(audioSTTAttrTypes)}
	form, diags = overlayAudioConfigPlan(context.Background(), current, m)
	if diags.HasError() || !reflect.DeepEqual(form, current) {
		t.Fatalf("null objects must keep the current config: %v %+v", diags, form)
	}

	// Configured but empty blocks (every nested attribute unknown) too.
	tts, stt := unknownTTS(), unknownSTT()
	form, diags = overlayAudioConfigPlan(context.Background(), current, audioObjects(t, &tts, &stt))
	if diags.HasError() || !reflect.DeepEqual(form, current) {
		t.Fatalf("all-unknown nested attributes must keep the current config: %v %+v", diags, form)
	}
}

// A stored non-object OPENAI_PARAMS (the GET has no response model) is sent
// back as null rather than echoed into a 422; an object is kept as is.
func TestAudioOverlayNonObjectOpenAIParamsBecomesNull(t *testing.T) {
	for _, raw := range []string{`"speed=1"`, `[1, 2]`, `3`, `true`, `null`} {
		current := stockAudioForm()
		current.TTS.OpenAIParams = json.RawMessage(raw)
		form, diags := overlayAudioConfigPlan(context.Background(), current, audioObjects(t, nil, nil))
		if diags.HasError() {
			t.Fatalf("%s: unexpected diags: %v", raw, diags)
		}
		if form.TTS.OpenAIParams != nil {
			t.Errorf("%s: expected nil OPENAI_PARAMS, got %s", raw, form.TTS.OpenAIParams)
		}
		// A non-object value coming from prior state is not sent either.
		tts := unknownTTS()
		tts.OpenAIParamsJSON = types.StringValue("speed=1")
		form, diags = overlayAudioConfigPlan(context.Background(), current, audioObjects(t, &tts, nil))
		if diags.HasError() || form.TTS.OpenAIParams != nil {
			t.Errorf("%s: state-derived non-object params must not be sent: %v %s", raw, diags, form.TTS.OpenAIParams)
		}
	}
	current := stockAudioForm()
	current.TTS.OpenAIParams = json.RawMessage(`{"speed": 1.50}`)
	form, _ := overlayAudioConfigPlan(context.Background(), current, audioObjects(t, nil, nil))
	if string(form.TTS.OpenAIParams) != `{"speed": 1.50}` {
		t.Errorf("object params must be kept verbatim, got %s", form.TTS.OpenAIParams)
	}
}

// Only tts configured: tts values change, stt is sent back untouched.
func TestAudioOverlayOnlyTTS(t *testing.T) {
	tts := unknownTTS()
	tts.Engine = types.StringValue("openai")
	tts.OpenAIAPIKey = types.StringValue("")
	tts.OpenAIParamsJSON = types.StringValue(`{ "speed": 1.25, "a": [1] }`)
	tts.SplitOn = types.StringValue("none")

	current := stockAudioForm()
	form, diags := overlayAudioConfigPlan(context.Background(), current, audioObjects(t, &tts, nil))
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if form.TTS.Engine != "openai" || form.TTS.OpenAIAPIKey != "" || form.TTS.SplitOn != "none" {
		t.Errorf("known tts values must be sent: %+v", form.TTS)
	}
	if string(form.TTS.OpenAIParams) != `{"a":[1],"speed":1.25}` {
		t.Errorf("expected canonical params, got %s", form.TTS.OpenAIParams)
	}
	if form.TTS.Model != "tts-1" || form.TTS.MistralAPIBaseURL != "https://api.mistral.ai/v1" {
		t.Errorf("unknown tts attributes must keep server values: %+v", form.TTS)
	}
	if !reflect.DeepEqual(form.STT, current.STT) {
		t.Errorf("stt must be unchanged\nwant %+v\ngot  %+v", current.STT, form.STT)
	}
}

// Only stt configured, covering the list shapes: [] clears, populated sets,
// unknown keeps the server value.
func TestAudioOverlayOnlySTTLists(t *testing.T) {
	stt := unknownSTT()
	stt.Engine = types.StringValue("openai")
	stt.MistralUseChatCompletions = types.BoolValue(true)
	stt.AllowedExtensions = types.ListValueMust(types.StringType, []attr.Value{})
	stt.AzureMaxSpeakers = types.StringValue("2")

	current := stockAudioForm()
	form, diags := overlayAudioConfigPlan(context.Background(), current, audioObjects(t, nil, &stt))
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if form.STT.Engine != "openai" || !form.STT.MistralUseChatCompletions || form.STT.AzureMaxSpeakers != "2" {
		t.Errorf("known stt values must be sent: %+v", form.STT)
	}
	if form.STT.AllowedExtensions == nil || len(form.STT.AllowedExtensions) != 0 {
		t.Errorf("[] must be sent as an empty list, got %#v", form.STT.AllowedExtensions)
	}
	if !reflect.DeepEqual(form.STT.SupportedContentTypes, []string{}) || form.STT.DeepgramAPIKey != "existing-dg-key" {
		t.Errorf("unknown stt attributes must keep server values: %+v", form.STT)
	}
	if !reflect.DeepEqual(form.TTS, current.TTS) {
		t.Errorf("tts must be unchanged")
	}

	stt.SupportedContentTypes = types.ListValueMust(types.StringType, []attr.Value{types.StringValue("audio/*"), types.StringValue("video/webm")})
	form, diags = overlayAudioConfigPlan(context.Background(), current, audioObjects(t, nil, &stt))
	if diags.HasError() || !reflect.DeepEqual(form.STT.SupportedContentTypes, []string{"audio/*", "video/webm"}) {
		t.Errorf("populated list must be sent: %v %#v", diags, form.STT.SupportedContentTypes)
	}
}

// Setting openai_api_request_format against a server that does not report it
// (Open WebUI before v0.10.2) must fail instead of silently being ignored.
func TestAudioOverlayRequestFormatNeedsNewerServer(t *testing.T) {
	stt := unknownSTT()
	stt.OpenAIAPIRequestFormat = types.StringValue("json")

	current := stockAudioForm()
	form, diags := overlayAudioConfigPlan(context.Background(), current, audioObjects(t, nil, &stt))
	if diags.HasError() || form.STT.OpenAIAPIRequestFormat == nil || *form.STT.OpenAIAPIRequestFormat != "json" {
		t.Fatalf("expected request format to be sent: %v %v", diags, form.STT.OpenAIAPIRequestFormat)
	}

	current.STT.OpenAIAPIRequestFormat = nil
	_, diags = overlayAudioConfigPlan(context.Background(), current, audioObjects(t, nil, &stt))
	if !diags.HasError() {
		t.Fatalf("expected an error for a pre-v0.10.2 server")
	}
}

func TestAudioModelFromFormShapes(t *testing.T) {
	ctx := context.Background()
	f := stockAudioForm()
	m, diags := audioConfigModelFromForm(ctx, f)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if m.ID.ValueString() != "audio" {
		t.Errorf("unexpected id %q", m.ID.ValueString())
	}
	tts, stt := ttsOf(t, m), sttOf(t, m)
	if tts.OpenAIParamsJSON.ValueString() != "{}" || tts.OpenAIAPIKey.ValueString() != "existing-tts-key" || tts.Model.ValueString() != "tts-1" {
		t.Errorf("unexpected tts %+v", tts)
	}
	if stt.OpenAIAPIRequestFormat.ValueString() != "multipart" || stt.DeepgramAPIKey.ValueString() != "existing-dg-key" {
		t.Errorf("unexpected stt %+v", stt)
	}
	if stt.SupportedContentTypes.IsNull() || len(stt.SupportedContentTypes.Elements()) != 0 {
		t.Errorf("[] must stay an empty (not null) list, got %v", stt.SupportedContentTypes)
	}
	if len(stt.AllowedExtensions.Elements()) != 2 {
		t.Errorf("unexpected extensions %v", stt.AllowedExtensions)
	}

	// null params, nil lists, absent request format (pre-v0.10.2).
	f.TTS.OpenAIParams = json.RawMessage(`null`)
	f.STT.SupportedContentTypes = nil
	f.STT.OpenAIAPIRequestFormat = nil
	f.TTS.OpenAIParams = json.RawMessage(`{"z": 1, "a": 2}`)
	m, diags = audioConfigModelFromForm(ctx, f)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	tts, stt = ttsOf(t, m), sttOf(t, m)
	if tts.OpenAIParamsJSON.ValueString() != `{"a":2,"z":1}` {
		t.Errorf("expected canonical params, got %v", tts.OpenAIParamsJSON)
	}
	if !stt.OpenAIAPIRequestFormat.IsNull() {
		t.Errorf("absent request format must be null, got %v", stt.OpenAIAPIRequestFormat)
	}
	if stt.SupportedContentTypes.IsNull() || len(stt.SupportedContentTypes.Elements()) != 0 {
		t.Errorf("nil list must become [], got %v", stt.SupportedContentTypes)
	}
	f.TTS.OpenAIParams = json.RawMessage(`null`)
	m, _ = audioConfigModelFromForm(ctx, f)
	if !ttsOf(t, m).OpenAIParamsJSON.IsNull() {
		t.Errorf("null params must be null in state")
	}
}

// A state-derived plan sends the same form back (no accidental rewrites).
func TestAudioStateRoundTrip(t *testing.T) {
	current := stockAudioForm()
	m, diags := audioConfigModelFromForm(context.Background(), current)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	form, diags := overlayAudioConfigPlan(context.Background(), current, m)
	if diags.HasError() || !reflect.DeepEqual(form, current) {
		t.Fatalf("round trip changed the form: %v\nwant %+v\ngot  %+v", diags, current, form)
	}
}

func TestAudioPreserveUserJSONFormatting(t *testing.T) {
	ctx := context.Background()
	fresh, _ := audioConfigModelFromForm(ctx, stockAudioForm())

	prior := ttsOf(t, fresh)
	prior.OpenAIParamsJSON = types.StringValue(`{ }`)
	preferred := audioObjects(t, &prior, nil)
	got := preserveAudioConfigJSON(ctx, preferred, fresh)
	if v := ttsOf(t, got).OpenAIParamsJSON.ValueString(); v != `{ }` {
		t.Errorf("equivalent JSON must keep the user's text, got %q", v)
	}

	prior.OpenAIParamsJSON = types.StringValue(`{"speed": 2}`)
	preferred = audioObjects(t, &prior, nil)
	got = preserveAudioConfigJSON(ctx, preferred, fresh)
	if v := ttsOf(t, got).OpenAIParamsJSON.ValueString(); v != `{}` {
		t.Errorf("different JSON must take the server value, got %q", v)
	}

	// Unknown preferred objects (create without a tts block) leave fresh alone.
	got = preserveAudioConfigJSON(ctx, audioObjects(t, nil, nil), fresh)
	if !got.TTS.Equal(fresh.TTS) || !got.STT.Equal(fresh.STT) {
		t.Errorf("unknown preferred objects must not change fresh")
	}
}

func TestAudioConfigPlanShapes(t *testing.T) {
	planOnlyValid(t, audioHCL(``))
	planOnlyValid(t, audioHCL(`  tts = { engine = "openai" }`))
	planOnlyValid(t, audioHCL(`  stt = { engine = "openai", supported_content_types = [] }`))
	planOnlyValid(t, audioHCL(`  stt = { allowed_extensions = ["mp3"] }
  tts = { openai_params_json = jsonencode({ speed = 1.2 }) }`))

	for _, ok := range []string{`{}`, `{\"speed\": 1}`} {
		planOnlyValid(t, audioHCL(`  tts = { openai_params_json = "`+ok+`" }`))
	}
	for _, bad := range []string{``, `null`, `[]`, `{`, `\"text\"`} {
		planOnlyRejected(t, audioHCL(`  tts = { openai_params_json = "`+bad+`" }`), `(?s)openai_params_json.*JSON\s+object`)
	}
}

func TestAudioConfigSchemaShape(t *testing.T) {
	var resp resource.SchemaResponse
	NewAudioConfigResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if len(resp.Schema.Attributes) != 3 {
		t.Fatalf("expected id, tts and stt, got %d attributes", len(resp.Schema.Attributes))
	}
	sensitive := map[string]bool{
		"tts.openai_api_key": true, "tts.api_key": true, "tts.mistral_api_key": true,
		"stt.openai_api_key": true, "stt.deepgram_api_key": true, "stt.azure_api_key": true, "stt.mistral_api_key": true,
	}
	wantTypes := map[string]map[string]attr.Type{"tts": audioTTSAttrTypes, "stt": audioSTTAttrTypes}
	wantCounts := map[string]int{"tts": 13, "stt": 17}
	for name, a := range resp.Schema.Attributes {
		if a.GetDescription() == "" || a.GetMarkdownDescription() == "" {
			t.Errorf("attribute %q needs both Description and MarkdownDescription", name)
		}
		if name == "id" {
			continue
		}
		nested, ok := a.(schema.SingleNestedAttribute)
		if !ok {
			t.Fatalf("%q must be a SingleNestedAttribute", name)
		}
		if !nested.IsOptional() || !nested.IsComputed() || nested.IsSensitive() {
			t.Errorf("%q must be Optional+Computed and not Sensitive as a whole", name)
		}
		if len(nested.Attributes) != wantCounts[name] {
			t.Errorf("%q: expected %d nested attributes, got %d", name, wantCounts[name], len(nested.Attributes))
		}
		objType, ok := nested.GetType().(types.ObjectType)
		if !ok || !reflect.DeepEqual(objType.AttrTypes, wantTypes[name]) {
			t.Errorf("%q: schema types do not match the attr type map", name)
		}
		for child, ca := range nested.Attributes {
			full := name + "." + child
			if ca.GetDescription() == "" || ca.GetMarkdownDescription() == "" {
				t.Errorf("attribute %q needs both Description and MarkdownDescription", full)
			}
			if ca.IsSensitive() != sensitive[full] {
				t.Errorf("attribute %q: unexpected Sensitive=%v", full, ca.IsSensitive())
			}
			if !ca.IsOptional() || !ca.IsComputed() {
				t.Errorf("attribute %q must be Optional+Computed", full)
			}
		}
	}
}
