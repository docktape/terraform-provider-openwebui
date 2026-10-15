package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

const adminConfigTestJSON = `{
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
	"CHANNEL_MODEL_RESPONSE_MODE": "thread",
	"ENABLE_CALENDAR": true,
	"ENABLE_MEMORIES": true,
	"ENABLE_MEMORY_SYSTEM_CONTEXT": true,
	"ENABLE_NOTES": false,
	"ENABLE_USER_WEBHOOKS": true,
	"ENABLE_USER_STATUS": false,
	"PENDING_USER_OVERLAY_TITLE": "Welcome",
	"PENDING_USER_OVERLAY_CONTENT": "Please wait for approval",
	"RESPONSE_WATERMARK": "Powered by Open WebUI"
}`

// TestGetSetAdminConfigRoundTrip tests round-tripping a representative subset of admin config fields.
func TestGetSetAdminConfigRoundTrip(t *testing.T) {
	c := newAdminConfigTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auths/admin/config" {
			t.Errorf("expected path /api/v1/auths/admin/config, got %s", r.URL.Path)
		}

		switch r.Method {
		case http.MethodGet:
			// no request body to inspect.
		case http.MethodPost:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("failed to decode request body: %v", err)
			}
			// ENABLE_MEMORY_SYSTEM_CONTEXT and CHANNEL_MODEL_RESPONSE_MODE have no
			// Python default at v0.11.0 (they're required fields on the backend's
			// Pydantic model), so the wire payload must always include the key -
			// omitting it (e.g. via a JSON `omitempty` tag) causes a 422 on a real
			// server.
			for _, key := range []string{"ENABLE_MEMORY_SYSTEM_CONTEXT", "CHANNEL_MODEL_RESPONSE_MODE", "AUTOMATION_MAX_COUNT"} {
				if _, ok := body[key]; !ok {
					t.Errorf("expected request body to contain key %q, got %v", key, body)
				}
			}
		default:
			t.Errorf("expected GET or POST, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(adminConfigTestJSON)); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	})

	// Get the config.
	got, err := c.GetAdminConfig(context.Background())
	if err != nil {
		t.Fatalf("GetAdminConfig: %v", err)
	}

	// Verify representative fields.
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
	if got.ChannelModelResponseMode != "thread" {
		t.Fatalf("expected ChannelModelResponseMode=thread, got %q", got.ChannelModelResponseMode)
	}
	if !got.EnableMemorySystemContext {
		t.Fatal("expected EnableMemorySystemContext=true")
	}
	if got.PendingUserOverlayTitle == nil || *got.PendingUserOverlayTitle != "Welcome" {
		t.Fatalf("expected PendingUserOverlayTitle=Welcome, got %v", got.PendingUserOverlayTitle)
	}

	// Set the config (round-trip).
	updated, err := c.SetAdminConfig(context.Background(), *got)
	if err != nil {
		t.Fatalf("SetAdminConfig: %v", err)
	}

	// Verify round-tripped fields.
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
	if updated.ChannelModelResponseMode != "thread" {
		t.Fatalf("expected round-tripped ChannelModelResponseMode=thread, got %q", updated.ChannelModelResponseMode)
	}
	if !updated.EnableMemorySystemContext {
		t.Fatal("expected round-tripped EnableMemorySystemContext=true")
	}
}

