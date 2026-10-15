package provider

import (
	"context"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// These tests exercise the plan-time validators on values that Open WebUI
// silently clamps or drops. They run the real resource schema through
// `terraform plan` against a dummy endpoint: validation and planning never
// call the API, so no server is required. Each rejected value would otherwise
// plan cleanly and fail only after apply with "Provider produced inconsistent
// result after apply".

func setDummyProviderEnv(t *testing.T) {
	t.Helper()
	t.Setenv("OPENWEBUI_ENDPOINT", "http://127.0.0.1:1")
	t.Setenv("OPENWEBUI_TOKEN", "test-token")
}

func planOnlyValid(t *testing.T, config string) {
	t.Helper()
	setDummyProviderEnv(t)
	tfresource.UnitTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []tfresource.TestStep{
			{
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true, // a never-applied resource always plans a create
			},
		},
	})
}

func planOnlyRejected(t *testing.T, config string, errRegexp string) {
	t.Helper()
	setDummyProviderEnv(t)
	tfresource.UnitTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []tfresource.TestStep{
			{
				Config:      config,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(errRegexp),
			},
		},
	})
}

func chatConfigHCL(attrs string) string {
	return `
resource "openwebui_chat_config" "test" {
` + attrs + `
}
`
}

func adminConfigHCL(attrs string) string {
	return `
resource "openwebui_admin_config" "test" {
` + attrs + `
}
`
}

func TestChatConfigValidators_RetentionPercentage(t *testing.T) {
	for _, value := range []string{"9", "51", "0", "-1", "100"} {
		t.Run("rejects "+value, func(t *testing.T) {
			planOnlyRejected(t,
				chatConfigHCL(`  context_compaction_retention_percentage = `+value),
				`(?s)context_compaction_retention_percentage.*between\s+10\s+and\s+50`)
		})
	}
	for _, value := range []string{"10", "40", "50"} {
		t.Run("accepts "+value, func(t *testing.T) {
			planOnlyValid(t, chatConfigHCL(`  context_compaction_retention_percentage = `+value))
		})
	}
	t.Run("accepts unset", func(t *testing.T) {
		planOnlyValid(t, chatConfigHCL(`  enable_context_compaction = true`))
	})
}

func TestChatConfigValidators_TokenThresholdAndCap(t *testing.T) {
	for _, attr := range []string{"context_compaction_token_threshold", "context_compaction_token_cap"} {
		for _, value := range []string{"0", "-5"} {
			t.Run(attr+" rejects "+value, func(t *testing.T) {
				planOnlyRejected(t,
					chatConfigHCL(`  `+attr+` = `+value),
					`(?s)`+attr+`.*at\s+least\s+1`)
			})
		}
		t.Run(attr+" accepts 1", func(t *testing.T) {
			planOnlyValid(t, chatConfigHCL(`  `+attr+` = 1`))
		})
		t.Run(attr+" accepts 8000", func(t *testing.T) {
			planOnlyValid(t, chatConfigHCL(`  `+attr+` = 8000`))
		})
	}
}

func TestAdminConfigValidators_JWTExpiresIn(t *testing.T) {
	for _, value := range []string{"", "30", "abc", "30days", "1y", "d", "-", "1.d", "30 d", "-2"} {
		t.Run("rejects "+value, func(t *testing.T) {
			planOnlyRejected(t,
				adminConfigHCL(`  jwt_expires_in = "`+value+`"`),
				`(?s)jwt_expires_in\s+must\s+be\s+-1,\s+0,\s+or\s+a\s+number`)
		})
	}
	for _, value := range []string{"-1", "0", "30d", "24h", "1.5h", "500ms", "10s", "5m", "2w", "-3d"} {
		t.Run("accepts "+value, func(t *testing.T) {
			planOnlyValid(t, adminConfigHCL(`  jwt_expires_in = "`+value+`"`))
		})
	}
	t.Run("accepts unset", func(t *testing.T) {
		planOnlyValid(t, adminConfigHCL(`  enable_signup = true`))
	})
}

func TestAdminConfigValidators_ZeroIntOrEmptyFields(t *testing.T) {
	for _, attr := range []string{"folder_max_file_count", "automation_max_count", "automation_min_interval"} {
		t.Run(attr+" rejects 0", func(t *testing.T) {
			planOnlyRejected(t,
				adminConfigHCL(`  `+attr+` = 0`),
				`(?s)`+attr+`.*none\s+of`)
		})
		t.Run(attr+" accepts 100", func(t *testing.T) {
			planOnlyValid(t, adminConfigHCL(`  `+attr+` = 100`))
		})
	}
}

