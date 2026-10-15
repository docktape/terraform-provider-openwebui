package client

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

// stockImagesResponse mirrors a stock Open WebUI v0.11.0 GET /images/config
// response (secrets blank, COMFYUI_WORKFLOW shortened), plus the extra
// USER_PERMISSIONS key the POST handler echoes because it returns the raw
// get_config_values() dict without a response_model.
const stockImagesResponse = `{
	"ENABLE_IMAGE_GENERATION": false, "ENABLE_IMAGE_PROMPT_GENERATION": true,
	"IMAGE_GENERATION_ENGINE": "openai", "IMAGE_GENERATION_MODEL": "", "IMAGE_SIZE": "512x512", "IMAGE_STEPS": 50,
	"IMAGES_OPENAI_API_BASE_URL": "https://api.openai.com/v1", "IMAGES_OPENAI_API_KEY": "", "IMAGES_OPENAI_API_VERSION": "",
	"IMAGES_OPENAI_API_PARAMS": {},
	"AUTOMATIC1111_BASE_URL": "", "AUTOMATIC1111_API_AUTH": "", "AUTOMATIC1111_PARAMS": {},
	"COMFYUI_BASE_URL": "", "COMFYUI_API_KEY": "", "COMFYUI_WORKFLOW": "{\"3\": {}}", "COMFYUI_WORKFLOW_NODES": [],
	"IMAGES_GEMINI_API_BASE_URL": "", "IMAGES_GEMINI_API_KEY": "", "IMAGES_GEMINI_ENDPOINT_METHOD": "",
	"ENABLE_IMAGE_EDIT": false, "IMAGE_EDIT_ENGINE": "openai", "IMAGE_EDIT_MODEL": "", "IMAGE_EDIT_SIZE": "",
	"IMAGES_EDIT_OPENAI_API_BASE_URL": "https://api.openai.com/v1", "IMAGES_EDIT_OPENAI_API_KEY": "",
	"IMAGES_EDIT_OPENAI_API_VERSION": "", "IMAGES_EDIT_GEMINI_API_BASE_URL": "", "IMAGES_EDIT_GEMINI_API_KEY": "",
	"IMAGES_EDIT_COMFYUI_BASE_URL": "", "IMAGES_EDIT_COMFYUI_API_KEY": "", "IMAGES_EDIT_COMFYUI_WORKFLOW": "",
	"IMAGES_EDIT_COMFYUI_WORKFLOW_NODES": [],
	"USER_PERMISSIONS": {"chat": {"file_upload": true}}
}`

func TestGetImagesConfig(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/images/config" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(stockImagesResponse))
	})
	c, server := newConfigTestClient(handler)
	defer server.Close()

	got, err := c.GetImagesConfig(context.Background())
	if err != nil {
		t.Fatalf("GetImagesConfig failed: %v", err)
	}
	if got.EnableImageGeneration || !got.EnableImagePromptGeneration || got.ImageGenerationEngine != "openai" {
		t.Errorf("unexpected scalars: %+v", got)
	}
	if got.ImageSize == nil || *got.ImageSize != "512x512" || got.ImageSteps == nil || *got.ImageSteps != 50 {
		t.Errorf("unexpected size/steps: %v / %v", got.ImageSize, got.ImageSteps)
	}
	if got.ImageEditSize == nil || *got.ImageEditSize != "" {
		t.Errorf("expected empty IMAGE_EDIT_SIZE, got %v", got.ImageEditSize)
	}
	if got.Automatic1111APIAuth == nil || *got.Automatic1111APIAuth != "" {
		t.Errorf("expected empty AUTOMATIC1111_API_AUTH, got %v", got.Automatic1111APIAuth)
	}
	if string(got.ImagesOpenAIAPIParams) != "{}" || string(got.Automatic1111Params) != "{}" {
		t.Errorf("expected raw {} params, got %s / %s", got.ImagesOpenAIAPIParams, got.Automatic1111Params)
	}
	if string(got.ComfyUIWorkflowNodes) != "[]" || string(got.ImagesEditComfyUIWorkflowNodes) != "[]" {
		t.Errorf("expected raw [] nodes, got %s / %s", got.ComfyUIWorkflowNodes, got.ImagesEditComfyUIWorkflowNodes)
	}
	if got.ComfyUIWorkflow != `{"3": {}}` {
		t.Errorf("expected workflow string verbatim, got %q", got.ComfyUIWorkflow)
	}
}

