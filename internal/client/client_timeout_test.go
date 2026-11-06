package client

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// slowPostServer answers GETs at once and POSTs after delay, like Open WebUI
// loading a model inside the update request.
func slowPostServer(t *testing.T, delay time.Duration, tlsServer bool) *httptest.Server {
	t.Helper()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/retrieval/embedding":
			_, _ = w.Write([]byte(stockEmbeddingConfig))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/retrieval/config":
			_, _ = w.Write([]byte(stockRetrievalWithWeb(stockWebSearchObject)))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/slow":
			time.Sleep(delay)
			_, _ = w.Write([]byte(`{}`))
		case r.Method == http.MethodPost:
			time.Sleep(delay)
			_, _ = w.Write([]byte(`{"status": true}`))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	})
	if tlsServer {
		return httptest.NewTLSServer(handler)
	}
	return httptest.NewServer(handler)
}

// isTimeout reports whether err is an http.Client timeout.
func isTimeout(err error) bool {
	var netErr interface{ Timeout() bool }
	return errors.As(err, &netErr) && netErr.Timeout()
}

func TestClientTimeoutDefaults(t *testing.T) {
	c, err := NewClient("https://openwebui.example", "token", false)
	if err != nil {
		t.Fatal(err)
	}
	if c.httpClient.Timeout != 30*time.Second || c.modelReloadHTTPClient.Timeout != 10*time.Minute {
		t.Errorf("timeouts: default %v, model reload %v", c.httpClient.Timeout, c.modelReloadHTTPClient.Timeout)
	}
	if c.modelReloadHTTPClient.Transport != c.httpClient.Transport {
		t.Errorf("the model-reload client must share the default client's transport")
	}
}

// TestModelReloadRequestsUseLongTimeout: with a short default timeout and a
// longer model-reload timeout (shortened for the test), an ordinary request
// slower than the default times out, while the three model-reloading POSTs
// (embedding update, retrieval config update, web-only retrieval update)
// wait for the slow server. A POST slower than the model-reload timeout still
// times out, so that timeout is really applied.
func TestModelReloadRequestsUseLongTimeout(t *testing.T) {
	server := slowPostServer(t, 150*time.Millisecond, false)
	defer server.Close()
	c, err := NewClient(server.URL, "token", false)
	if err != nil {
		t.Fatal(err)
	}
	c.httpClient.Timeout = 50 * time.Millisecond
	c.modelReloadHTTPClient.Timeout = 5 * time.Second
	ctx := context.Background()

	if err := c.do(ctx, http.MethodGet, "slow", nil, nil, nil); !isTimeout(err) {
		t.Errorf("an ordinary request must time out at the default timeout, got %v", err)
	}
	size := int64(42)
	if err := c.UpdateRetrievalEmbeddingConfig(ctx, RetrievalEmbeddingConfigUpdate{RAGEmbeddingBatchSize: &size}); err != nil {
		t.Errorf("embedding update: %v", err)
	}
	topK := int64(9)
	if err := c.UpdateRetrievalConfig(ctx, RetrievalConfigUpdateForm{TopK: &topK}); err != nil {
		t.Errorf("retrieval config update: %v", err)
	}
	count := int64(9)
	if _, err := c.UpdateWebSearchConfig(ctx, WebSearchConfigUpdate{WebSearchResultCount: &count}); err != nil {
		t.Errorf("web search update: %v", err)
	}

	c.modelReloadHTTPClient.Timeout = 50 * time.Millisecond
	if err := c.UpdateRetrievalEmbeddingConfig(ctx, RetrievalEmbeddingConfigUpdate{RAGEmbeddingBatchSize: &size}); !isTimeout(err) {
		t.Errorf("a POST slower than the model-reload timeout must time out, got %v", err)
	}
}

// TestModelReloadClientKeepsTLSSettings: with insecure_skip_verify both
// clients share one transport that skips verification, so a model-reloading
// POST to a self-signed server works like any other request.
func TestModelReloadClientKeepsTLSSettings(t *testing.T) {
	server := slowPostServer(t, 0, true)
	defer server.Close()
	c, err := NewClient(server.URL, "token", true)
	if err != nil {
		t.Fatal(err)
	}
	transport, ok := c.httpClient.Transport.(*http.Transport)
	if !ok || c.modelReloadHTTPClient.Transport != c.httpClient.Transport {
		t.Fatalf("the model-reload client must share the default client's transport")
	}
	if transport.TLSClientConfig == nil || !transport.TLSClientConfig.InsecureSkipVerify {
		t.Errorf("the shared transport must skip TLS verification")
	}
	size := int64(42)
	if err := c.UpdateRetrievalEmbeddingConfig(context.Background(), RetrievalEmbeddingConfigUpdate{RAGEmbeddingBatchSize: &size}); err != nil {
		t.Errorf("embedding update over self-signed TLS: %v", err)
	}

	// Without insecure_skip_verify the same update fails verification.
	strict, err := NewClient(server.URL, "token", false)
	if err != nil {
		t.Fatal(err)
	}
	err = strict.UpdateRetrievalEmbeddingConfig(context.Background(), RetrievalEmbeddingConfigUpdate{RAGEmbeddingBatchSize: &size})
	var tlsErr *tls.CertificateVerificationError
	if !errors.As(err, &tlsErr) {
		t.Errorf("expected a certificate error without insecure_skip_verify, got %v", err)
	}
}
