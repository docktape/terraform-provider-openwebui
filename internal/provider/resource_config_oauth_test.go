package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

func oauthHCL(attrs string) string {
	return `
resource "openwebui_oauth_config" "test" {
` + attrs + `
}
`
}

// All-unknown plan (a first apply that sets nothing) must send an empty body:
// every unknown/null attribute is omitted so the server keeps its values.
func TestOAuthFormFromModelOmitsUnknownAndNull(t *testing.T) {
	m := oauthConfigModelFromForm(client.OAuthConfigForm{})
	m.EnableOAuth = types.BoolUnknown()
	m.OAuthProviderName = types.StringUnknown()
	m.OAuthClientSecret = types.StringUnknown()
	m.OAuthSubClaim = types.StringNull()
	m.OAuthTimeout = types.StringUnknown()
	f := oauthConfigFormFromModel(m)
	if f != (client.OAuthConfigForm{}) {
		t.Fatalf("expected an empty form, got %+v", f)
	}
}

func TestOAuthFormFromModelPreservesFalseAndEmpty(t *testing.T) {
	m := oauthConfigModelFromForm(client.OAuthConfigForm{})
	m.OAuthAutoRedirect = types.BoolValue(false)
	m.OAuthAudience = types.StringValue("")
	m.OAuthTimeout = types.StringValue("")
	f := oauthConfigFormFromModel(m)
	if f.OAuthAutoRedirect == nil || *f.OAuthAutoRedirect {
		t.Errorf("expected explicit false to be sent")
	}
	if f.OAuthAudience == nil || *f.OAuthAudience != "" || f.OAuthTimeout == nil || *f.OAuthTimeout != "" {
		t.Errorf("expected explicit empty strings to be sent, got %+v", f)
	}
}

// Round trip for the three shapes the string-typed union/nullable fields take:
// null (absent/None), "" (the os.getenv default) and populated.
func TestOAuthModelFromFormNullEmptyPopulated(t *testing.T) {
	s := func(v string) *string { return &v }
	tru := true
	form := client.OAuthConfigForm{
		EnableOAuth:              &tru,
		OAuthCodeChallengeMethod: nil,
		OAuthTimeout:             s(""),
		OAuthClientTimeout:       s("45"),
		OAuthGroupDefaultShare:   s("members"),
		OAuthClientSecret:        s("plain"),
		OAuthSubClaim:            nil,
	}
	m := oauthConfigModelFromForm(form)
	if m.ID.ValueString() != "oauth_config" {
		t.Errorf("unexpected id %q", m.ID.ValueString())
	}
	if !m.OAuthCodeChallengeMethod.IsNull() || !m.OAuthSubClaim.IsNull() {
		t.Errorf("expected null for nil fields")
	}
	if m.OAuthTimeout.IsNull() || m.OAuthTimeout.ValueString() != "" {
		t.Errorf("expected empty-string timeout, got %v", m.OAuthTimeout)
	}
	if m.OAuthClientTimeout.ValueString() != "45" || m.OAuthGroupDefaultShare.ValueString() != "members" ||
		m.OAuthClientSecret.ValueString() != "plain" || !m.EnableOAuth.ValueBool() {
		t.Errorf("unexpected populated values: %+v", m)
	}
	if !m.EnableOAuthSignup.IsNull() {
		t.Errorf("expected absent bool (v0.10.x shape) to be null")
	}

	back := oauthConfigFormFromModel(m)
	if back.OAuthTimeout == nil || *back.OAuthTimeout != "" || back.OAuthClientTimeout == nil || *back.OAuthClientTimeout != "45" ||
		back.OAuthCodeChallengeMethod != nil || back.OAuthGroupDefaultShare == nil || *back.OAuthGroupDefaultShare != "members" {
		t.Errorf("round trip lost values: %+v", back)
	}
}

func TestOAuthConfigValidators_CommaLists(t *testing.T) {
	for _, attr := range []string{"oauth_allowed_domains", "oauth_admin_roles", "oauth_allowed_roles"} {
		for _, ok := range []string{"", "*", "a.com,b.com", "Super Admin,user"} {
			t.Run(attr+" accepts '"+ok+"'", func(t *testing.T) {
				planOnlyValid(t, oauthHCL(`  `+attr+` = "`+ok+`"`))
			})
		}
		for _, bad := range []string{"a.com, b.com", " a.com", "a.com,", ",a.com", "a,,b"} {
			t.Run(attr+" rejects '"+bad+"'", func(t *testing.T) {
				planOnlyRejected(t, oauthHCL(`  `+attr+` = "`+bad+`"`), `(?s)`+attr+`.*(comma-separated|Attribute)`)
			})
		}
	}
}

func TestOAuthConfigValidators_ScalarFields(t *testing.T) {
	for _, ok := range []string{"true", "false", "members"} {
		planOnlyValid(t, oauthHCL(`  oauth_group_default_share = "`+ok+`"`))
	}
	planOnlyRejected(t, oauthHCL(`  oauth_group_default_share = "everyone"`), `(?s)oauth_group_default_share.*(one of|must be)`)

	for _, attr := range []string{"oauth_timeout", "oauth_client_timeout"} {
		for _, ok := range []string{"", "0", "30"} {
			planOnlyValid(t, oauthHCL(`  `+attr+` = "`+ok+`"`))
		}
		for _, bad := range []string{"30s", "-1", "1.5", "030", "00", "05"} {
			planOnlyRejected(t, oauthHCL(`  `+attr+` = "`+bad+`"`), `(?s)`+attr+`.*whole number`)
		}
	}

	for _, ok := range []string{"", "S256"} {
		planOnlyValid(t, oauthHCL(`  oauth_code_challenge_method = "`+ok+`"`))
	}
	planOnlyRejected(t, oauthHCL(`  oauth_code_challenge_method = "plain"`), `(?s)oauth_code_challenge_method.*(one of|must be)`)
}

