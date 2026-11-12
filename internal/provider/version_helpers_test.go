package provider

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"0.11.0", "0.12.0", -1},
		{"0.12.0", "0.11.0", 1},
		{"v0.12.0", "0.12.0", 0},
		{"0.11.10", "0.11.4", 1},
		{"0.12", "0.12.0", 0},
		{"0.12.0", "0.12", 0},
		{"1", "0.99.99", 1},
		{"0.11.4", "0.12.0", -1},
		{"", "0.0.0", 0},
	}
	for _, tc := range tests {
		got := compareVersions(tc.a, tc.b)
		sign := 0
		if got < 0 {
			sign = -1
		} else if got > 0 {
			sign = 1
		}
		if sign != tc.want {
			t.Errorf("compareVersions(%q, %q) = %d, want sign %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestRequiresNewerServer(t *testing.T) {
	d := requiresNewerServer(path.Root("foo").AtName("bar"), "v0.12.0")
	if d.Severity() != diag.SeverityError {
		t.Fatalf("severity = %v, want error", d.Severity())
	}
	if d.Summary() != "Requires a newer Open WebUI" {
		t.Errorf("summary = %q", d.Summary())
	}
	withPath, ok := d.(diag.DiagnosticWithPath)
	if !ok {
		t.Fatalf("diagnostic %T does not carry an attribute path", d)
	}
	if !withPath.Path().Equal(path.Root("foo").AtName("bar")) {
		t.Errorf("path = %s", withPath.Path())
	}
	if !strings.Contains(d.Detail(), "Open WebUI v0.12.0 or later") {
		t.Errorf("detail = %q", d.Detail())
	}
	if !strings.Contains(d.Detail(), "`foo.bar`") {
		t.Errorf("detail should name the attribute: %q", d.Detail())
	}
}

// TestAccServerVersion checks the live instance against the supported range
// and exercises the version-gating helpers the other acceptance tests use.
func TestAccServerVersion(t *testing.T) {
	testAccPreCheck(t)
	v := testAccOpenWebUIVersion(t)
	if compareVersions(v, "0.11.0") < 0 {
		t.Fatalf("Open WebUI %s is below the supported floor v0.11.0", v)
	}

	t.Run("skip below v0.12.0", func(t *testing.T) {
		testAccSkipBelow(t, "0.12.0")
		if compareVersions(v, "0.12.0") < 0 {
			t.Errorf("ran on %s although it is below v0.12.0", v)
		}
	})
	t.Run("skip at or above v0.12.0", func(t *testing.T) {
		testAccSkipAtOrAbove(t, "0.12.0")
		if compareVersions(v, "0.12.0") >= 0 {
			t.Errorf("ran on %s although it is at or above v0.12.0", v)
		}
	})
}

// TestAccVersionHelpersSkipOutsideAcceptanceMode guards the helpers that
// acceptance tests call at the top: without TF_ACC they must skip, never fail
// on a failed version lookup.
func TestAccVersionHelpersSkipOutsideAcceptanceMode(t *testing.T) {
	t.Setenv("TF_ACC", "")

	helpers := map[string]func(*testing.T){
		"testAccOpenWebUIVersion": func(t *testing.T) { testAccOpenWebUIVersion(t) },
		"testAccSkipBelow":        func(t *testing.T) { testAccSkipBelow(t, "0.12.0") },
		"testAccSkipAtOrAbove":    func(t *testing.T) { testAccSkipAtOrAbove(t, "0.11.1") },
	}
	for name, helper := range helpers {
		t.Run(name, func(t *testing.T) {
			// t.Skip ends the goroutine, so record the outcome in a defer.
			var skipped, failed bool
			ok := t.Run("inner", func(inner *testing.T) {
				defer func() {
					skipped = inner.Skipped()
					failed = inner.Failed()
				}()
				helper(inner)
			})
			if !skipped || failed || !ok {
				t.Errorf("helper should skip without TF_ACC (skipped=%v failed=%v ok=%v)", skipped, failed, ok)
			}
		})
	}
}
