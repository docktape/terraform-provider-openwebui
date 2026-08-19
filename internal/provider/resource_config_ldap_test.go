package provider

import (
	"testing"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestLDAPConfigModelRoundTrip(t *testing.T) {
	enableTrue := true
	enableFalse := false

	tests := []struct {
		name      string
		formValue *bool
	}{
		{
			name:      "true",
			formValue: &enableTrue,
		},
		{
			name:      "false",
			formValue: &enableFalse,
		},
		{
			name:      "nil",
			formValue: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create model from form
			form := client.LDAPConfigForm{
				EnableLDAP: tt.formValue,
			}

			model := ldapConfigModel{
				ID:         types.StringValue("ldap"),
				EnableLDAP: boolValueOrNull(form.EnableLDAP),
			}

			// Convert back to form-like structure
			expectedEnable := boolPtr(model.EnableLDAP)

			// Verify round-trip
			if (expectedEnable == nil) != (tt.formValue == nil) {
				t.Errorf("EnableLDAP nil mismatch: expected %v, got %v", tt.formValue == nil, expectedEnable == nil)
			}

			if expectedEnable != nil && tt.formValue != nil && *expectedEnable != *tt.formValue {
				t.Errorf("EnableLDAP value mismatch: expected %v, got %v", *tt.formValue, *expectedEnable)
			}
		})
	}
}
