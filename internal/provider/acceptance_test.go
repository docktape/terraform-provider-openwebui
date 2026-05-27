package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
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
		t.Fatal("OPENWEBUI_TOKEN must be set for acceptance tests")
	}
}

func testAccRequireEnv(t *testing.T, key string) string {
	value := os.Getenv(key)
	if value == "" {
		t.Skipf("%s must be set for this acceptance test", key)
	}

	return value
}

func testAccProviderConfig() string {
	endpoint := os.Getenv("OPENWEBUI_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:3000/api/v1"
	}

	return fmt.Sprintf(`
provider "openwebui" {
  endpoint = %q
}
`, endpoint)
}
