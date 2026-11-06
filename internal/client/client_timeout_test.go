package client

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
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

// TestModelReloadRequestsUseLongTimeout: with the default timeout shortened
// to 1 s and the model-reload timeout to 10 s, an ordinary request slower than
// the default (1.5 s) times out, while the three model-reloading POSTs
// (embedding update, retrieval config update, web-only retrieval update) wait
// for the equally slow server. A second client whose model-reload timeout is
// 1 s times out on the slow POST, so that timeout is really applied. The GETs
// that precede the POSTs answer immediately, so the 1 s default leaves them a
// wide margin. The checks run concurrently (about 1.5 s in total).
func TestModelReloadRequestsUseLongTimeout(t *testing.T) {
	const delay = 1500 * time.Millisecond
	server := slowPostServer(t, delay, false)
	defer server.Close()
	newClient := func(reload time.Duration) *Client {
		c, err := NewClient(server.URL, "token", false)
		if err != nil {
			t.Fatal(err)
		}
		c.httpClient.Timeout = time.Second
		c.modelReloadHTTPClient.Timeout = reload
		return c
	}
	c, short := newClient(10*time.Second), newClient(time.Second)
	ctx := context.Background()
	size, topK, count := int64(42), int64(9), int64(9)

	checks := map[string]func() error{
		"ordinary request times out": func() error {
			if err := c.do(ctx, http.MethodGet, "slow", nil, nil, nil); !isTimeout(err) {
				return fmt.Errorf("must time out at the default timeout, got %v", err)
			}
			return nil
		},
		"embedding update": func() error {
			return c.UpdateRetrievalEmbeddingConfig(ctx, RetrievalEmbeddingConfigUpdate{RAGEmbeddingBatchSize: &size})
		},
		"retrieval config update": func() error {
			return c.UpdateRetrievalConfig(ctx, RetrievalConfigUpdateForm{TopK: &topK})
		},
		"web search update": func() error {
			_, err := c.UpdateWebSearchConfig(ctx, WebSearchConfigUpdate{WebSearchResultCount: &count})
			return err
		},
		"model-reload timeout applies": func() error {
			if err := short.UpdateRetrievalEmbeddingConfig(ctx, RetrievalEmbeddingConfigUpdate{RAGEmbeddingBatchSize: &size}); !isTimeout(err) {
				return fmt.Errorf("a POST slower than the model-reload timeout must time out, got %v", err)
			}
			return nil
		},
	}
	var wg sync.WaitGroup
	errs := make(chan string, len(checks))
	for name, check := range checks {
		wg.Add(1)
		go func(name string, check func() error) {
			defer wg.Done()
			if err := check(); err != nil {
				errs <- name + ": " + err.Error()
			}
		}(name, check)
	}
	wg.Wait()
	close(errs)
	for msg := range errs {
		t.Error(msg)
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
