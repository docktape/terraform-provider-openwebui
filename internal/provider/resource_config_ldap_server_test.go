package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

func ldapServerHCL(attrs string) string {
	return `
resource "openwebui_ldap_server_config" "test" {
  label       = "Corp LDAP"
  host        = "ldap.example.com"
  search_base = "ou=people,dc=example,dc=com"
` + attrs + `
}
`
}

func stockLDAPServerForm() client.LDAPServerConfigForm {
	port := int64(636)
	tru := true
	s := func(v string) *string { return &v }
	return client.LDAPServerConfigForm{
		Label: "Old", Host: "old.example.com", Port: &port, AttributeForMail: s("mail"), AttributeForUsername: s("uid"),
		AppDN: "cn=app,dc=example,dc=com", AppDNPassword: "s3cret", SearchBase: "ou=old", SearchFilters: s("(objectClass=person)"),
		UseTLS: &tru, CertificatePath: nil, ValidateCert: &tru, Ciphers: nil,
		EnableGroupManagement: &tru, EnableGroupCreation: &tru, AttributeForGroups: s("memberOf"),
	}
}

// Unknown (unset on first apply) and null plan values keep the current
// server value; only known values are overlaid.
func TestLDAPServerOverlayKeepsUnknownAndNull(t *testing.T) {
	m := ldapServerConfigModel{
		Label:                 types.StringValue("L"),
		Host:                  types.StringValue("H"),
		Port:                  types.Int64Unknown(),
		AttributeForMail:      types.StringUnknown(),
		AttributeForUsername:  types.StringNull(),
		AppDN:                 types.StringUnknown(),
		AppDNPassword:         types.StringUnknown(),
		SearchBase:            types.StringValue("sb"),
		SearchFilters:         types.StringUnknown(),
		UseTLS:                types.BoolUnknown(),
		CertificatePath:       types.StringUnknown(),
		ValidateCert:          types.BoolValue(false),
		Ciphers:               types.StringUnknown(),
		EnableGroupManagement: types.BoolUnknown(),
		EnableGroupCreation:   types.BoolNull(),
		AttributeForGroups:    types.StringUnknown(),
	}
	cur := stockLDAPServerForm()
	f := overlayLDAPServerConfigPlan(cur, m)
	if f.Label != "L" || f.Host != "H" || f.SearchBase != "sb" {
		t.Fatalf("required plan values not applied: %+v", f)
	}
	if f.AppDN != "cn=app,dc=example,dc=com" || f.AppDNPassword != "s3cret" {
		t.Fatalf("unset app_dn/app_dn_password must keep the server values, got %q/%q", f.AppDN, f.AppDNPassword)
	}
	if f.Port == nil || *f.Port != 636 || *f.AttributeForMail != "mail" || *f.AttributeForUsername != "uid" ||
		*f.SearchFilters != "(objectClass=person)" || !*f.UseTLS || f.CertificatePath != nil || f.Ciphers != nil ||
		!*f.EnableGroupManagement || !*f.EnableGroupCreation || *f.AttributeForGroups != "memberOf" {
		t.Fatalf("unset fields must keep the server values, got %+v", f)
	}
	if f.ValidateCert == nil || *f.ValidateCert {
		t.Fatalf("explicit validate_cert=false must be applied")
	}
	// The input form is not modified.
	if cur.Label != "Old" || !*cur.ValidateCert {
		t.Fatalf("overlay mutated the current form: %+v", cur)
	}
}