// TestAdminConfigFormEnableMemorySystemContextZeroValueSentAsFalse guards the
// exact bug this fix addresses: ENABLE_MEMORY_SYSTEM_CONTEXT has no Python
// default at v0.11.0 and is declared as a plain, non-Optional `bool` on the
// backend's AdminConfig (backend/open_webui/routers/auths.py), so it rejects
// both a missing key and an explicit `null` with a 422. A Terraform user who
// never sets enable_memory_system_context must still produce a request that
// sends the key as a genuine, valid boolean - never omitted and never null.
// This exercises the "field genuinely unset by the user" path directly via
// json.Marshal on a zero-value form, rather than round-tripping a value that
// was already decoded from a populated response (which would not catch a
// regression back to a *bool/omitempty representation).
func TestAdminConfigFormEnableMemorySystemContextZeroValueSentAsFalse(t *testing.T) {
	form := AdminConfigForm{}

	encoded, err := json.Marshal(form)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatalf("json.Unmarshal of encoded form: %v", err)
	}
	raw, ok := fields["ENABLE_MEMORY_SYSTEM_CONTEXT"]
	if !ok {
		t.Fatalf("expected key ENABLE_MEMORY_SYSTEM_CONTEXT to be present, got %s", encoded)
	}
	if string(raw) != "false" {
		t.Fatalf("expected ENABLE_MEMORY_SYSTEM_CONTEXT to be false (not null), got %s", raw)
	}
}

// TestAdminConfigFormIntOrEmptyStringFields guards the exact bug this fix
// addresses: FOLDER_MAX_FILE_COUNT, AUTOMATION_MAX_COUNT, and
// AUTOMATION_MIN_INTERVAL each default via Python's os.getenv(key, "")
// (backend/open_webui/config.py) and are stored as int(v) if v else "" by
// the update handler (backend/open_webui/routers/auths.py's
// update_admin_config) - so a stock, never-configured server serves the
// JSON string "" for these keys rather than a number, null, or an omitted
// key. Unmarshaling "" into a plain *int64 field crashes with a type error
// before any write ever happens.
func TestAdminConfigFormIntOrEmptyStringFields(t *testing.T) {
	t.Run("empty string decodes to nil", func(t *testing.T) {
		var form AdminConfigForm
		if err := json.Unmarshal([]byte(`{"FOLDER_MAX_FILE_COUNT":""}`), &form); err != nil {
			t.Fatalf("json.Unmarshal: %v", err)
		}
		if form.FolderMaxFileCount != nil {
			t.Fatalf("expected FolderMaxFileCount=nil, got %v", *form.FolderMaxFileCount)
		}
	})

	t.Run("real integer decodes to that value", func(t *testing.T) {
		var form AdminConfigForm
		if err := json.Unmarshal([]byte(`{"AUTOMATION_MAX_COUNT":50}`), &form); err != nil {
			t.Fatalf("json.Unmarshal: %v", err)
		}
		if form.AutomationMaxCount == nil || *form.AutomationMaxCount != 50 {
			t.Fatalf("expected AutomationMaxCount=50, got %v", form.AutomationMaxCount)
		}
	})

	t.Run("null decodes to nil", func(t *testing.T) {
		var form AdminConfigForm
		if err := json.Unmarshal([]byte(`{"AUTOMATION_MIN_INTERVAL":null}`), &form); err != nil {
			t.Fatalf("json.Unmarshal: %v", err)
		}
		if form.AutomationMinInterval != nil {
			t.Fatalf("expected AutomationMinInterval=nil, got %v", *form.AutomationMinInterval)
		}
	})

	t.Run("Set round-trips with the field left unset", func(t *testing.T) {
		c := newAdminConfigTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.Method {
			case http.MethodGet:
				// A stock, never-configured server: all three fields are "".
				_, _ = w.Write([]byte(`{"FOLDER_MAX_FILE_COUNT":"","AUTOMATION_MAX_COUNT":"","AUTOMATION_MIN_INTERVAL":""}`))
			case http.MethodPost:
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatalf("failed to decode request body: %v", err)
				}
				// A nil *int64 must marshal to a bare `null`, which the real
				// backend's `int | str | None = None` request-body field accepts.
				for _, key := range []string{"FOLDER_MAX_FILE_COUNT", "AUTOMATION_MAX_COUNT", "AUTOMATION_MIN_INTERVAL"} {
					v, ok := body[key]
					if !ok {
						t.Errorf("expected request body to contain key %q, got %v", key, body)
					}
					if v != nil {
						t.Errorf("expected request body key %q to be null, got %v", key, v)
					}
				}
				_, _ = w.Write([]byte(`{"FOLDER_MAX_FILE_COUNT":"","AUTOMATION_MAX_COUNT":"","AUTOMATION_MIN_INTERVAL":""}`))
			default:
				t.Errorf("expected GET or POST, got %s", r.Method)
			}
		})

		got, err := c.GetAdminConfig(context.Background())
		if err != nil {
			t.Fatalf("GetAdminConfig: %v", err)
		}
		if got.FolderMaxFileCount != nil || got.AutomationMaxCount != nil || got.AutomationMinInterval != nil {
			t.Fatalf("expected all three fields nil on a stock server, got %v %v %v", got.FolderMaxFileCount, got.AutomationMaxCount, got.AutomationMinInterval)
		}

		updated, err := c.SetAdminConfig(context.Background(), *got)
		if err != nil {
			t.Fatalf("SetAdminConfig: %v", err)
		}
		if updated.FolderMaxFileCount != nil || updated.AutomationMaxCount != nil || updated.AutomationMinInterval != nil {
			t.Fatalf("expected all three round-tripped fields nil, got %v %v %v", updated.FolderMaxFileCount, updated.AutomationMaxCount, updated.AutomationMinInterval)
		}
	})
}

