package client

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

// fullOAuthResponse mirrors a stock Open WebUI v0.11.0 GET response: nulls for
// OAUTH_CODE_CHALLENGE_METHOD / OAUTH_TOKEN_ENDPOINT_AUTH_METHOD /
// OAUTH_SUB_CLAIM, "" for both timeouts, a JSON bool for
// OAUTH_GROUP_DEFAULT_SHARE.
const fullOAuthResponse = `{
	"ENABLE_OAUTH": true, "ENABLE_OAUTH_SIGNUP": false, "OAUTH_MERGE_ACCOUNTS_BY_EMAIL": false,
	"OAUTH_AUTO_REDIRECT": false, "OAUTH_ALLOWED_DOMAINS": "*", "OAUTH_BLOCKED_GROUPS": "[]",
	"ENABLE_OAUTH_ROLE_MANAGEMENT": false, "OAUTH_ROLES_CLAIM": "roles", "OAUTH_ADMIN_ROLES": "admin",
	"OAUTH_ALLOWED_ROLES": "user,admin", "ENABLE_OAUTH_GROUP_MANAGEMENT": false,
	"ENABLE_OAUTH_GROUP_CREATION": false, "OAUTH_GROUP_CLAIM": "groups", "OAUTH_GROUP_DEFAULT_SHARE": true,
	"OAUTH_PROVIDER_NAME": "SSO", "OPENID_PROVIDER_URL": "", "OAUTH_CLIENT_ID": "", "OAUTH_CLIENT_SECRET": "plain-secret",
	"OPENID_REDIRECT_URI": "", "OAUTH_SCOPES": "openid email profile", "OAUTH_CODE_CHALLENGE_METHOD": null,
	"OAUTH_TOKEN_ENDPOINT_AUTH_METHOD": null, "OPENID_END_SESSION_ENDPOINT": "", "OAUTH_TIMEOUT": "",
	"OAUTH_CLIENT_TIMEOUT": "", "OAUTH_EMAIL_CLAIM": "email", "OAUTH_USERNAME_CLAIM": "name",
	"OAUTH_PICTURE_CLAIM": "picture", "OAUTH_SUB_CLAIM": null, "OAUTH_AUDIENCE": "",
	"OAUTH_UPDATE_EMAIL_ON_LOGIN": false, "OAUTH_UPDATE_NAME_ON_LOGIN": false,
	"OAUTH_UPDATE_PICTURE_ON_LOGIN": false, "OAUTH_REFRESH_TOKEN_INCLUDE_SCOPE": false
}`

func TestGetOAuthConfig(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/auths/admin/config/oauth" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fullOAuthResponse))
	})
	c, server := newLDAPConfigTestClient(handler)
	defer server.Close()

	got, err := c.GetOAuthConfig(context.Background())
	if err != nil {
		t.Fatalf("GetOAuthConfig failed: %v", err)
	}
	if got.EnableOAuth == nil || !*got.EnableOAuth || got.EnableOAuthSignup == nil || *got.EnableOAuthSignup {
		t.Errorf("unexpected bools: %+v", got)
	}
	if got.OAuthAllowedDomains == nil || *got.OAuthAllowedDomains != "*" || got.OAuthClientSecret == nil || *got.OAuthClientSecret != "plain-secret" {
		t.Errorf("unexpected strings: %+v", got)
	}
	if got.OAuthCodeChallengeMethod != nil || got.OAuthTokenEndpointAuthMethod != nil || got.OAuthSubClaim != nil {
		t.Errorf("expected null fields to be nil: %+v", got)
	}
	if got.OAuthTimeout == nil || *got.OAuthTimeout != "" || got.OAuthClientTimeout == nil || *got.OAuthClientTimeout != "" {
		t.Errorf("expected empty-string timeouts to decode as \"\", got %v / %v", got.OAuthTimeout, got.OAuthClientTimeout)
	}
	if got.OAuthGroupDefaultShare == nil || *got.OAuthGroupDefaultShare != "true" {
		t.Errorf("expected JSON bool true to decode as \"true\", got %v", got.OAuthGroupDefaultShare)
	}
}

