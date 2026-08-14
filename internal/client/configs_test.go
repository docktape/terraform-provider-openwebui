package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSetModelsConfigSendsDefaultMetadataAndParams(t *testing.T) {
	var gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		gotBody = string(buf)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"DEFAULT_MODELS":null,"DEFAULT_PINNED_MODELS":null,"MODEL_ORDER_LIST":[],"DEFAULT_MODEL_METADATA":{"source":"terraform"},"DEFAULT_MODEL_PARAMS":{"temperature":0.5}}`))
	}))
	t.Cleanup(server.Close)

	c, err := NewClient(server.URL, "test-token", false)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	form := ModelsConfigForm{
		DefaultModelMetadata: map[string]any{"source": "terraform"},
		DefaultModelParams:   map[string]any{"temperature": 0.5},
	}
	resp, err := c.SetModelsConfig(context.Background(), form)
	if err != nil {
		t.Fatalf("SetModelsConfig: %v", err)
	}

	if !contains(gotBody, `"DEFAULT_MODEL_METADATA":{"source":"terraform"}`) {
		t.Fatalf("expected DEFAULT_MODEL_METADATA in request body, got %s", gotBody)
	}
	if !contains(gotBody, `"DEFAULT_MODEL_PARAMS":{"temperature":0.5}`) {
		t.Fatalf("expected DEFAULT_MODEL_PARAMS in request body, got %s", gotBody)
	}
	if resp.DefaultModelMetadata["source"] != "terraform" {
		t.Fatalf("expected response DefaultModelMetadata[source]=terraform, got %v", resp.DefaultModelMetadata)
	}
	if resp.DefaultModelParams["temperature"] != 0.5 {
		t.Fatalf("expected response DefaultModelParams[temperature]=0.5, got %v", resp.DefaultModelParams)
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && indexOf(haystack, needle) >= 0
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
