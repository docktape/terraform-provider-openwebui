package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccOAuthClientResource(t *testing.T) {
	clientID := acctest.RandomWithPrefix("tf-acc-oauth")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccRequireEnv(t, "OPENWEBUI_OAUTH_CLIENT_URL")
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccOAuthClientResourceConfig(clientID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_oauth_client.test", "id", clientID),
					resource.TestCheckResourceAttr("openwebui_oauth_client.test", "client_id", clientID),
				),
			},
		},
	})
}

func testAccOAuthClientResourceConfig(clientID string) string {
	clientURL := os.Getenv("OPENWEBUI_OAUTH_CLIENT_URL")
	clientName := os.Getenv("OPENWEBUI_OAUTH_CLIENT_NAME")
	clientType := os.Getenv("OPENWEBUI_OAUTH_CLIENT_TYPE")

	clientNameLine := ""
	if clientName != "" {
		clientNameLine = fmt.Sprintf("  client_name = %q\n", clientName)
	}

	clientTypeLine := ""
	if clientType != "" {
		clientTypeLine = fmt.Sprintf("  type = %q\n", clientType)
	}

	return fmt.Sprintf(`%s
resource "openwebui_oauth_client" "test" {
  url       = %q
  client_id = %q
%s%s}
`, testAccProviderConfig(), clientURL, clientID, clientNameLine, clientTypeLine)
}
