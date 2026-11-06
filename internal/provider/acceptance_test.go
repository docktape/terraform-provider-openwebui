package provider

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

func testAccProtoV6ProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"openwebui": providerserver.NewProtocol6WithError(New()),
	}
}

func testAccPreCheck(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC must be set to run acceptance tests")
	}

	if os.Getenv("OPENWEBUI_TOKEN") == "" {
		t.Skip("OPENWEBUI_TOKEN must be set to run acceptance tests")
	}

	if os.Getenv("OPENWEBUI_ENDPOINT") == "" {
		t.Skip("OPENWEBUI_ENDPOINT must be set to run acceptance tests")
	}
}

func testAccRequireEnv(t *testing.T, key string) string {
	value := os.Getenv(key)
	if value == "" {
		t.Skipf("%s must be set for this acceptance test", key)
	}

	return value
}

// testAccOptionalEnv returns the env var value or "" without skipping.
func testAccOptionalEnv(_ *testing.T, key string) string {
	return os.Getenv(key)
}

func testAccProviderConfig() string {
	return fmt.Sprintf(`
provider "openwebui" {
  endpoint = %q
}
`, os.Getenv("OPENWEBUI_ENDPOINT"))
}

var (
	testAccVersionOnce sync.Once
	testAccVersion     string
	testAccVersionErr  error
)

// testAccOpenWebUIVersion returns the version of the Open WebUI instance the
// acceptance tests run against (for example "0.12.0"). The lookup uses
// OPENWEBUI_ENDPOINT / OPENWEBUI_TOKEN and is cached for the process. The test
// fails when the version cannot be read.
func testAccOpenWebUIVersion(t *testing.T) string {
	t.Helper()
	testAccVersionOnce.Do(func() {
		c, err := client.NewClient(os.Getenv("OPENWEBUI_ENDPOINT"), os.Getenv("OPENWEBUI_TOKEN"), false)
		if err != nil {
			testAccVersionErr = err
			return
		}
		testAccVersion, testAccVersionErr = c.GetVersion(context.Background())
	})
	if testAccVersionErr != nil {
		t.Fatalf("read Open WebUI version: %v", testAccVersionErr)
	}
	return testAccVersion
}

// testAccSkipBelow skips the test when the instance is older than minVersion, for
// attributes that only exist in newer Open WebUI releases.
func testAccSkipBelow(t *testing.T, minVersion string) {
	t.Helper()
	if v := testAccOpenWebUIVersion(t); compareVersions(v, minVersion) < 0 {
		t.Skipf("requires Open WebUI %s or later (instance is %s)", minVersion, v)
	}
}

// testAccSkipAtOrAbove skips the test when the instance is v or newer, for
// tests that pin behaviour a later release changed.
func testAccSkipAtOrAbove(t *testing.T, v string) {
	t.Helper()
	if got := testAccOpenWebUIVersion(t); compareVersions(got, v) >= 0 {
		t.Skipf("only applies below Open WebUI %s (instance is %s)", v, got)
	}
}
