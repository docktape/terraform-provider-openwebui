package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"testing"
)

// stockAudioResponse mirrors a stock Open WebUI v0.11.0 GET /audio/config
// response (secrets blank).
const stockAudioResponse = `{
	"tts": {
		"OPENAI_API_BASE_URL": "https://api.openai.com/v1", "OPENAI_API_KEY": "", "OPENAI_PARAMS": {},
		"API_KEY": "", "ENGINE": "", "MODEL": "tts-1", "VOICE": "alloy", "SPLIT_ON": "punctuation",
		"AZURE_SPEECH_REGION": "", "AZURE_SPEECH_BASE_URL": "",
		"AZURE_SPEECH_OUTPUT_FORMAT": "audio-24khz-160kbitrate-mono-mp3",
		"MISTRAL_API_KEY": "", "MISTRAL_API_BASE_URL": "https://api.mistral.ai/v1"
	},
	"stt": {
		"OPENAI_API_BASE_URL": "https://api.openai.com/v1", "OPENAI_API_KEY": "", "OPENAI_API_REQUEST_FORMAT": "multipart",
		"ENGINE": "", "MODEL": "", "SUPPORTED_CONTENT_TYPES": [],
		"ALLOWED_EXTENSIONS": ["mp3", "wav", "m4a"],
		"WHISPER_MODEL": "base", "DEEPGRAM_API_KEY": "", "AZURE_API_KEY": "", "AZURE_REGION": "",
		"AZURE_LOCALES": "", "AZURE_BASE_URL": "", "AZURE_MAX_SPEAKERS": "",
		"MISTRAL_API_KEY": "", "MISTRAL_API_BASE_URL": "https://api.mistral.ai/v1",
		"MISTRAL_USE_CHAT_COMPLETIONS": false
	}
}`

var audioTTSWireKeys = []string{
	"OPENAI_API_BASE_URL", "OPENAI_API_KEY", "OPENAI_PARAMS", "API_KEY", "ENGINE", "MODEL", "VOICE", "SPLIT_ON",
	"AZURE_SPEECH_REGION", "AZURE_SPEECH_BASE_URL", "AZURE_SPEECH_OUTPUT_FORMAT", "MISTRAL_API_KEY", "MISTRAL_API_BASE_URL",
}

var audioSTTWireKeys = []string{
	"OPENAI_API_BASE_URL", "OPENAI_API_KEY", "OPENAI_API_REQUEST_FORMAT", "ENGINE", "MODEL", "SUPPORTED_CONTENT_TYPES",
	"ALLOWED_EXTENSIONS", "WHISPER_MODEL", "DEEPGRAM_API_KEY", "AZURE_API_KEY", "AZURE_REGION", "AZURE_LOCALES",
	"AZURE_BASE_URL", "AZURE_MAX_SPEAKERS", "MISTRAL_API_KEY", "MISTRAL_API_BASE_URL", "MISTRAL_USE_CHAT_COMPLETIONS",
}

func TestGetAudioConfig(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/audio/config" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(stockAudioResponse))
	})
	c, server := newLDAPConfigTestClient(handler)
	defer server.Close()

	got, err := c.GetAudioConfig(context.Background())
	if err != nil {
		t.Fatalf("GetAudioConfig failed: %v", err)
	}
	if got.TTS.Model != "tts-1" || got.TTS.Voice != "alloy" || got.TTS.SplitOn != "punctuation" ||
		got.TTS.AzureSpeechOutputFormat != "audio-24khz-160kbitrate-mono-mp3" || got.TTS.MistralAPIBaseURL != "https://api.mistral.ai/v1" {
		t.Errorf("unexpected tts: %+v", got.TTS)
	}
	if string(got.TTS.OpenAIParams) != "{}" {
		t.Errorf("expected raw {} params, got %s", got.TTS.OpenAIParams)
	}
	if got.STT.OpenAIAPIRequestFormat == nil || *got.STT.OpenAIAPIRequestFormat != "multipart" {
		t.Errorf("unexpected request format %v", got.STT.OpenAIAPIRequestFormat)
	}
	if got.STT.SupportedContentTypes == nil || len(got.STT.SupportedContentTypes) != 0 {
		t.Errorf("expected empty non-nil content types, got %#v", got.STT.SupportedContentTypes)
	}
	if !reflect.DeepEqual(got.STT.AllowedExtensions, []string{"mp3", "wav", "m4a"}) {
		t.Errorf("unexpected extensions %#v", got.STT.AllowedExtensions)
	}
	if got.STT.WhisperModel != "base" || got.STT.MistralUseChatCompletions || got.STT.AzureMaxSpeakers != "" {
		t.Errorf("unexpected stt: %+v", got.STT)
	}
}

