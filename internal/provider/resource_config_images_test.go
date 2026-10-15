package provider

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

func imagesHCL(attrs string) string {
	return `
resource "openwebui_images_config" "test" {
` + attrs + `
}
`
}

func stockImagesForm() client.ImagesConfigForm {
	s := func(v string) *string { return &v }
	steps := int64(50)
	return client.ImagesConfigForm{
		EnableImagePromptGeneration: true,
		ImageGenerationEngine:       "openai",
		ImageSize:                   s("512x512"),
		ImageSteps:                  &steps,
		ImagesOpenAIAPIBaseURL:      "https://api.openai.com/v1",
		ImagesOpenAIAPIKey:          "existing-key",
		ImagesOpenAIAPIParams:       json.RawMessage(`{}`),
		Automatic1111APIAuth:        s(""),
		Automatic1111Params:         json.RawMessage(`null`),
		ComfyUIWorkflow:             `{"3": {"inputs": {}}}`,
		ComfyUIWorkflowNodes:        json.RawMessage(`[]`),
		ImageEditEngine:             "openai",
		ImageEditSize:               nil,
		ImagesEditComfyUIWorkflowNodes: json.RawMessage(
			`[{"type": "image", "key": "image", "node_ids": ["1"]}]`),
	}
}

func unknownImagesModel() imagesConfigModel {
	m := imagesConfigModelFromForm(client.ImagesConfigForm{})
	m.EnableImageGeneration = types.BoolUnknown()
	m.EnableImagePromptGeneration = types.BoolUnknown()
	m.ImageGenerationEngine = types.StringUnknown()
	m.ImageGenerationModel = types.StringUnknown()
	m.ImageSize = types.StringUnknown()
	m.ImageSteps = types.Int64Unknown()
	m.ImagesOpenAIAPIBaseURL = types.StringUnknown()
	m.ImagesOpenAIAPIKey = types.StringUnknown()
	m.ImagesOpenAIAPIVersion = types.StringUnknown()
	m.ImagesOpenAIAPIParamsJSON = types.StringUnknown()
	m.Automatic1111BaseURL = types.StringUnknown()
	m.Automatic1111APIAuth = types.StringUnknown()
	m.Automatic1111ParamsJSON = types.StringUnknown()
	m.ComfyUIBaseURL = types.StringUnknown()
	m.ComfyUIAPIKey = types.StringUnknown()
	m.ComfyUIWorkflow = types.StringUnknown()
	m.ComfyUIWorkflowNodesJSON = types.StringUnknown()
	m.ImagesGeminiAPIBaseURL = types.StringUnknown()
	m.ImagesGeminiAPIKey = types.StringUnknown()
	m.ImagesGeminiEndpointMethod = types.StringUnknown()
	m.EnableImageEdit = types.BoolUnknown()
	m.ImageEditEngine = types.StringUnknown()
	m.ImageEditModel = types.StringUnknown()
	m.ImageEditSize = types.StringUnknown()
	m.ImagesEditOpenAIAPIBaseURL = types.StringUnknown()
	m.ImagesEditOpenAIAPIKey = types.StringUnknown()
	m.ImagesEditOpenAIAPIVersion = types.StringUnknown()
	m.ImagesEditGeminiAPIBaseURL = types.StringUnknown()
	m.ImagesEditGeminiAPIKey = types.StringUnknown()
	m.ImagesEditComfyUIBaseURL = types.StringUnknown()
	m.ImagesEditComfyUIAPIKey = types.StringUnknown()
	m.ImagesEditComfyUIWorkflow = types.StringUnknown()
	m.ImagesEditComfyUIWorkflowNodesJSON = types.StringUnknown()
	return m
}

// A first apply that sets nothing must send the server's current config back
// unchanged: the POST is a full-object write, so every unknown attribute is
// filled from the GET (including nulls and the raw JSON values).
func TestImagesOverlayAllUnknownKeepsCurrent(t *testing.T) {
	current := stockImagesForm()
	form, diags := overlayImagesConfigPlan(current, unknownImagesModel())
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if !reflect.DeepEqual(form, current) {
		t.Fatalf("expected the current config to be sent unchanged\nwant %+v\ngot  %+v", current, form)
	}
}