func TestOAuthConfigSchemaShape(t *testing.T) {
	var resp resource.SchemaResponse
	NewOAuthConfigResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if len(resp.Schema.Attributes) != 35 {
		t.Fatalf("expected id plus 34 attributes, got %d", len(resp.Schema.Attributes))
	}
	for name, attr := range resp.Schema.Attributes {
		if attr.GetDescription() == "" || attr.GetMarkdownDescription() == "" {
			t.Errorf("attribute %q needs both Description and MarkdownDescription", name)
		}
		if name == "id" {
			continue
		}
		if !attr.IsOptional() || !attr.IsComputed() {
			t.Errorf("attribute %q must be Optional+Computed", name)
		}
		if attr.IsSensitive() != (name == "oauth_client_secret") {
			t.Errorf("attribute %q: unexpected Sensitive=%v", name, attr.IsSensitive())
		}
	}
}

// fakeOAuthServer stands in for GET/POST /api/v1/auths/admin/config/oauth.
// The POST is a partial update (exclude_none); keys outside `known` are
// dropped, mimicking the response model of the emulated version.
type fakeOAuthServer struct {
	mu    sync.Mutex
	store map[string]any
	known map[string]bool
	gets  int
	posts int
}

func (f *fakeOAuthServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if r.URL.Path != "/api/v1/auths/admin/config/oauth" {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		f.gets++
	case http.MethodPost:
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, `{"detail":"invalid"}`, http.StatusUnprocessableEntity)
			return
		}
		for k, v := range body {
			if f.known[k] && v != nil {
				f.store[k] = v
			}
		}
		f.posts++
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(f.store)
}

func newFakeOAuthServer(withEnableOAuth bool) *fakeOAuthServer {
	f := &fakeOAuthServer{
		store: map[string]any{"ENABLE_OAUTH_SIGNUP": false, "OAUTH_PROVIDER_NAME": "SSO"},
		known: map[string]bool{"ENABLE_OAUTH_SIGNUP": true, "OAUTH_PROVIDER_NAME": true},
	}
	if withEnableOAuth {
		f.store["ENABLE_OAUTH"] = true
		f.known["ENABLE_OAUTH"] = true
	}
	return f
}

// TestOAuthEnableOAuthGuard: setting enable_oauth against a server that does
// not report ENABLE_OAUTH (v0.10.x) fails before anything is written, while a
// v0.11.0-like server accepts it; leaving it unset needs no extra read.
func TestOAuthEnableOAuthGuard(t *testing.T) {
	const name = "openwebui_oauth_config.test"

	t.Run("v0.10.x rejects before POST", func(t *testing.T) {
		fake := newFakeOAuthServer(false)
		server := httptest.NewServer(fake)
		defer server.Close()
		t.Setenv("OPENWEBUI_ENDPOINT", server.URL)
		t.Setenv("OPENWEBUI_TOKEN", "test-token")
		tfresource.UnitTest(t, tfresource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
			Steps: []tfresource.TestStep{{
				Config: oauthHCL(`  enable_oauth        = false
  oauth_provider_name = "Corp"`),
				ExpectError: regexp.MustCompile(`(?s)Unsupported by this Open WebUI.*v0\.11\.0`),
			}},
		})
		if fake.posts != 0 || fake.store["OAUTH_PROVIDER_NAME"] != "SSO" {
			t.Errorf("nothing must be written: posts=%d store=%v", fake.posts, fake.store)
		}
	})

	t.Run("v0.10.x without enable_oauth", func(t *testing.T) {
		fake := newFakeOAuthServer(false)
		server := httptest.NewServer(fake)
		defer server.Close()
		t.Setenv("OPENWEBUI_ENDPOINT", server.URL)
		t.Setenv("OPENWEBUI_TOKEN", "test-token")
		config := oauthHCL(`  oauth_provider_name = "Corp"`)
		tfresource.UnitTest(t, tfresource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
			Steps: []tfresource.TestStep{
				{Config: config, Check: tfresource.ComposeTestCheckFunc(
					tfresource.TestCheckResourceAttr(name, "oauth_provider_name", "Corp"),
					tfresource.TestCheckNoResourceAttr(name, "enable_oauth"),
				)},
				{Config: config, PlanOnly: true},
			},
		})
	})

	t.Run("v0.11.0 accepts", func(t *testing.T) {
		fake := newFakeOAuthServer(true)
		server := httptest.NewServer(fake)
		defer server.Close()
		t.Setenv("OPENWEBUI_ENDPOINT", server.URL)
		t.Setenv("OPENWEBUI_TOKEN", "test-token")
		config := oauthHCL(`  enable_oauth = false`)
		tfresource.UnitTest(t, tfresource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
			Steps: []tfresource.TestStep{
				{Config: config, Check: tfresource.TestCheckResourceAttr(name, "enable_oauth", "false")},
				{Config: config, PlanOnly: true},
			},
		})
		if fake.store["ENABLE_OAUTH"] != false {
			t.Errorf("ENABLE_OAUTH not written: %v", fake.store)
		}
	})
}
