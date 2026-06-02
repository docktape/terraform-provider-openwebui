package client

import (
	"net/http"
	"strings"
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
		endpoint    string
		wantRootURL string
		wantBaseURL string
	}{
		{"http://localhost:8080", "http://localhost:8080", "http://localhost:8080/api/v1"},
		{"https://openwebui.example.com", "https://openwebui.example.com", "https://openwebui.example.com/api/v1"},
		{"http://localhost:3000", "http://localhost:3000", "http://localhost:3000/api/v1"},
		{"http://10.0.0.1:9000", "http://10.0.0.1:9000", "http://10.0.0.1:9000/api/v1"},
		{"http://localhost:8080/", "http://localhost:8080", "http://localhost:8080/api/v1"}, // trailing slash
	}
	for _, tc := range tests {
		c, err := NewClient(tc.endpoint, "tok", false)
		if err != nil {
			t.Fatalf("NewClient(%q): %v", tc.endpoint, err)
		}
		if c.rootURL != tc.wantRootURL {
			t.Errorf("endpoint=%q: rootURL=%q, want %q", tc.endpoint, c.rootURL, tc.wantRootURL)
		}
		if c.baseURL != tc.wantBaseURL {
			t.Errorf("endpoint=%q: baseURL=%q, want %q", tc.endpoint, c.baseURL, tc.wantBaseURL)
		}
	}
}

func TestNewClient_RejectsApiV1Suffix(t *testing.T) {
	bad := []string{
		"http://localhost:8080/api/v1",
		"https://openwebui.example.com/api/v1",
		"http://localhost:8080/api/v1/",
	}
	for _, endpoint := range bad {
		_, err := NewClient(endpoint, "tok", false)
		if err == nil {
			t.Errorf("NewClient(%q): expected error for /api/v1 suffix, got nil", endpoint)
			continue
		}
		if !strings.Contains(err.Error(), "bare base URL") {
			t.Errorf("NewClient(%q): error %q does not mention 'bare base URL'", endpoint, err.Error())
		}
	}
}