// TestGetOAuthConfigUnionShapes covers the int|str and bool|str unions: a JSON
// number timeout, a JSON false share value, and the string "members".
func TestGetOAuthConfigUnionShapes(t *testing.T) {
	cases := []struct {
		name, body, timeout, share string
	}{
		{"int timeout and false share", `{"OAUTH_TIMEOUT": 30, "OAUTH_CLIENT_TIMEOUT": 45, "OAUTH_GROUP_DEFAULT_SHARE": false}`, "30", "false"},
		{"string timeout and members share", `{"OAUTH_TIMEOUT": "30", "OAUTH_CLIENT_TIMEOUT": "", "OAUTH_GROUP_DEFAULT_SHARE": "members"}`, "30", "members"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tc.body))
			})
			c, server := newLDAPConfigTestClient(handler)
			defer server.Close()
			got, err := c.GetOAuthConfig(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if got.OAuthTimeout == nil || *got.OAuthTimeout != tc.timeout {
				t.Errorf("timeout: got %v want %q", got.OAuthTimeout, tc.timeout)
			}
			if got.OAuthGroupDefaultShare == nil || *got.OAuthGroupDefaultShare != tc.share {
				t.Errorf("share: got %v want %q", got.OAuthGroupDefaultShare, tc.share)
			}
		})
	}
}

// TestGetOAuthConfigV0100Shape: v0.10.x has no ENABLE_OAUTH key.
func TestGetOAuthConfigV0100Shape(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ENABLE_OAUTH_SIGNUP": true}`))
	})
	c, server := newLDAPConfigTestClient(handler)
	defer server.Close()
	got, err := c.GetOAuthConfig(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.EnableOAuth != nil || got.OAuthTimeout != nil || got.OAuthGroupDefaultShare != nil {
		t.Errorf("expected absent keys to be nil: %+v", got)
	}
}

// TestSetOAuthConfigPartialBody pins the literal wire keys and proves the
// request is a genuine partial update: only the fields that are set appear,
// every nil field is omitted (never sent as null).
func TestSetOAuthConfigPartialBody(t *testing.T) {
	var body map[string]any
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/auths/admin/config/oauth" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fullOAuthResponse))
	})
	c, server := newLDAPConfigTestClient(handler)
	defer server.Close()

	name := "Corp SSO"
	f := false
	empty := ""
	if _, err := c.SetOAuthConfig(context.Background(), OAuthConfigForm{
		OAuthProviderName: &name, EnableOAuthSignup: &f, OAuthAudience: &empty,
	}); err != nil {
		t.Fatalf("SetOAuthConfig failed: %v", err)
	}
	want := map[string]any{"OAUTH_PROVIDER_NAME": "Corp SSO", "ENABLE_OAUTH_SIGNUP": false, "OAUTH_AUDIENCE": ""}
	if len(body) != len(want) {
		t.Fatalf("expected exactly %d keys, got body %v", len(want), body)
	}
	for k, v := range want {
		if got, ok := body[k]; !ok || got != v {
			t.Errorf("wire key %q: expected %v, got %v (present=%v)", k, v, got, ok)
		}
	}
}

