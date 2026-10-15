package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// decodeOptionalJSON maps a Terraform string attribute containing JSON into a Go map.
func decodeOptionalJSON(value types.String, attribute path.Path, diags *diag.Diagnostics) map[string]any {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}

	raw := strings.TrimSpace(value.ValueString())
	if raw == "" {
		return nil
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		diags.AddAttributeError(
			attribute,
			"Invalid JSON value",
			fmt.Sprintf("Expected attribute %s to contain valid JSON object text: %v", attribute.String(), err),
		)
		return nil
	}

	return result
}

// encodeOptionalJSON converts a map into a Terraform string attribute containing canonical JSON.
func encodeOptionalJSON(data map[string]any) (types.String, error) {
	if data == nil {
		return types.StringNull(), nil
	}

	encoded, err := json.Marshal(data)
	if err != nil {
		return types.StringNull(), fmt.Errorf("marshal JSON: %w", err)
	}

	return types.StringValue(string(encoded)), nil
}

// encodeOptionalJSONValue converts an arbitrary value into a JSON string.
func encodeOptionalJSONValue(value any) (types.String, error) {
	if value == nil {
		return types.StringNull(), nil
	}

	encoded, err := json.Marshal(value)
	if err != nil {
		return types.StringNull(), fmt.Errorf("marshal JSON: %w", err)
	}

	return types.StringValue(string(encoded)), nil
}

// canonicalJSON re-encodes JSON text in a canonical form (compact, object keys
// sorted, numbers kept verbatim) so that two texts differing only in key order
// or whitespace compare equal.
func canonicalJSON(raw []byte) (string, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return "", err
	}
	if decoder.More() {
		return "", fmt.Errorf("unexpected data after the top-level JSON value")
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

// equivalentJSON reports whether two JSON texts are semantically equal. Texts
// that are not valid JSON are compared verbatim.
func equivalentJSON(a, b string) bool {
	if a == b {
		return true
	}
	ca, errA := canonicalJSON([]byte(a))
	cb, errB := canonicalJSON([]byte(b))
	return errA == nil && errB == nil && ca == cb
}

// preserveEquivalentJSON returns preferred (the plan or prior state value)
// when it is semantically equal to fresh (the value just read from the
// server), so a user's own key order and whitespace never show up as drift or
// as an inconsistent result after apply. Otherwise it returns fresh.
func preserveEquivalentJSON(preferred, fresh types.String) types.String {
	if preferred.IsNull() || preferred.IsUnknown() || fresh.IsNull() || fresh.IsUnknown() {
		return fresh
	}
	if equivalentJSON(preferred.ValueString(), fresh.ValueString()) {
		return preferred
	}
	return fresh
}

// jsonShapeValidator checks at plan time that a string attribute holds JSON
// text of the expected top-level shape: an object, or an array of objects.
type jsonShapeValidator struct {
	arrayOfObjects bool
}

func (v jsonShapeValidator) Description(_ context.Context) string {
	if v.arrayOfObjects {
		return "value must be a JSON array of objects"
	}
	return "value must be a JSON object"
}

func (v jsonShapeValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v jsonShapeValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	raw := []byte(req.ConfigValue.ValueString())
	var ok bool
	if v.arrayOfObjects {
		var list []map[string]any
		ok = json.Unmarshal(raw, &list) == nil && list != nil
		if ok {
			for _, item := range list {
				if item == nil {
					ok = false
				}
			}
		}
	} else {
		var object map[string]any
		ok = json.Unmarshal(raw, &object) == nil && object != nil
	}
	if !ok {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid JSON value", fmt.Sprintf("Attribute %s: %s, got %q.", req.Path.String(), v.Description(ctx), req.ConfigValue.ValueString()))
	}
}

// jsonObjectValidator requires a JSON object such as `{}` or `{"a":1}`.
func jsonObjectValidator() validator.String { return jsonShapeValidator{} }

// jsonArrayOfObjectsValidator requires a JSON array of objects such as `[]`.
func jsonArrayOfObjectsValidator() validator.String {
	return jsonShapeValidator{arrayOfObjects: true}
}