// TestDecodeIntOrEmptyString covers every branch of the helper that decodes
// the backend's `int | ""` union, including the numeric-string branch and the
// garbage-input error path.
func TestDecodeIntOrEmptyString(t *testing.T) {
	i64 := func(v int64) *int64 { return &v }

	tests := []struct {
		name    string
		raw     json.RawMessage
		want    *int64
		wantErr string
	}{
		{name: "absent", raw: nil, want: nil},
		{name: "null", raw: json.RawMessage(`null`), want: nil},
		{name: "empty string", raw: json.RawMessage(`""`), want: nil},
		{name: "number", raw: json.RawMessage(`42`), want: i64(42)},
		{name: "zero", raw: json.RawMessage(`0`), want: i64(0)},
		{name: "negative number", raw: json.RawMessage(`-5`), want: i64(-5)},
		{name: "numeric string", raw: json.RawMessage(`"42"`), want: i64(42)},
		{name: "negative numeric string", raw: json.RawMessage(`"-7"`), want: i64(-7)},
		{name: "non-numeric string is an error", raw: json.RawMessage(`"abc"`), wantErr: `non-numeric string "abc"`},
		{name: "fractional number is an error", raw: json.RawMessage(`4.5`), wantErr: "expected a JSON number or string"},
		{name: "boolean is an error", raw: json.RawMessage(`true`), wantErr: "expected a JSON number or string"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := decodeIntOrEmptyString(tc.raw)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got value %v", tc.wantErr, got)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			switch {
			case tc.want == nil && got != nil:
				t.Fatalf("expected nil, got %d", *got)
			case tc.want != nil && got == nil:
				t.Fatalf("expected %d, got nil", *tc.want)
			case tc.want != nil && *got != *tc.want:
				t.Fatalf("expected %d, got %d", *tc.want, *got)
			}
		})
	}
}

// TestAdminConfigFormGarbageIntFieldIsNotSwallowed proves a non-numeric string
// in one of the int-or-empty fields surfaces as an error naming the key,
// rather than being silently read as "unset".
func TestAdminConfigFormGarbageIntFieldIsNotSwallowed(t *testing.T) {
	var form AdminConfigForm
	err := json.Unmarshal([]byte(`{"AUTOMATION_MAX_COUNT":"abc"}`), &form)
	if err == nil {
		t.Fatal("expected an error for a non-numeric AUTOMATION_MAX_COUNT string, got nil")
	}
	if !strings.Contains(err.Error(), "AUTOMATION_MAX_COUNT") {
		t.Fatalf("expected error to name AUTOMATION_MAX_COUNT, got %v", err)
	}
}
