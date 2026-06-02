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

func TestNewClientRootURL(t *testing.T) {
	tests := []struct {
		endpoint string
		want     string
	}{
		{"http://localhost:8080/api/v1", "http://localhost:8080"},
		{"https://openwebui.example.com/api/v1", "https://openwebui.example.com"},
		{"http://localhost:3000", "http://localhost:3000"},
		{"http://10.0.0.1:9000/api/v1", "http://10.0.0.1:9000"},
	}
	for _, tc := range tests {
		c, err := NewClient(tc.endpoint, "tok", false)
		if err != nil {
			t.Fatalf("NewClient(%q): %v", tc.endpoint, err)
		}
		if c.rootURL != tc.want {
			t.Errorf("endpoint=%q: rootURL=%q, want %q", tc.endpoint, c.rootURL, tc.want)
		}
	}
}
