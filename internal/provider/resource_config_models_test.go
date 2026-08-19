package provider

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

func TestApplyModelsConfig_DefaultMetadataAndParamsRoundTrip(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"DEFAULT_MODELS":null,"DEFAULT_PINNED_MODELS":null,"MODEL_ORDER_LIST":[],"DEFAULT_MODEL_METADATA":{"source":"terraform"},"DEFAULT_MODEL_PARAMS":{"temperature":0.5}}`))
	}))
	t.Cleanup(server.Close)

	apiClient, err := client.NewClient(server.URL, "test-token", false)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	ctx := context.Background()
	plan := modelsConfigModel{
		DefaultModelMetadataJSON: types.StringValue(`{"source":"terraform"}`),
		DefaultModelParamsJSON:   types.StringValue(`{"temperature":0.5}`),
		ModelOrderList:           types.ListNull(types.StringType),
		DefaultModels:            types.StringNull(),
		DefaultPinnedModels:      types.StringNull(),
	}

	state, diags := applyModelsConfig(ctx, apiClient, plan)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}

	if state.DefaultModelMetadataJSON.IsNull() {
		t.Fatal("expected non-null default_model_metadata_json")
	}
	if state.DefaultModelMetadataJSON.ValueString() != `{"source":"terraform"}` {
		t.Fatalf("expected metadata round-trip, got %s", state.DefaultModelMetadataJSON.ValueString())
	}
	if state.DefaultModelParamsJSON.IsNull() {
		t.Fatal("expected non-null default_model_params_json")
	}
	if state.DefaultModelParamsJSON.ValueString() != `{"temperature":0.5}` {
		t.Fatalf("expected params round-trip, got %s", state.DefaultModelParamsJSON.ValueString())
	}
}

func TestApplyModelsConfig_UnsetFieldsPreserveServerValue(t *testing.T) {
	getCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			getCalls++
			_, _ = w.Write([]byte(`{"DEFAULT_MODELS":null,"DEFAULT_PINNED_MODELS":null,"MODEL_ORDER_LIST":[],"DEFAULT_MODEL_METADATA":{"source":"admin-ui"},"DEFAULT_MODEL_PARAMS":{"temperature":0.9}}`))
			return
		}
		body, _ := io.ReadAll(r.Body)
		if !bytes.Contains(body, []byte(`"DEFAULT_MODEL_METADATA":{"source":"admin-ui"}`)) {
			t.Errorf("expected request to echo back the server's existing DEFAULT_MODEL_METADATA, got body: %s", body)
		}
		if !bytes.Contains(body, []byte(`"DEFAULT_MODEL_PARAMS":{"temperature":0.9}`)) {
			t.Errorf("expected request to echo back the server's existing DEFAULT_MODEL_PARAMS, got body: %s", body)
		}
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)

	apiClient, err := client.NewClient(server.URL, "test-token", false)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	ctx := context.Background()
	// Plan does NOT set default_model_metadata_json/default_model_params_json —
	// this is the common case (e.g. the first apply against an
	// already-configured instance) that must not wipe existing server values.
	plan := modelsConfigModel{
		DefaultModelMetadataJSON: types.StringNull(),
		DefaultModelParamsJSON:   types.StringNull(),
		ModelOrderList:           types.ListNull(types.StringType),
		DefaultModels:            types.StringNull(),
		DefaultPinnedModels:      types.StringNull(),
	}

	_, diags := applyModelsConfig(ctx, apiClient, plan)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if getCalls != 1 {
		t.Fatalf("expected exactly 1 GET call to fetch current config, got %d", getCalls)
	}
}