func TestImagesOverlayKnownValues(t *testing.T) {
	m := unknownImagesModel()
	m.EnableImagePromptGeneration = types.BoolValue(false)
	m.ImageSize = types.StringValue("")
	m.ImageSteps = types.Int64Value(0)
	m.ImagesOpenAIAPIKey = types.StringValue("")
	m.ImagesOpenAIAPIParamsJSON = types.StringValue(`{ "quality": "hd", "n": 1 }`)
	m.Automatic1111APIAuth = types.StringValue("user:pass")
	m.Automatic1111ParamsJSON = types.StringValue(`{}`)
	m.ComfyUIWorkflowNodesJSON = types.StringValue(`[{"type":"prompt","key":"text","node_ids":["6"]}]`)
	m.ImageEditSize = types.StringValue("1024x1024")

	form, diags := overlayImagesConfigPlan(stockImagesForm(), m)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if form.EnableImagePromptGeneration || form.ImageSize == nil || *form.ImageSize != "" ||
		form.ImageSteps == nil || *form.ImageSteps != 0 || form.ImagesOpenAIAPIKey != "" {
		t.Errorf("explicit false/\"\"/0 values must be sent: %+v", form)
	}
	if string(form.ImagesOpenAIAPIParams) != `{"n":1,"quality":"hd"}` {
		t.Errorf("expected canonical params, got %s", form.ImagesOpenAIAPIParams)
	}
	if string(form.Automatic1111Params) != `{}` || form.Automatic1111APIAuth == nil || *form.Automatic1111APIAuth != "user:pass" {
		t.Errorf("unexpected automatic1111 values: %s / %v", form.Automatic1111Params, form.Automatic1111APIAuth)
	}
	if string(form.ComfyUIWorkflowNodes) != `[{"key":"text","node_ids":["6"],"type":"prompt"}]` {
		t.Errorf("unexpected nodes %s", form.ComfyUIWorkflowNodes)
	}
	if form.ImageEditSize == nil || *form.ImageEditSize != "1024x1024" {
		t.Errorf("unexpected edit size %v", form.ImageEditSize)
	}
	// Unknown attributes still come from the server.
	if form.ImagesOpenAIAPIBaseURL != "https://api.openai.com/v1" || form.ComfyUIWorkflow != `{"3": {"inputs": {}}}` {
		t.Errorf("unknown attributes must keep server values: %+v", form)
	}
}

// A plan value that is semantically equal to what the server already stores
// keeps the server's raw value, so a legacy JSON-string params value is not
// silently converted into an object by an unrelated apply.
func TestImagesOverlayEquivalentJSONKeepsRaw(t *testing.T) {
	current := stockImagesForm()
	current.Automatic1111Params = json.RawMessage(`"{\"cfg_scale\": 7}"`)
	m := imagesConfigModelFromForm(current)
	form, diags := overlayImagesConfigPlan(current, m)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if string(form.Automatic1111Params) != `"{\"cfg_scale\": 7}"` {
		t.Errorf("expected the legacy raw string to be kept, got %s", form.Automatic1111Params)
	}
	if !reflect.DeepEqual(form, current) {
		t.Errorf("expected a state-derived plan to round-trip to the same form\nwant %+v\ngot  %+v", current, form)
	}
}

// Null, "{}"/"[]" and populated shapes for every JSON attribute, plus the
// nullable scalars.
func TestImagesModelFromFormJSONShapes(t *testing.T) {
	f := stockImagesForm()
	f.ImagesOpenAIAPIParams = json.RawMessage(`null`)
	f.Automatic1111Params = json.RawMessage(`{"b": 2, "a": 1}`)
	f.ComfyUIWorkflowNodes = json.RawMessage(`[]`)
	f.ImageSteps = nil
	m := imagesConfigModelFromForm(f)

	if !m.ImagesOpenAIAPIParamsJSON.IsNull() {
		t.Errorf("null params must be null in state, got %v", m.ImagesOpenAIAPIParamsJSON)
	}
	if m.Automatic1111ParamsJSON.ValueString() != `{"a":1,"b":2}` {
		t.Errorf("expected canonical params, got %v", m.Automatic1111ParamsJSON)
	}
	if m.ComfyUIWorkflowNodesJSON.ValueString() != "[]" {
		t.Errorf("expected [] nodes, got %v", m.ComfyUIWorkflowNodesJSON)
	}
	if m.ImagesEditComfyUIWorkflowNodesJSON.ValueString() != `[{"key":"image","node_ids":["1"],"type":"image"}]` {
		t.Errorf("unexpected edit nodes %v", m.ImagesEditComfyUIWorkflowNodesJSON)
	}
	if !m.ImageSteps.IsNull() || !m.ImageEditSize.IsNull() {
		t.Errorf("nil steps / edit size must be null")
	}
	if m.ImageSize.ValueString() != "512x512" || m.Automatic1111APIAuth.ValueString() != "" || m.ImagesOpenAIAPIKey.ValueString() != "existing-key" {
		t.Errorf("unexpected scalars %+v", m)
	}
	if m.ComfyUIWorkflow.ValueString() != `{"3": {"inputs": {}}}` {
		t.Errorf("workflow string must be kept verbatim, got %q", m.ComfyUIWorkflow.ValueString())
	}
	if m.ID.ValueString() != "images" {
		t.Errorf("unexpected id %q", m.ID.ValueString())
	}

	// {} is distinct from null.
	f.ImagesOpenAIAPIParams = json.RawMessage(`{}`)
	if got := imagesConfigModelFromForm(f).ImagesOpenAIAPIParamsJSON; got.ValueString() != "{}" {
		t.Errorf("expected {} params, got %v", got)
	}
	// A legacy JSON-string value is shown as its (canonical) JSON content.
	f.ImagesOpenAIAPIParams = json.RawMessage(`"{\"z\": 1, \"a\": 2}"`)
	if got := imagesConfigModelFromForm(f).ImagesOpenAIAPIParamsJSON; got.ValueString() != `{"a":2,"z":1}` {
		t.Errorf("expected canonical legacy-string params, got %v", got)
	}
}