func TestLDAPServerOverlayAppliesKnownValues(t *testing.T) {
	m := ldapServerConfigModel{
		Label: types.StringValue("L"), Host: types.StringValue("H"), Port: types.Int64Value(389),
		AttributeForMail: types.StringValue("email"), AttributeForUsername: types.StringValue("sAMAccountName"),
		AppDN: types.StringValue(""), AppDNPassword: types.StringValue(""), SearchBase: types.StringValue("sb"),
		SearchFilters: types.StringValue(""), UseTLS: types.BoolValue(false), CertificatePath: types.StringValue(""),
		ValidateCert: types.BoolValue(false), Ciphers: types.StringValue("HIGH"),
		EnableGroupManagement: types.BoolValue(false), EnableGroupCreation: types.BoolValue(false),
		AttributeForGroups: types.StringValue("groups"),
	}
	f := overlayLDAPServerConfigPlan(stockLDAPServerForm(), m)
	if *f.Port != 389 || *f.AttributeForMail != "email" || *f.AttributeForUsername != "sAMAccountName" ||
		f.AppDN != "" || f.AppDNPassword != "" || *f.SearchFilters != "" || *f.UseTLS || *f.CertificatePath != "" ||
		*f.ValidateCert || *f.Ciphers != "HIGH" || *f.EnableGroupManagement || *f.EnableGroupCreation || *f.AttributeForGroups != "groups" {
		t.Fatalf("known plan values not applied: %+v", f)
	}
}

// Against an older server the group fields are absent: unset plan values
// leave them nil, so the request omits them.
func TestLDAPServerOverlayOlderServer(t *testing.T) {
	m := ldapServerConfigModel{
		Label: types.StringValue("L"), Host: types.StringValue("H"), SearchBase: types.StringValue("sb"),
		Port: types.Int64Unknown(), AttributeForMail: types.StringUnknown(), AttributeForUsername: types.StringUnknown(),
		AppDN: types.StringUnknown(), AppDNPassword: types.StringUnknown(), SearchFilters: types.StringUnknown(),
		UseTLS: types.BoolUnknown(), CertificatePath: types.StringUnknown(), ValidateCert: types.BoolUnknown(),
		Ciphers: types.StringUnknown(), EnableGroupManagement: types.BoolUnknown(), EnableGroupCreation: types.BoolUnknown(),
		AttributeForGroups: types.StringUnknown(),
	}
	f := overlayLDAPServerConfigPlan(client.LDAPServerConfigForm{Label: "x", Host: "y", SearchBase: "z", AppDN: "dn"}, m)
	if f.EnableGroupManagement != nil || f.EnableGroupCreation != nil || f.AttributeForGroups != nil || f.AppDN != "dn" {
		t.Fatalf("unexpected overlay on older server form: %+v", f)
	}
}

func TestLDAPServerModelFromFormRoundTrip(t *testing.T) {
	port := int64(636)
	tru := true
	fls := false
	s := func(v string) *string { return &v }
	form := client.LDAPServerConfigForm{
		Label: "L", Host: "H", Port: &port, AttributeForMail: s("mail"), AttributeForUsername: s("uid"),
		AppDN: "dn", AppDNPassword: "pw", SearchBase: "sb", SearchFilters: s(""),
		UseTLS: &tru, ValidateCert: &fls, Ciphers: s("ALL"),
		EnableGroupManagement: &tru, EnableGroupCreation: &fls, AttributeForGroups: s("memberOf"),
	}
	m := ldapServerConfigModelFromForm(form)
	if m.ID.ValueString() != "ldap_server" || m.Port.ValueInt64() != 636 || m.AppDNPassword.ValueString() != "pw" {
		t.Fatalf("unexpected model: %+v", m)
	}
	if !m.CertificatePath.IsNull() {
		t.Fatalf("expected null certificate_path")
	}
	if m.SearchFilters.IsNull() || m.SearchFilters.ValueString() != "" {
		t.Fatalf("expected search_filters empty string, got %v", m.SearchFilters)
	}
	back := overlayLDAPServerConfigPlan(client.LDAPServerConfigForm{}, m)
	if back.Port == nil || *back.Port != 636 || back.EnableGroupManagement == nil || !*back.EnableGroupManagement ||
		back.ValidateCert == nil || *back.ValidateCert || back.Ciphers == nil || *back.Ciphers != "ALL" {
		t.Fatalf("round-trip lost values: %+v", back)
	}
}