// TestSetOAuthConfigEveryWireKey pins all 34 literal wire keys and the special
// encodings: numeric timeouts as JSON numbers, "true"/"false" share as JSON
// bools, "members" as a JSON string.
func TestSetOAuthConfigEveryWireKey(t *testing.T) {
	var body map[string]any
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	})
	c, server := newLDAPConfigTestClient(handler)
	defer server.Close()

	tr := true
	s := func(v string) *string { return &v }
	form := OAuthConfigForm{
		EnableOAuth: &tr, EnableOAuthSignup: &tr, OAuthMergeAccountsByEmail: &tr, OAuthAutoRedirect: &tr,
		OAuthAllowedDomains: s("a.com,b.com"), OAuthBlockedGroups: s("[]"),
		EnableOAuthRoleManagement: &tr, OAuthRolesClaim: s("roles"), OAuthAdminRoles: s("admin"), OAuthAllowedRoles: s("user,admin"),
		EnableOAuthGroupManagement: &tr, EnableOAuthGroupCreation: &tr, OAuthGroupClaim: s("groups"), OAuthGroupDefaultShare: s("members"),
		OAuthProviderName: s("SSO"), OpenIDProviderURL: s("https://idp"), OAuthClientID: s("cid"), OAuthClientSecret: s("sec"),
		OpenIDRedirectURI: s("https://app/cb"), OAuthScopes: s("openid"), OAuthCodeChallengeMethod: s("S256"),
		OAuthTokenEndpointAuthMethod: s("client_secret_post"), OpenIDEndSessionEndpoint: s("https://idp/logout"),
		OAuthTimeout: s("30"), OAuthClientTimeout: s(""),
		OAuthEmailClaim: s("email"), OAuthUsernameClaim: s("name"), OAuthPictureClaim: s("picture"), OAuthSubClaim: s("sub"),
		OAuthAudience: s("aud"), OAuthUpdateEmailOnLogin: &tr, OAuthUpdateNameOnLogin: &tr, OAuthUpdatePictureOnLogin: &tr,
		OAuthRefreshTokenIncludeScope: &tr,
	}
	if _, err := c.SetOAuthConfig(context.Background(), form); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"ENABLE_OAUTH": true, "ENABLE_OAUTH_SIGNUP": true, "OAUTH_MERGE_ACCOUNTS_BY_EMAIL": true, "OAUTH_AUTO_REDIRECT": true,
		"OAUTH_ALLOWED_DOMAINS": "a.com,b.com", "OAUTH_BLOCKED_GROUPS": "[]",
		"ENABLE_OAUTH_ROLE_MANAGEMENT": true, "OAUTH_ROLES_CLAIM": "roles", "OAUTH_ADMIN_ROLES": "admin", "OAUTH_ALLOWED_ROLES": "user,admin",
		"ENABLE_OAUTH_GROUP_MANAGEMENT": true, "ENABLE_OAUTH_GROUP_CREATION": true, "OAUTH_GROUP_CLAIM": "groups",
		"OAUTH_GROUP_DEFAULT_SHARE": "members", "OAUTH_PROVIDER_NAME": "SSO", "OPENID_PROVIDER_URL": "https://idp",
		"OAUTH_CLIENT_ID": "cid", "OAUTH_CLIENT_SECRET": "sec", "OPENID_REDIRECT_URI": "https://app/cb", "OAUTH_SCOPES": "openid",
		"OAUTH_CODE_CHALLENGE_METHOD": "S256", "OAUTH_TOKEN_ENDPOINT_AUTH_METHOD": "client_secret_post",
		"OPENID_END_SESSION_ENDPOINT": "https://idp/logout", "OAUTH_TIMEOUT": float64(30), "OAUTH_CLIENT_TIMEOUT": "",
		"OAUTH_EMAIL_CLAIM": "email", "OAUTH_USERNAME_CLAIM": "name", "OAUTH_PICTURE_CLAIM": "picture", "OAUTH_SUB_CLAIM": "sub",
		"OAUTH_AUDIENCE": "aud", "OAUTH_UPDATE_EMAIL_ON_LOGIN": true, "OAUTH_UPDATE_NAME_ON_LOGIN": true,
		"OAUTH_UPDATE_PICTURE_ON_LOGIN": true, "OAUTH_REFRESH_TOKEN_INCLUDE_SCOPE": true,
	}
	if len(want) != 34 {
		t.Fatalf("test table must cover 34 keys, has %d", len(want))
	}
	if len(body) != len(want) {
		t.Errorf("expected %d keys on the wire, got %d: %v", len(want), len(body), body)
	}
	for k, v := range want {
		if got, ok := body[k]; !ok || got != v {
			t.Errorf("wire key %q: expected %#v, got %#v (present=%v)", k, v, got, ok)
		}
	}
}

func TestSetOAuthConfigShareBoolEncoding(t *testing.T) {
	for share, want := range map[string]any{"true": true, "false": false, "members": "members"} {
		var body map[string]any
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&body)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{}`))
		})
		c, server := newLDAPConfigTestClient(handler)
		v := share
		if _, err := c.SetOAuthConfig(context.Background(), OAuthConfigForm{OAuthGroupDefaultShare: &v}); err != nil {
			t.Fatal(err)
		}
		server.Close()
		if body["OAUTH_GROUP_DEFAULT_SHARE"] != want {
			t.Errorf("share %q: expected wire value %#v, got %#v", share, want, body["OAUTH_GROUP_DEFAULT_SHARE"])
		}
	}
}
