package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestFloat64Ptr(t *testing.T) {
	if got := float64Ptr(types.Float64Null()); got != nil {
		t.Errorf("null must map to nil, got %v", *got)
	}
	if got := float64Ptr(types.Float64Unknown()); got != nil {
		t.Errorf("unknown must map to nil, got %v", *got)
	}
	if got := float64Ptr(types.Float64Value(0)); got == nil || *got != 0 {
		t.Errorf("0 must map to a pointer to 0, got %v", got)
	}
	if got := float64Ptr(types.Float64Value(0.25)); got == nil || *got != 0.25 {
		t.Errorf("0.25 must map to a pointer to 0.25, got %v", got)
	}
}

func TestFloat64ValueOrNull(t *testing.T) {
	if got := float64ValueOrNull(nil); !got.IsNull() {
		t.Errorf("nil must map to null, got %v", got)
	}
	zero := 0.0
	if got := float64ValueOrNull(&zero); got.IsNull() || got.ValueFloat64() != 0 {
		t.Errorf("0 must map to a known 0, got %v", got)
	}
	half := 0.5
	if got := float64ValueOrNull(&half); got.ValueFloat64() != 0.5 {
		t.Errorf("0.5 must map to 0.5, got %v", got)
	}
}
