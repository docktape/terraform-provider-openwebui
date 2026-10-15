package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

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

func TestLDAPServerFormFromModelOmitsUnknownAndNull(t *testing.T) {
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
	f := ldapServerConfigFormFromModel(m)
	if f.Port != nil || f.AttributeForMail != nil || f.AttributeForUsername != nil || f.SearchFilters != nil ||
		f.UseTLS != nil || f.CertificatePath != nil || f.Ciphers != nil || f.EnableGroupManagement != nil ||
		f.EnableGroupCreation != nil || f.AttributeForGroups != nil {
		t.Fatalf("expected unknown/null optional fields to be nil, got %+v", f)
	}
	if f.ValidateCert == nil || *f.ValidateCert {
		t.Fatalf("expected explicit validate_cert=false to be preserved")
	}
	if f.AppDN != "" || f.AppDNPassword != "" {
		t.Fatalf("expected unknown required strings to be sent as empty, got %q/%q", f.AppDN, f.AppDNPassword)
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
	back := ldapServerConfigFormFromModel(m)
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
