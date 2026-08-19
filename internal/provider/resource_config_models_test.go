package provider

import (
	"context"
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