func TestImagesPreserveUserJSONFormatting(t *testing.T) {
	fresh := imagesConfigModelFromForm(stockImagesForm())
	prior := fresh
	prior.ImagesEditComfyUIWorkflowNodesJSON = types.StringValue(`[ { "type": "image", "node_ids": ["1"], "key": "image" } ]`)
	prior.ImagesOpenAIAPIParamsJSON = types.StringValue(`{ "different": true }`)

	got := preserveImagesConfigJSON(prior, fresh)
	if got.ImagesEditComfyUIWorkflowNodesJSON != prior.ImagesEditComfyUIWorkflowNodesJSON {
		t.Errorf("equivalent JSON must keep the user's text, got %v", got.ImagesEditComfyUIWorkflowNodesJSON)
	}
	if got.ImagesOpenAIAPIParamsJSON != fresh.ImagesOpenAIAPIParamsJSON {
		t.Errorf("different JSON must take the server value, got %v", got.ImagesOpenAIAPIParamsJSON)
	}
}

func TestImagesConfigValidators(t *testing.T) {
	for _, ok := range []string{"", "auto", "512x512", "1792x1024"} {
		planOnlyValid(t, imagesHCL(`  image_size = "`+ok+`"`))
	}
	for _, bad := range []string{"512", "512 x 512", "large", "x512"} {
		planOnlyRejected(t, imagesHCL(`  image_size = "`+bad+`"`), `(?s)image_size.*WIDTHxHEIGHT`)
	}

	planOnlyValid(t, imagesHCL(`  image_steps = 0`))
	planOnlyRejected(t, imagesHCL(`  image_steps = -1`), `(?s)image_steps.*at\s+least\s+0`)

	for _, attr := range []string{"comfyui_base_url", "images_edit_comfyui_base_url"} {
		for _, ok := range []string{"", "http://comfy:8188", "http://comfy/sub/path"} {
			planOnlyValid(t, imagesHCL(`  `+attr+` = "`+ok+`"`))
		}
		for _, bad := range []string{"http://comfy:8188/", "/relative"} {
			planOnlyRejected(t, imagesHCL(`  `+attr+` = "`+bad+`"`), `(?s)`+attr+`.*slash`)
		}
	}

	for _, attr := range []string{"images_openai_api_params_json", "automatic1111_params_json"} {
		for _, ok := range []string{`{}`, `{\"quality\": \"hd\"}`} {
			planOnlyValid(t, imagesHCL(`  `+attr+` = "`+ok+`"`))
		}
		for _, bad := range []string{``, `null`, `[]`, `{`, `\"text\"`} {
			planOnlyRejected(t, imagesHCL(`  `+attr+` = "`+bad+`"`), `(?s)`+attr+`.*JSON\s+object`)
		}
	}
	for _, attr := range []string{"comfyui_workflow_nodes_json", "images_edit_comfyui_workflow_nodes_json"} {
		for _, ok := range []string{`[]`, `[{\"type\": \"prompt\", \"key\": \"text\", \"node_ids\": []}]`} {
			planOnlyValid(t, imagesHCL(`  `+attr+` = "`+ok+`"`))
		}
		for _, bad := range []string{``, `null`, `{}`, `[1]`, `[null]`} {
			planOnlyRejected(t, imagesHCL(`  `+attr+` = "`+bad+`"`), `(?s)`+attr+`.*array\s+of\s+objects`)
		}
	}
}

func TestImagesConfigSchemaShape(t *testing.T) {
	var resp resource.SchemaResponse
	NewImagesConfigResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if len(resp.Schema.Attributes) != 34 {
		t.Fatalf("expected id plus 33 attributes, got %d", len(resp.Schema.Attributes))
	}
	sensitive := map[string]bool{
		"images_openai_api_key": true, "automatic1111_api_auth": true, "comfyui_api_key": true,
		"images_gemini_api_key": true, "images_edit_openai_api_key": true, "images_edit_gemini_api_key": true,
		"images_edit_comfyui_api_key": true,
	}
	for name, attr := range resp.Schema.Attributes {
		if attr.GetDescription() == "" || attr.GetMarkdownDescription() == "" {
			t.Errorf("attribute %q needs both Description and MarkdownDescription", name)
		}
		if attr.IsSensitive() != sensitive[name] {
			t.Errorf("attribute %q: unexpected Sensitive=%v", name, attr.IsSensitive())
		}
		if name == "id" {
			continue
		}
		if !attr.IsOptional() || !attr.IsComputed() {
			t.Errorf("attribute %q must be Optional+Computed", name)
		}
	}
}
