package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"strconv"
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

// rawJSONToState renders a raw server JSON value as a canonical JSON string.
// null/absent becomes a null attribute. A JSON string (a legacy shape of the
// dict | str | None params fields) is shown as its content, canonicalised when
// that content is itself JSON.
func rawJSONToState(raw json.RawMessage) types.String {
	if len(raw) == 0 || string(raw) == "null" {
		return types.StringNull()
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		if canonical, err := canonicalJSON([]byte(text)); err == nil {
			return types.StringValue(canonical)
		}
		return types.StringValue(text)
	}
	if canonical, err := canonicalJSON(raw); err == nil {
		return types.StringValue(canonical)
	}
	return types.StringValue(string(raw))
}

// isJSONObject reports whether raw is JSON object text. null, absent and
// every other JSON shape report false.
func isJSONObject(raw json.RawMessage) bool {
	var object map[string]json.RawMessage
	return json.Unmarshal(raw, &object) == nil && object != nil
}

// decodeJSONWithNumbers decodes one JSON value, keeping numbers as their
// literal json.Number text.
func decodeJSONWithNumbers(text string) (any, error) {
	decoder := json.NewDecoder(strings.NewReader(text))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	if decoder.More() {
		return nil, fmt.Errorf("unexpected data after the top-level JSON value")
	}
	return value, nil
}

// equivalentJSON reports whether two JSON texts are semantically equal: key
// order and whitespace are ignored and numbers compare by value, so `1.50`
// equals `1.5` and `1e2` equals `100.0` (Open WebUI re-serialises numbers
// through Python floats). Texts that are not valid JSON are compared verbatim.
func equivalentJSON(a, b string) bool {
	if a == b {
		return true
	}
	va, errA := decodeJSONWithNumbers(a)
	vb, errB := decodeJSONWithNumbers(b)
	return errA == nil && errB == nil && jsonValuesEqual(va, vb)
}