// TestGetImagesConfigNullableShapes covers the nullable/union fields: null
// IMAGE_SIZE/IMAGE_STEPS/IMAGE_EDIT_SIZE, IMAGE_STEPS as "" (the
// os.getenv(...,”) idiom, defensively) or a numeric string,
// AUTOMATIC1111_API_AUTH as null or a dict (dict|str|None server-side), and
// the params as null or a legacy JSON string.
func TestGetImagesConfigNullableShapes(t *testing.T) {
	cases := []struct {
		name, body string
		check      func(t *testing.T, f *ImagesConfigForm)
	}{
		{"nulls", `{"IMAGE_SIZE": null, "IMAGE_STEPS": null, "IMAGE_EDIT_SIZE": null, "AUTOMATIC1111_API_AUTH": null, "IMAGES_OPENAI_API_PARAMS": null}`,
			func(t *testing.T, f *ImagesConfigForm) {
				if f.ImageSize != nil || f.ImageSteps != nil || f.ImageEditSize != nil || f.Automatic1111APIAuth != nil {
					t.Errorf("expected nil pointers, got %+v", f)
				}
				if string(f.ImagesOpenAIAPIParams) != "null" {
					t.Errorf("expected raw null params, got %s", f.ImagesOpenAIAPIParams)
				}
			}},
		{"empty-string steps", `{"IMAGE_STEPS": ""}`, func(t *testing.T, f *ImagesConfigForm) {
			if f.ImageSteps != nil {
				t.Errorf("expected nil steps for \"\", got %v", *f.ImageSteps)
			}
		}},
		{"numeric-string steps", `{"IMAGE_STEPS": "30"}`, func(t *testing.T, f *ImagesConfigForm) {
			if f.ImageSteps == nil || *f.ImageSteps != 30 {
				t.Errorf("expected 30, got %v", f.ImageSteps)
			}
		}},
		{"string auth and string params", `{"AUTOMATIC1111_API_AUTH": "user:pass", "AUTOMATIC1111_PARAMS": "{\"cfg_scale\": 7}"}`,
			func(t *testing.T, f *ImagesConfigForm) {
				if f.Automatic1111APIAuth == nil || *f.Automatic1111APIAuth != "user:pass" {
					t.Errorf("expected user:pass, got %v", f.Automatic1111APIAuth)
				}
				if string(f.Automatic1111Params) != `"{\"cfg_scale\": 7}"` {
					t.Errorf("expected the JSON string kept raw, got %s", f.Automatic1111Params)
				}
			}},
		{"dict auth", `{"AUTOMATIC1111_API_AUTH": {"u": "p"}}`, func(t *testing.T, f *ImagesConfigForm) {
			if f.Automatic1111APIAuth == nil || *f.Automatic1111APIAuth != `{"u": "p"}` {
				t.Errorf("expected the raw dict text, got %v", f.Automatic1111APIAuth)
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tc.body))
			})
			c, server := newConfigTestClient(handler)
			defer server.Close()
			got, err := c.GetImagesConfig(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			tc.check(t, got)
		})
	}
}