// TestConfigResourceSchemaValidatorsWired runs the validators attached to the
// real resource schema directly, proving the wiring (attribute -> rule) and
// not only the validator library's behaviour.
func TestConfigResourceSchemaValidatorsWired(t *testing.T) {
	ctx := context.Background()

	chat := &chatConfigResource{}
	chatResp := &resource.SchemaResponse{}
	chat.Schema(ctx, resource.SchemaRequest{}, chatResp)

	admin := &adminConfigResource{}
	adminResp := &resource.SchemaResponse{}
	admin.Schema(ctx, resource.SchemaRequest{}, adminResp)

	runInt64 := func(t *testing.T, vs []validator.Int64, value int64) bool {
		t.Helper()
		var failed bool
		for _, v := range vs {
			resp := &validator.Int64Response{}
			v.ValidateInt64(ctx, validator.Int64Request{ConfigValue: types.Int64Value(value)}, resp)
			failed = failed || resp.Diagnostics.HasError()
		}
		return failed
	}
	runString := func(t *testing.T, vs []validator.String, value string) bool {
		t.Helper()
		var failed bool
		for _, v := range vs {
			resp := &validator.StringResponse{}
			v.ValidateString(ctx, validator.StringRequest{ConfigValue: types.StringValue(value)}, resp)
			failed = failed || resp.Diagnostics.HasError()
		}
		return failed
	}

	int64Validators := func(t *testing.T, resp *resource.SchemaResponse, name string) []validator.Int64 {
		t.Helper()
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("attribute %q missing from schema", name)
		}
		a, ok := attr.(interface{ Int64Validators() []validator.Int64 })
		if !ok {
			t.Fatalf("attribute %q is not an int64 attribute", name)
		}
		vs := a.Int64Validators()
		if len(vs) == 0 {
			t.Fatalf("attribute %q has no validators", name)
		}
		return vs
	}

	retention := int64Validators(t, chatResp, "context_compaction_retention_percentage")
	for value, wantFail := range map[int64]bool{9: true, 10: false, 40: false, 50: false, 51: true} {
		if got := runInt64(t, retention, value); got != wantFail {
			t.Errorf("retention_percentage=%d: rejected=%v, want %v", value, got, wantFail)
		}
	}

	for _, name := range []string{"context_compaction_token_threshold", "context_compaction_token_cap"} {
		vs := int64Validators(t, chatResp, name)
		for value, wantFail := range map[int64]bool{-1: true, 0: true, 1: false, 80000: false} {
			if got := runInt64(t, vs, value); got != wantFail {
				t.Errorf("%s=%d: rejected=%v, want %v", name, value, got, wantFail)
			}
		}
	}

	for _, name := range []string{"folder_max_file_count", "automation_max_count", "automation_min_interval"} {
		vs := int64Validators(t, adminResp, name)
		for value, wantFail := range map[int64]bool{0: true, 1: false, 100: false} {
			if got := runInt64(t, vs, value); got != wantFail {
				t.Errorf("%s=%d: rejected=%v, want %v", name, value, got, wantFail)
			}
		}
	}

	jwtAttr, ok := adminResp.Schema.Attributes["jwt_expires_in"].(interface{ StringValidators() []validator.String })
	if !ok {
		t.Fatal("jwt_expires_in is not a string attribute")
	}
	jwt := jwtAttr.StringValidators()
	if len(jwt) == 0 {
		t.Fatal("jwt_expires_in has no validators")
	}
	for value, wantFail := range map[string]bool{
		"": true, "30": true, "abc": true, "1y": true,
		"-1": false, "0": false, "30d": false, "1.5h": false, "500ms": false, "2w": false,
	} {
		if got := runString(t, jwt, value); got != wantFail {
			t.Errorf("jwt_expires_in=%q: rejected=%v, want %v", value, got, wantFail)
		}
	}
}

// TestConfigResourceValidatorsIgnoreNullAndUnknown confirms the validators do
// not reject unset (null) or not-yet-known (unknown) values.
func TestConfigResourceValidatorsIgnoreNullAndUnknown(t *testing.T) {
	ctx := context.Background()
	chatResp := &resource.SchemaResponse{}
	(&chatConfigResource{}).Schema(ctx, resource.SchemaRequest{}, chatResp)
	adminResp := &resource.SchemaResponse{}
	(&adminConfigResource{}).Schema(ctx, resource.SchemaRequest{}, adminResp)

	for _, name := range []string{"context_compaction_retention_percentage", "context_compaction_token_threshold", "context_compaction_token_cap"} {
		a := chatResp.Schema.Attributes[name].(interface{ Int64Validators() []validator.Int64 }) //nolint:forcetypeassert // test-only
		for _, v := range []types.Int64{types.Int64Null(), types.Int64Unknown()} {
			for _, val := range a.Int64Validators() {
				resp := &validator.Int64Response{}
				val.ValidateInt64(ctx, validator.Int64Request{ConfigValue: v}, resp)
				if resp.Diagnostics.HasError() {
					t.Errorf("%s: validator rejected %s", name, v)
				}
			}
		}
	}
	jwt := adminResp.Schema.Attributes["jwt_expires_in"].(interface{ StringValidators() []validator.String }) //nolint:forcetypeassert // test-only
	for _, v := range []types.String{types.StringNull(), types.StringUnknown()} {
		for _, val := range jwt.StringValidators() {
			resp := &validator.StringResponse{}
			val.ValidateString(ctx, validator.StringRequest{ConfigValue: v}, resp)
			if resp.Diagnostics.HasError() {
				t.Errorf("jwt_expires_in: validator rejected %s", v)
			}
		}
	}
}