// Older servers omit the group keys and may return a null port; both must
// flatten to null (not zero values).
func TestLDAPServerModelFromFormOlderServer(t *testing.T) {
	m := ldapServerConfigModelFromForm(client.LDAPServerConfigForm{Label: "L", Host: "H", SearchBase: "sb"})
	if !m.Port.IsNull() || !m.EnableGroupManagement.IsNull() || !m.EnableGroupCreation.IsNull() || !m.AttributeForGroups.IsNull() {
		t.Fatalf("expected null optional fields, got %+v", m)
	}
}

func TestLDAPServerConfigValidators_RequiredNonEmpty(t *testing.T) {
	for _, attr := range []string{"label", "host", "search_base"} {
		t.Run(attr+" rejects empty", func(t *testing.T) {
			planOnlyRejected(t, ldapServerHCLOverride(attr), `(?s)`+attr+`.*(at\s+least|length)`)
		})
	}
	for _, attr := range []string{"attribute_for_mail", "attribute_for_username"} {
		t.Run(attr+" rejects empty", func(t *testing.T) {
			planOnlyRejected(t, ldapServerHCL(`  `+attr+` = ""`), `(?s)`+attr+`.*(at\s+least|length)`)
		})
		t.Run(attr+" accepts value", func(t *testing.T) {
			planOnlyValid(t, ldapServerHCL(`  `+attr+` = "cn"`))
		})
	}
}

func ldapServerHCLOverride(empty string) string {
	vals := map[string]string{"label": "Corp", "host": "ldap.example.com", "search_base": "ou=p"}
	vals[empty] = ""
	return `
resource "openwebui_ldap_server_config" "test" {
  label       = "` + vals["label"] + `"
  host        = "` + vals["host"] + `"
  search_base = "` + vals["search_base"] + `"
}
`
}

func TestLDAPServerConfigValidators_GroupManagementNeedsAttribute(t *testing.T) {
	for _, blank := range []string{"", "   "} {
		t.Run("rejects blank attribute_for_groups '"+blank+"'", func(t *testing.T) {
			planOnlyRejected(t, ldapServerHCL(`
  enable_group_management = true
  attribute_for_groups    = "`+blank+`"`), `(?s)attribute_for_groups.*(non-blank|blank)`)
		})
	}
	t.Run("accepts management with attribute", func(t *testing.T) {
		planOnlyValid(t, ldapServerHCL(`
  enable_group_management = true
  attribute_for_groups    = "memberOf"`))
	})
	t.Run("accepts management with attribute unset", func(t *testing.T) {
		planOnlyValid(t, ldapServerHCL(`  enable_group_management = true`))
	})
	t.Run("accepts blank attribute when management disabled", func(t *testing.T) {
		planOnlyValid(t, ldapServerHCL(`
  enable_group_management = false
  attribute_for_groups    = ""`))
	})
}

// fakeLDAPServer is a stateful stand-in for GET/POST
// /api/v1/auths/admin/config/ldap/server at v0.11.0: the POST is a
// full-object replace through the LdapServerConfig model, so an omitted
// field falls back to the Pydantic default and null is rejected for
// non-nullable fields.
type fakeLDAPServer struct {
	mu    sync.Mutex
	store map[string]any
	posts int
}

var ldapServerModelDefaults = map[string]any{
	"port": nil, "attribute_for_mail": "mail", "attribute_for_username": "uid", "search_filters": "",
	"use_tls": true, "certificate_path": nil, "validate_cert": true, "ciphers": "ALL",
	"enable_group_management": false, "enable_group_creation": false, "attribute_for_groups": "memberOf",
}