// TestSetImagesConfigEveryWireKey pins all 33 literal wire keys of the
// ImagesConfig Pydantic model. Every field is required by the model (no
// defaults), so the body must always carry all of them: nil pointers are sent
// as explicit null, never omitted, and the raw JSON fields pass through
// unchanged.
func TestSetImagesConfigEveryWireKey(t *testing.T) {
	var body map[string]any
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/images/config/update" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(stockImagesResponse))
	})
	c, server := newConfigTestClient(handler)
	defer server.Close()

	s := func(v string) *string { return &v }
	steps := int64(25)
	form := ImagesConfigForm{
		EnableImageGeneration: true, EnableImagePromptGeneration: false,
		ImageGenerationEngine: "comfyui", ImageGenerationModel: "sdxl", ImageSize: s("1024x1024"), ImageSteps: &steps,
		ImagesOpenAIAPIBaseURL: "https://oai", ImagesOpenAIAPIKey: "oai-key", ImagesOpenAIAPIVersion: "2024-01",
		ImagesOpenAIAPIParams: json.RawMessage(`{"quality":"hd"}`),
		Automatic1111BaseURL:  "http://a1111", Automatic1111APIAuth: s("user:pass"),
		Automatic1111Params: json.RawMessage(`null`),
		ComfyUIBaseURL:      "http://comfy", ComfyUIAPIKey: "comfy-key", ComfyUIWorkflow: `{"1":{}}`,
		ComfyUIWorkflowNodes:   json.RawMessage(`[{"type":"prompt","key":"text","node_ids":["6"]}]`),
		ImagesGeminiAPIBaseURL: "https://gem", ImagesGeminiAPIKey: "gem-key", ImagesGeminiEndpointMethod: "predict",
		EnableImageEdit: true, ImageEditEngine: "gemini", ImageEditModel: "edit-m", ImageEditSize: nil,
		ImagesEditOpenAIAPIBaseURL: "https://oai-e", ImagesEditOpenAIAPIKey: "oai-e-key", ImagesEditOpenAIAPIVersion: "v",
		ImagesEditGeminiAPIBaseURL: "https://gem-e", ImagesEditGeminiAPIKey: "gem-e-key",
		ImagesEditComfyUIBaseURL: "http://comfy-e", ImagesEditComfyUIAPIKey: "comfy-e-key", ImagesEditComfyUIWorkflow: "",
		ImagesEditComfyUIWorkflowNodes: json.RawMessage(`[]`),
	}
	if _, err := c.SetImagesConfig(context.Background(), form); err != nil {
		t.Fatalf("SetImagesConfig failed: %v", err)
	}

	want := map[string]any{
		"ENABLE_IMAGE_GENERATION": true, "ENABLE_IMAGE_PROMPT_GENERATION": false,
		"IMAGE_GENERATION_ENGINE": "comfyui", "IMAGE_GENERATION_MODEL": "sdxl", "IMAGE_SIZE": "1024x1024", "IMAGE_STEPS": float64(25),
		"IMAGES_OPENAI_API_BASE_URL": "https://oai", "IMAGES_OPENAI_API_KEY": "oai-key", "IMAGES_OPENAI_API_VERSION": "2024-01",
		"AUTOMATIC1111_BASE_URL": "http://a1111", "AUTOMATIC1111_API_AUTH": "user:pass", "AUTOMATIC1111_PARAMS": nil,
		"COMFYUI_BASE_URL": "http://comfy", "COMFYUI_API_KEY": "comfy-key", "COMFYUI_WORKFLOW": `{"1":{}}`,
		"IMAGES_GEMINI_API_BASE_URL": "https://gem", "IMAGES_GEMINI_API_KEY": "gem-key", "IMAGES_GEMINI_ENDPOINT_METHOD": "predict",
		"ENABLE_IMAGE_EDIT": true, "IMAGE_EDIT_ENGINE": "gemini", "IMAGE_EDIT_MODEL": "edit-m", "IMAGE_EDIT_SIZE": nil,
		"IMAGES_EDIT_OPENAI_API_BASE_URL": "https://oai-e", "IMAGES_EDIT_OPENAI_API_KEY": "oai-e-key", "IMAGES_EDIT_OPENAI_API_VERSION": "v",
		"IMAGES_EDIT_GEMINI_API_BASE_URL": "https://gem-e", "IMAGES_EDIT_GEMINI_API_KEY": "gem-e-key",
		"IMAGES_EDIT_COMFYUI_BASE_URL": "http://comfy-e", "IMAGES_EDIT_COMFYUI_API_KEY": "comfy-e-key", "IMAGES_EDIT_COMFYUI_WORKFLOW": "",
	}
	// The three structured keys are compared separately below.
	if len(want)+3 != 33 {
		t.Fatalf("test table must cover 33 keys, has %d", len(want)+3)
	}
	if len(body) != 33 {
		t.Errorf("expected 33 keys on the wire, got %d: %v", len(body), body)
	}
	for k, v := range want {
		if got, ok := body[k]; !ok || got != v {
			t.Errorf("wire key %q: expected %#v, got %#v (present=%v)", k, v, got, ok)
		}
	}
	if params, ok := body["IMAGES_OPENAI_API_PARAMS"].(map[string]any); !ok || params["quality"] != "hd" {
		t.Errorf("IMAGES_OPENAI_API_PARAMS: expected a JSON object, got %#v", body["IMAGES_OPENAI_API_PARAMS"])
	}
	nodes, ok := body["COMFYUI_WORKFLOW_NODES"].([]any)
	var first map[string]any
	if ok && len(nodes) == 1 {
		first, ok = nodes[0].(map[string]any)
	}
	if !ok || len(nodes) != 1 || first["key"] != "text" {
		t.Errorf("COMFYUI_WORKFLOW_NODES: expected a list of objects, got %#v", body["COMFYUI_WORKFLOW_NODES"])
	}
	if editNodes, ok := body["IMAGES_EDIT_COMFYUI_WORKFLOW_NODES"].([]any); !ok || len(editNodes) != 0 {
		t.Errorf("IMAGES_EDIT_COMFYUI_WORKFLOW_NODES: expected [], got %#v", body["IMAGES_EDIT_COMFYUI_WORKFLOW_NODES"])
	}
}

