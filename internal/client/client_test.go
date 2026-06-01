package client

import (
	"net/http"
	"testing"
)

func TestNewClient_InsecureFalse_DefaultTransport(t *testing.T) {
	c, err := NewClient("http://localhost:3000", "token", false)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if c.httpClient.Transport != nil {
		t.Errorf("expected nil transport for insecure=false, got %T", c.httpClient.Transport)
	}
}

func TestNewClient_InsecureTrue_SkipsVerify(t *testing.T) {
	c, err := NewClient("http://localhost:3000", "token", true)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	tr, ok := c.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected *http.Transport, got %T", c.httpClient.Transport)
	}
	if tr.TLSClientConfig == nil {
		t.Fatal("expected TLSClientConfig to be set")
	}
	if !tr.TLSClientConfig.InsecureSkipVerify {
		t.Error("expected InsecureSkipVerify=true")
	}
}