func (f *fakeLDAPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if r.URL.Path != "/api/v1/auths/admin/config/ldap/server" {
		http.NotFound(w, r)
		return
	}
	if r.Method == http.MethodPost {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, `{"detail":"invalid"}`, http.StatusUnprocessableEntity)
			return
		}
		next := map[string]any{}
		for _, k := range []string{"label", "host", "app_dn", "app_dn_password", "search_base"} {
			v, ok := body[k].(string)
			if !ok {
				http.Error(w, `{"detail":"missing `+k+`"}`, http.StatusUnprocessableEntity)
				return
			}
			next[k] = v
		}
		for k, def := range ldapServerModelDefaults {
			v, present := body[k]
			if !present {
				v = def
			}
			nullable := k == "port" || k == "certificate_path" || k == "ciphers"
			if v == nil && !nullable {
				http.Error(w, `{"detail":"`+k+` must not be null"}`, http.StatusUnprocessableEntity)
				return
			}
			next[k] = v
		}
		if next["enable_group_management"] == true && strings.TrimSpace(fmt.Sprint(next["attribute_for_groups"])) == "" {
			http.Error(w, `{"detail":"attribute_for_groups is required"}`, http.StatusBadRequest)
			return
		}
		f.store = next
		f.posts++
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(f.store)
}

// TestLDAPServerConfigAgainstFakeServer shows that a first apply with a
// partial configuration keeps every unset server value (it used to reset
// app_dn/app_dn_password and the other unset fields to defaults).
func TestLDAPServerConfigAgainstFakeServer(t *testing.T) {
	fake := &fakeLDAPServer{store: map[string]any{
		"label": "Old", "host": "old.example.com", "port": 636, "attribute_for_mail": "email",
		"attribute_for_username": "sAMAccountName", "app_dn": "cn=app,dc=example,dc=com", "app_dn_password": "s3cret",
		"search_base": "ou=old", "search_filters": "(objectClass=person)", "use_tls": false, "certificate_path": "/ca.pem",
		"validate_cert": false, "ciphers": nil, "enable_group_management": true, "enable_group_creation": true,
		"attribute_for_groups": "groups",
	}}
	server := httptest.NewServer(fake)
	defer server.Close()
	t.Setenv("OPENWEBUI_ENDPOINT", server.URL)
	t.Setenv("OPENWEBUI_TOKEN", "test-token")

	const name = "openwebui_ldap_server_config.test"
	partial := ldapServerHCL(``)
	cleared := ldapServerHCL(`
  app_dn           = ""
  app_dn_password  = ""
  certificate_path = ""
  ciphers          = "HIGH"`)

	tfresource.UnitTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []tfresource.TestStep{
			{
				Config: partial,
				Check: tfresource.ComposeTestCheckFunc(
					tfresource.TestCheckResourceAttr(name, "label", "Corp LDAP"),
					tfresource.TestCheckResourceAttr(name, "app_dn", "cn=app,dc=example,dc=com"),
					tfresource.TestCheckResourceAttr(name, "app_dn_password", "s3cret"),
					tfresource.TestCheckResourceAttr(name, "port", "636"),
					tfresource.TestCheckResourceAttr(name, "attribute_for_mail", "email"),
					tfresource.TestCheckResourceAttr(name, "use_tls", "false"),
					tfresource.TestCheckResourceAttr(name, "certificate_path", "/ca.pem"),
					tfresource.TestCheckNoResourceAttr(name, "ciphers"),
					tfresource.TestCheckResourceAttr(name, "enable_group_creation", "true"),
					tfresource.TestCheckResourceAttr(name, "attribute_for_groups", "groups"),
				),
			},
			{Config: partial, PlanOnly: true},
			{
				Config: cleared,
				Check: tfresource.ComposeTestCheckFunc(
					tfresource.TestCheckResourceAttr(name, "app_dn", ""),
					tfresource.TestCheckResourceAttr(name, "app_dn_password", ""),
					tfresource.TestCheckResourceAttr(name, "certificate_path", ""),
					tfresource.TestCheckResourceAttr(name, "ciphers", "HIGH"),
					tfresource.TestCheckResourceAttr(name, "port", "636"),
				),
			},
			{Config: cleared, PlanOnly: true},
			// Removing the attributes from the configuration keeps the
			// last-read values.
			{Config: partial, Check: tfresource.TestCheckResourceAttr(name, "ciphers", "HIGH")},
			{Config: partial, PlanOnly: true},
		},
	})

	if fake.store["app_dn_password"] != "" || fake.store["ciphers"] != "HIGH" || fake.store["attribute_for_mail"] != "email" {
		t.Errorf("unexpected final server state %v", fake.store)
	}
}

