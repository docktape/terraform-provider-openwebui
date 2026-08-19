package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newAdminConfigTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	c, err := NewClient(server.URL, "test-token", false)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

// TestGetSetAdminConfigRoundTrip tests round-tripping a representative subset of admin config fields
func TestGetSetAdminConfigRoundTrip(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Representative subset: a bool, strings, optional strings, optional int64s, and ENABLE_MEMORY_SYSTEM_CONTEXT
		_, _ = w.Write([]byte(`{
			"SHOW_ADMIN_DETAILS": true,
			"ADMIN_EMAIL": "admin@example.com",
			"WEBUI_URL": "https://webui.example.com",
			"ENABLE_SIGNUP": false,
			"ENABLE_API_KEYS": true,
			"ENABLE_API_KEYS_ENDPOINT_RESTRICTIONS": false,
			"API_KEYS_ALLOWED_ENDPOINTS": "/api/v1/chat/completions,/api/v1/models",
			"DEFAULT_USER_ROLE": "user",
			"DEFAULT_GROUP_ID": "default",
			"JWT_EXPIRES_IN": "30d",
			"ENABLE_COMMUNITY_SHARING": true,
			"ENABLE_MESSAGE_RATING": false,
			"ENABLE_FOLDERS": true,
			"FOLDER_MAX_FILE_COUNT": 100,
			"AUTOMATION_MAX_COUNT": 50,
			"AUTOMATION_MIN_INTERVAL": 1000,
			"ENABLE_AUTOMATIONS": true,
			"ENABLE_CHANNELS": false,
			"ENABLE_CALENDAR": true,
			"ENABLE_MEMORIES": true,
			"ENABLE_MEMORY_SYSTEM_CONTEXT": true,
			"ENABLE_NOTES": false,
			"ENABLE_USER_WEBHOOKS": true,
			"ENABLE_USER_STATUS": false,
			"PENDING_USER_OVERLAY_TITLE": "Welcome",
			"PENDING_USER_OVERLAY_CONTENT": "Please wait for approval",
			"RESPONSE_WATERMARK": "Powered by Open WebUI"
		}`))
	}))
	t.Cleanup(server.Close)

	c, err := NewClient(server.URL, "test-token", false)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	// Get the config
	got, err := c.GetAdminConfig(context.Background())
	if err != nil {
		t.Fatalf("GetAdminConfig: %v", err)
	}

	// Verify representative fields
	if !got.ShowAdminDetails {
		t.Fatalf("expected ShowAdminDetails=true, got false")
	}
	if got.AdminEmail == nil || *got.AdminEmail != "admin@example.com" {
		t.Fatalf("expected AdminEmail=admin@example.com, got %v", got.AdminEmail)
	}
	if got.WebUIURL != "https://webui.example.com" {
		t.Fatalf("expected WebUIURL=https://webui.example.com, got %q", got.WebUIURL)
	}
	if got.DefaultUserRole != "user" {
		t.Fatalf("expected DefaultUserRole=user, got %q", got.DefaultUserRole)
	}
	if got.FolderMaxFileCount == nil || *got.FolderMaxFileCount != 100 {
		t.Fatalf("expected FolderMaxFileCount=100, got %v", got.FolderMaxFileCount)
	}
	if got.EnableMemorySystemContext == nil || !*got.EnableMemorySystemContext {
		t.Fatalf("expected EnableMemorySystemContext=true, got %v", got.EnableMemorySystemContext)
	}
	if got.PendingUserOverlayTitle == nil || *got.PendingUserOverlayTitle != "Welcome" {
		t.Fatalf("expected PendingUserOverlayTitle=Welcome, got %v", got.PendingUserOverlayTitle)
	}

	// Set the config (round-trip)
	updated, err := c.SetAdminConfig(context.Background(), *got)
	if err != nil {
		t.Fatalf("SetAdminConfig: %v", err)
	}

	// Verify round-tripped fields
	if !updated.ShowAdminDetails {
		t.Fatalf("expected round-tripped ShowAdminDetails=true, got false")
	}
	if updated.AdminEmail == nil || *updated.AdminEmail != "admin@example.com" {
		t.Fatalf("expected round-tripped AdminEmail=admin@example.com, got %v", updated.AdminEmail)
	}
	if updated.DefaultUserRole != "user" {
		t.Fatalf("expected round-tripped DefaultUserRole=user, got %q", updated.DefaultUserRole)
	}
	if updated.FolderMaxFileCount == nil || *updated.FolderMaxFileCount != 100 {
		t.Fatalf("expected round-tripped FolderMaxFileCount=100, got %v", updated.FolderMaxFileCount)
	}
	if updated.EnableMemorySystemContext == nil || !*updated.EnableMemorySystemContext {
		t.Fatalf("expected round-tripped EnableMemorySystemContext=true, got %v", updated.EnableMemorySystemContext)
	}
}