// jsonValuesEqual compares two values produced by decodeJSONWithNumbers.
func jsonValuesEqual(a, b any) bool {
	switch av := a.(type) {
	case map[string]any:
		bv, ok := b.(map[string]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for k, x := range av {
			y, present := bv[k]
			if !present || !jsonValuesEqual(x, y) {
				return false
			}
		}
		return true
	case []any:
		bv, ok := b.([]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for i := range av {
			if !jsonValuesEqual(av[i], bv[i]) {
				return false
			}
		}
		return true
	case json.Number:
		bv, ok := b.(json.Number)
		return ok && jsonNumbersEqual(av, bv)
	default:
		// string, bool or nil.
		return a == b
	}
}

// maxComparableExponent bounds the decimal exponent jsonNumbersEqual expands
// exactly. Larger exponents (e.g. `1e999999999`) would make big.Rat allocate
// an enormous power of ten, so such numbers are compared by their text. This
// is deliberately conservative: equal values written differently with such an
// exponent (`1e1001` and `10e1000`) compare unequal. The plan-time JSON
// validators reject these literals (see pythonRoundTripNumberError), so they
// only reach this comparison from server-side values.
const maxComparableExponent = 1000

// jsonNumbersEqual compares two JSON number literals by exact value.
func jsonNumbersEqual(a, b json.Number) bool {
	if a == b {
		return true
	}
	ra, okA := jsonNumberRat(a)
	rb, okB := jsonNumberRat(b)
	if !okA || !okB {
		return false
	}
	return ra.Cmp(rb) == 0
}

// jsonNumberRat parses a JSON number literal into an exact rational. It
// reports false for an exponent beyond maxComparableExponent.
func jsonNumberRat(n json.Number) (*big.Rat, bool) {
	s := string(n)
	if i := strings.IndexAny(s, "eE"); i >= 0 {
		exp, err := strconv.ParseInt(s[i+1:], 10, 64)
		if err != nil || exp > maxComparableExponent || exp < -maxComparableExponent {
			return nil, false
		}
	}
	r, ok := new(big.Rat).SetString(s)
	return r, ok
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
// text of the expected top-level shape (an object, or an array of objects)
// whose number literals survive Open WebUI's Python round trip (see
// pythonRoundTripNumberError).
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
	value, err := decodeJSONWithNumbers(req.ConfigValue.ValueString())
	ok := err == nil
	if ok && v.arrayOfObjects {
		list, isList := value.([]any)
		ok = isList
		for _, item := range list {
			if _, isObject := item.(map[string]any); !isObject {
				ok = false
			}
		}
	} else if ok {
		_, ok = value.(map[string]any)
	}
	if !ok {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid JSON value", fmt.Sprintf("Attribute %s: %s, got %q.", req.Path.String(), v.Description(ctx), req.ConfigValue.ValueString()))
		return
	}
	if msg := firstUnroundTrippableNumber(value); msg != "" {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid JSON number", fmt.Sprintf("Attribute %s: %s", req.Path.String(), msg))
	}
}

// firstUnroundTrippableNumber walks a value produced by decodeJSONWithNumbers
// and returns the error message for the first number literal that
// pythonRoundTripNumberError rejects, or "".
func firstUnroundTrippableNumber(value any) string {
	switch v := value.(type) {
	case map[string]any:
		for _, item := range v {
			if msg := firstUnroundTrippableNumber(item); msg != "" {
				return msg
			}
		}
	case []any:
		for _, item := range v {
			if msg := firstUnroundTrippableNumber(item); msg != "" {
				return msg
			}
		}
	case json.Number:
		return pythonRoundTripNumberError(v)
	}
	return ""
}

// pythonMaxIntDigits is CPython's default limit (sys.int_info.default_max_str_digits,
// Python 3.11+) on the number of digits json.loads accepts in an integer.
const pythonMaxIntDigits = 4300

// pythonRoundTripNumberError reports why a JSON number literal would not come
// back from Open WebUI with the same value, or "" when it would. Open WebUI
// parses JSON with Python's json module: an integer literal (no fraction or
// exponent) becomes an exact Python int, while any other literal becomes a
// float64 and is written back as its shortest repr. So:
//   - integer literals are accepted at any size up to pythonMaxIntDigits;
//   - a fraction or exponent literal is accepted only when its exact value
//     equals the float64 Python would store (1.5, 0.25, 1e2 and 0.1 pass,
//     because repr(0.1) is "0.1"; 9007199254740993.0 does not, because the
//     float stores 9007199254740992);
//   - a literal that overflows to infinity (1e400) is rejected, because
//     Python would write it back as the invalid token Infinity;
//   - a fraction or exponent literal whose exponent is beyond
//     maxComparableExponent is rejected, because equivalentJSON cannot compare
//     it with the server's value.
func pythonRoundTripNumberError(n json.Number) string {
	s := string(n)
	if !strings.ContainsAny(s, ".eE") {
		if digits := strings.TrimPrefix(s, "-"); len(digits) > pythonMaxIntDigits {
			return fmt.Sprintf("integer literal with %d digits exceeds the %d-digit limit of Open WebUI's JSON parser.", len(digits), pythonMaxIntDigits)
		}
		return ""
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil && math.IsInf(f, 0) {
		return fmt.Sprintf("number %s is outside the range of a 64-bit float; Open WebUI would store it as Infinity, which is not valid JSON.", s)
	}
	exact, ok := jsonNumberRat(n)
	if !ok {
		return fmt.Sprintf("number %s has an exponent beyond ±%d, which this provider cannot compare with the value Open WebUI stores; write it without the exponent or as a plain integer.", s, maxComparableExponent)
	}
	stored := strconv.FormatFloat(f, 'g', -1, 64)
	if storedRat, ok := new(big.Rat).SetString(stored); !ok || storedRat.Cmp(exact) != 0 {
		return fmt.Sprintf("number %s cannot be stored exactly: Open WebUI parses it as a 64-bit float and would store %s. Write %s instead, or use an integer literal without a fraction or exponent for exact large values.", s, stored, stored)
	}
	return ""
}

// jsonObjectValidator requires a JSON object such as `{}` or `{"a":1}`.
func jsonObjectValidator() validator.String { return jsonShapeValidator{} }

// jsonArrayOfObjectsValidator requires a JSON array of objects such as `[]`.
func jsonArrayOfObjectsValidator() validator.String {
	return jsonShapeValidator{arrayOfObjects: true}
}

// jsonNumberNote is appended to the description of every attribute validated
// by jsonObjectValidator or jsonArrayOfObjectsValidator.
const jsonNumberNote = " Number literals must survive Open WebUI's Python JSON round trip: integers are exact (up to Python's 4300-digit limit), but a fraction or exponent that a 64-bit float cannot hold exactly (for example `9007199254740993.0`) or that overflows it (`1e400`) is rejected at plan time."
