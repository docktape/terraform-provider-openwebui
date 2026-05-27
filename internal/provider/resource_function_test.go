package provider

import "testing"

func TestFunctionTogglesNeeded(t *testing.T) {
	cases := []struct {
		name                                              string
		curActive, curGlobal, wantActive, wantGlobal      bool
		expectActive, expectGlobal                        bool
	}{
		{"no change", false, false, false, false, false, false},
		{"activate", false, false, true, false, true, false},
		{"globalize", false, false, false, true, false, true},
		{"both", false, false, true, true, true, true},
		{"deactivate", true, true, false, true, true, false},
		{"already on", true, true, true, true, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotActive, gotGlobal := functionTogglesNeeded(tc.curActive, tc.curGlobal, tc.wantActive, tc.wantGlobal)
			if gotActive != tc.expectActive || gotGlobal != tc.expectGlobal {
				t.Fatalf("got (%v,%v) want (%v,%v)", gotActive, gotGlobal, tc.expectActive, tc.expectGlobal)
			}
		})
	}
}