// Pre-v0.10.2 servers have no OPENAI_API_REQUEST_FORMAT key; AZURE_MAX_SPEAKERS
// is decoded leniently in case a number was stored; OPENAI_PARAMS may be null.
func TestGetAudioConfigOlderAndLenientShapes(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"tts": {"OPENAI_PARAMS": null}, "stt": {"AZURE_MAX_SPEAKERS": 4, "SUPPORTED_CONTENT_TYPES": ["audio/*"]}}`))
	})
	c, server := newLDAPConfigTestClient(handler)
	defer server.Close()

	got, err := c.GetAudioConfig(context.Background())
	if err != nil {
		t.Fatalf("GetAudioConfig failed: %v", err)
	}
	if got.STT.OpenAIAPIRequestFormat != nil {
		t.Errorf("absent request format must stay nil, got %v", *got.STT.OpenAIAPIRequestFormat)
	}
	if got.STT.AzureMaxSpeakers != "4" {
		t.Errorf("expected numeric max speakers as \"4\", got %q", got.STT.AzureMaxSpeakers)
	}
	if string(got.TTS.OpenAIParams) != "null" {
		t.Errorf("expected raw null params, got %q", got.TTS.OpenAIParams)
	}
	if !reflect.DeepEqual(got.STT.SupportedContentTypes, []string{"audio/*"}) {
		t.Errorf("unexpected content types %#v", got.STT.SupportedContentTypes)
	}
}

func TestSetAudioConfigEveryWireKey(t *testing.T) {
	var body map[string]map[string]any
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/audio/config/update" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		raw, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		_, _ = w.Write([]byte(stockAudioResponse))
	})
	c, server := newLDAPConfigTestClient(handler)
	defer server.Close()

	format := "json"
	form := AudioConfigForm{
		TTS: AudioTTSConfigForm{
			OpenAIAPIBaseURL: "https://tts.example/v1", OpenAIAPIKey: "tts-openai-key", OpenAIParams: json.RawMessage(`{"speed":1.5}`),
			APIKey: "eleven-key", Engine: "openai", Model: "tts-1-hd", Voice: "nova", SplitOn: "paragraphs",
			AzureSpeechRegion: "westeurope", AzureSpeechBaseURL: "https://azure.example", AzureSpeechOutputFormat: "riff-16khz-16bit-mono-pcm",
			MistralAPIKey: "tts-mistral-key", MistralAPIBaseURL: "https://mistral.example/v1",
		},
		STT: AudioSTTConfigForm{
			OpenAIAPIBaseURL: "https://stt.example/v1", OpenAIAPIKey: "stt-openai-key", OpenAIAPIRequestFormat: &format,
			Engine: "openai", Model: "whisper-1", SupportedContentTypes: []string{"audio/*"}, AllowedExtensions: []string{"mp3"},
			WhisperModel: "base", DeepgramAPIKey: "dg-key", AzureAPIKey: "az-key", AzureRegion: "eastus",
			AzureLocales: "en-US,de-DE", AzureBaseURL: "https://az-stt.example", AzureMaxSpeakers: "3",
			MistralAPIKey: "stt-mistral-key", MistralAPIBaseURL: "https://mistral-stt.example/v1", MistralUseChatCompletions: true,
		},
	}
	if _, err := c.SetAudioConfig(context.Background(), form); err != nil {
		t.Fatalf("SetAudioConfig failed: %v", err)
	}

	if len(body) != 2 || body["tts"] == nil || body["stt"] == nil {
		t.Fatalf("expected exactly tts and stt objects, got %v", body)
	}
	for _, k := range audioTTSWireKeys {
		if _, ok := body["tts"][k]; !ok {
			t.Errorf("tts key %s missing", k)
		}
	}
	for _, k := range audioSTTWireKeys {
		if _, ok := body["stt"][k]; !ok {
			t.Errorf("stt key %s missing", k)
		}
	}
	if len(body["tts"]) != len(audioTTSWireKeys) || len(body["stt"]) != len(audioSTTWireKeys) {
		t.Errorf("unexpected extra keys: tts=%d stt=%d", len(body["tts"]), len(body["stt"]))
	}
	tts, stt := body["tts"], body["stt"]
	if tts["OPENAI_API_KEY"] != "tts-openai-key" || tts["API_KEY"] != "eleven-key" || tts["SPLIT_ON"] != "paragraphs" ||
		tts["AZURE_SPEECH_OUTPUT_FORMAT"] != "riff-16khz-16bit-mono-pcm" || tts["MISTRAL_API_KEY"] != "tts-mistral-key" {
		t.Errorf("unexpected tts body %v", tts)
	}
	if params, ok := tts["OPENAI_PARAMS"].(map[string]any); !ok || params["speed"] != 1.5 {
		t.Errorf("unexpected OPENAI_PARAMS %v", tts["OPENAI_PARAMS"])
	}
	if stt["OPENAI_API_REQUEST_FORMAT"] != "json" || stt["DEEPGRAM_API_KEY"] != "dg-key" || stt["AZURE_API_KEY"] != "az-key" ||
		stt["AZURE_LOCALES"] != "en-US,de-DE" || stt["AZURE_MAX_SPEAKERS"] != "3" || stt["MISTRAL_USE_CHAT_COMPLETIONS"] != true {
		t.Errorf("unexpected stt body %v", stt)
	}
	if !reflect.DeepEqual(stt["SUPPORTED_CONTENT_TYPES"], []any{"audio/*"}) || !reflect.DeepEqual(stt["ALLOWED_EXTENSIONS"], []any{"mp3"}) {
		t.Errorf("unexpected lists %v / %v", stt["SUPPORTED_CONTENT_TYPES"], stt["ALLOWED_EXTENSIONS"])
	}
}

// Nil lists must go out as [] (list[str] is not nullable), nil params as null
// (Optional[dict]), and a nil request format is omitted so the backend default
// applies on v0.10.2+ and older servers never see the key.
func TestSetAudioConfigZeroValues(t *testing.T) {
	var body map[string]map[string]any
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		_, _ = w.Write([]byte(stockAudioResponse))
	})
	c, server := newLDAPConfigTestClient(handler)
	defer server.Close()

	if _, err := c.SetAudioConfig(context.Background(), AudioConfigForm{}); err != nil {
		t.Fatalf("SetAudioConfig failed: %v", err)
	}
	if v, ok := body["tts"]["OPENAI_PARAMS"]; !ok || v != nil {
		t.Errorf("expected explicit null OPENAI_PARAMS, got %v (present=%v)", v, ok)
	}
	for _, k := range []string{"SUPPORTED_CONTENT_TYPES", "ALLOWED_EXTENSIONS"} {
		if !reflect.DeepEqual(body["stt"][k], []any{}) {
			t.Errorf("expected [] for %s, got %#v", k, body["stt"][k])
		}
	}
	if _, ok := body["stt"]["OPENAI_API_REQUEST_FORMAT"]; ok {
		t.Errorf("nil OPENAI_API_REQUEST_FORMAT must be omitted")
	}
	if body["stt"]["MISTRAL_USE_CHAT_COMPLETIONS"] != false || body["tts"]["ENGINE"] != "" {
		t.Errorf("zero scalars must be sent explicitly: %v", body)
	}
}
