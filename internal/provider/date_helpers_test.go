package provider

import (
	"testing"
)

func TestFormatDateValue_Zero(t *testing.T) {
	result := formatDateValue(0)
	if !result.IsNull() {
		t.Fatalf("expected null for zero timestamp, got %s", result.ValueString())
	}
}

func TestFormatDateValue_Negative(t *testing.T) {
	result := formatDateValue(-1)
	if !result.IsNull() {
		t.Fatalf("expected null for negative timestamp, got %s", result.ValueString())
	}
}

func TestFormatDateValue_KnownDate(t *testing.T) {
	// 2021-01-01 00:00:00 UTC
	result := formatDateValue(1609459200)
	if result.IsNull() {
		t.Fatal("expected non-null for valid timestamp")
	}
	if result.ValueString() != "2021-01-01" {
		t.Fatalf("expected 2021-01-01, got %s", result.ValueString())
	}
}

func TestFormatDateValue_AnotherKnownDate(t *testing.T) {
	// 2023-06-15 00:00:00 UTC
	result := formatDateValue(1686787200)
	if result.IsNull() {
		t.Fatal("expected non-null for valid timestamp")
	}
	if result.ValueString() != "2023-06-15" {
		t.Fatalf("expected 2023-06-15, got %s", result.ValueString())
	}
}

func TestFormatDateValue_ReturnsUTC(t *testing.T) {
	// 2024-12-31 23:00:00 UTC — should format as 2024-12-31, not roll over to 2025-01-01
	result := formatDateValue(1735685200)
	if result.IsNull() {
		t.Fatal("expected non-null")
	}
	got := result.ValueString()
	if len(got) != 10 {
		t.Fatalf("expected YYYY-MM-DD format, got %q", got)
	}
	if got[:4] != "2024" {
		t.Fatalf("expected year 2024 for UTC timestamp, got %s", got)
	}
}
