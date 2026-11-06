package provider

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
)

// requiresNewerServer reports a configured attribute that the connected Open
// WebUI server did not store because it predates minVersion. Attributes added
// after v0.11.0 use it instead of surfacing Terraform's generic "inconsistent
// result after apply" error.
func requiresNewerServer(attr path.Path, minVersion string) diag.Diagnostic {
	return diag.NewAttributeErrorDiagnostic(
		attr,
		"Requires a newer Open WebUI",
		fmt.Sprintf("`%s` requires Open WebUI %s or later. The server did not store it: remove the attribute or upgrade Open WebUI.", attr, minVersion),
	)
}

// compareVersions compares two numeric x.y.z version strings. It returns a
// negative number when a < b, zero when they are equal and a positive number
// when a > b. A leading "v" is ignored, missing parts count as 0, and any
// pre-release or build suffix on a part ("0.12.0rc1", "0.12.0-beta") is
// ignored.
func compareVersions(a, b string) int {
	pa, pb := versionParts(a), versionParts(b)
	for i := 0; i < len(pa) || i < len(pb); i++ {
		var x, y int
		if i < len(pa) {
			x = pa[i]
		}
		if i < len(pb) {
			y = pb[i]
		}
		if x != y {
			if x < y {
				return -1
			}
			return 1
		}
	}
	return 0
}

func versionParts(v string) []int {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	fields := strings.Split(v, ".")
	parts := make([]int, 0, len(fields))
	for _, f := range fields {
		end := 0
		for end < len(f) && f[end] >= '0' && f[end] <= '9' {
			end++
		}
		n, _ := strconv.Atoi(f[:end])
		parts = append(parts, n)
	}
	return parts
}
