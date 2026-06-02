package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// minimalConfig wraps a provider block with a data source so that Terraform
// actually calls ConfigureProvider (it is skipped for provider-only configs).
func minimalConfig(providerBlock string) string {
	return providerBlock + `
data "openwebui_user" "x" {
  user_id = "test-id"
}
`
}

func TestProvider_Configure_MissingEndpoint(t *testing.T) {
	t.Setenv("OPENWEBUI_ENDPOINT", "")
	t.Setenv("OPENWEBUI_TOKEN", "test-token")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      minimalConfig(`provider "openwebui" { token = "test-token" }`),
				ExpectError: regexp.MustCompile(`Missing Open WebUI API endpoint`),
			},
		},
	})
}

func TestProvider_Configure_MissingToken(t *testing.T) {
	t.Setenv("OPENWEBUI_ENDPOINT", "http://fake.example.com")
	t.Setenv("OPENWEBUI_TOKEN", "")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      minimalConfig(`provider "openwebui" { endpoint = "http://fake.example.com" }`),
				ExpectError: regexp.MustCompile(`Missing Open WebUI API token`),
			},
		},
	})
}

func TestProvider_Configure_BothMissing(t *testing.T) {
	t.Setenv("OPENWEBUI_ENDPOINT", "")
	t.Setenv("OPENWEBUI_TOKEN", "")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      minimalConfig(`provider "openwebui" {}`),
				ExpectError: regexp.MustCompile(`Missing Open WebUI API endpoint`),
			},
		},
	})
}