// TestSetImagesConfigNilStructuredFields: list[dict] fields are not nullable
// server-side, so unset raw node lists are sent as [] rather than null; unset
// dict|str|None params are sent as null.
func TestSetImagesConfigNilStructuredFields(t *testing.T) {
	var body map[string]any
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	})
	c, server := newConfigTestClient(handler)
	defer server.Close()

	if _, err := c.SetImagesConfig(context.Background(), ImagesConfigForm{}); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"COMFYUI_WORKFLOW_NODES", "IMAGES_EDIT_COMFYUI_WORKFLOW_NODES"} {
		if nodes, ok := body[k].([]any); !ok || len(nodes) != 0 {
			t.Errorf("%s: expected [], got %#v", k, body[k])
		}
	}
	for _, k := range []string{"IMAGES_OPENAI_API_PARAMS", "AUTOMATIC1111_PARAMS", "AUTOMATIC1111_API_AUTH", "IMAGE_SIZE", "IMAGE_STEPS"} {
		if v, ok := body[k]; !ok || v != nil {
			t.Errorf("%s: expected explicit null, got %#v (present=%v)", k, v, ok)
		}
	}
}

// TestImagesConfigDictAuthKeepsType: a dict-valued AUTOMATIC1111_API_AUTH read
// from the server is written back as the same JSON object while the text is
// unchanged, and as a plain string once the caller replaces it.
func TestImagesConfigDictAuthKeepsType(t *testing.T) {
	var posted []map[string]json.RawMessage
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var body map[string]json.RawMessage
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			posted = append(posted, body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"AUTOMATIC1111_API_AUTH": {"u": "p"}}`))
	})
	c, server := newConfigTestClient(handler)
	defer server.Close()

	current, err := c.GetImagesConfig(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if current.Automatic1111APIAuth == nil || *current.Automatic1111APIAuth != `{"u": "p"}` {
		t.Fatalf("expected the dict as its raw JSON text, got %v", current.Automatic1111APIAuth)
	}

	// Unchanged: sent back as the original object.
	if _, err := c.SetImagesConfig(context.Background(), *current); err != nil {
		t.Fatal(err)
	}
	// Changed: sent as a string.
	changed := *current
	auth := "user:pass"
	changed.Automatic1111APIAuth = &auth
	if _, err := c.SetImagesConfig(context.Background(), changed); err != nil {
		t.Fatal(err)
	}
	// A string that happens to look like JSON on a form that never held a
	// dict stays a string.
	plain := ImagesConfigForm{Automatic1111APIAuth: current.Automatic1111APIAuth}
	if _, err := c.SetImagesConfig(context.Background(), plain); err != nil {
		t.Fatal(err)
	}

	if len(posted) != 3 {
		t.Fatalf("expected 3 POSTs, got %d", len(posted))
	}
	if got := string(posted[0]["AUTOMATIC1111_API_AUTH"]); got != `{"u":"p"}` {
		t.Errorf("unchanged dict auth: expected the JSON object, got %s", got)
	}
	if got := string(posted[1]["AUTOMATIC1111_API_AUTH"]); got != `"user:pass"` {
		t.Errorf("changed auth: expected a JSON string, got %s", got)
	}
	if got := string(posted[2]["AUTOMATIC1111_API_AUTH"]); got != `"{\"u\": \"p\"}"` {
		t.Errorf("plain form: expected a JSON string, got %s", got)
	}
}