// TestLDAPServerGroupAttributeCheck covers the apply-time check: a blank
// attribute_for_groups that comes from the server (L2) or from a plan value
// copied from state (L3) is rejected before the POST.
func TestLDAPServerGroupAttributeCheck(t *testing.T) {
	blank, set, on, off := "", "memberOf", true, false
	current := client.LDAPServerConfigForm{Label: "x", Host: "h", SearchBase: "b", EnableGroupManagement: &off, AttributeForGroups: &blank}
	plan := ldapServerConfigModelFromForm(current)

	// From the server: management enabled in config, attribute unset (unknown).
	m := plan
	m.EnableGroupManagement = types.BoolValue(true)
	m.AttributeForGroups = types.StringUnknown()
	if diags := checkLDAPServerGroupAttribute(overlayLDAPServerConfigPlan(current, m)); !diags.HasError() {
		t.Errorf("blank server attribute_for_groups must be rejected")
	}

	// From state: the server now holds a value, but the plan carries the
	// blank state value (UseStateForUnknown under -refresh=false).
	current.AttributeForGroups = &set
	m.AttributeForGroups = types.StringValue("  ")
	if diags := checkLDAPServerGroupAttribute(overlayLDAPServerConfigPlan(current, m)); !diags.HasError() {
		t.Errorf("blank state attribute_for_groups must be rejected")
	}

	// Accepted: a non-blank value, management off, or an older server
	// without group settings (nil).
	m.AttributeForGroups = types.StringValue("groups")
	if diags := checkLDAPServerGroupAttribute(overlayLDAPServerConfigPlan(current, m)); diags.HasError() {
		t.Errorf("non-blank attribute_for_groups rejected: %v", diags)
	}
	if diags := checkLDAPServerGroupAttribute(client.LDAPServerConfigForm{EnableGroupManagement: &off, AttributeForGroups: &blank}); diags.HasError() {
		t.Errorf("blank attribute with management off rejected: %v", diags)
	}
	if diags := checkLDAPServerGroupAttribute(client.LDAPServerConfigForm{EnableGroupManagement: &on}); diags.HasError() {
		t.Errorf("older server (nil attribute) rejected: %v", diags)
	}
}

// TestLDAPServerGroupAttributeFromServerFailsBeforePost runs the provider
// against a fake that holds a blank attribute_for_groups with group
// management off: enabling management without setting the attribute fails
// with a clear diagnostic and nothing is POSTed.
func TestLDAPServerGroupAttributeFromServerFailsBeforePost(t *testing.T) {
	fake := &fakeLDAPServer{store: map[string]any{
		"label": "Old", "host": "old.example.com", "port": 389, "attribute_for_mail": "mail",
		"attribute_for_username": "uid", "app_dn": "", "app_dn_password": "",
		"search_base": "ou=old", "search_filters": "", "use_tls": true, "certificate_path": "",
		"validate_cert": true, "ciphers": "ALL", "enable_group_management": false, "enable_group_creation": false,
		"attribute_for_groups": "",
	}}
	server := httptest.NewServer(fake)
	defer server.Close()
	t.Setenv("OPENWEBUI_ENDPOINT", server.URL)
	t.Setenv("OPENWEBUI_TOKEN", "test-token")

	tfresource.UnitTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []tfresource.TestStep{
			{
				Config:      ldapServerHCL(`  enable_group_management = true`),
				ExpectError: regexp.MustCompile(`Blank attribute_for_groups`),
			},
		},
	})
	if fake.posts != 0 {
		t.Errorf("expected no POST, got %d", fake.posts)
	}
}
