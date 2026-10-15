package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

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
		for _, ok := range []string{"", "30"} {
			planOnlyValid(t, oauthHCL(`  `+attr+` = "`+ok+`"`))
		}
		for _, bad := range []string{"30s", "-1", "1.5"} {
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
