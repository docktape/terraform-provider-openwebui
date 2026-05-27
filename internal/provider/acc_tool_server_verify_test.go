package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccToolServerVerifyDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccRequireEnv(t, "OPENWEBUI_TOOL_SERVER_URL")
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccToolServerVerifyDataSourceConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.openwebui_tool_server_verify.server", "verified", "true"),
				),
			},
		},
	})
}

func testAccToolServerVerifyDataSourceConfig() string {
	toolServerURL := os.Getenv("OPENWEBUI_TOOL_SERVER_URL")
	toolServerPath := os.Getenv("OPENWEBUI_TOOL_SERVER_PATH")
	if toolServerPath == "" {
		toolServerPath = "/"
	}

	authType := os.Getenv("OPENWEBUI_TOOL_SERVER_AUTH_TYPE")
	key := os.Getenv("OPENWEBUI_TOOL_SERVER_KEY")
	headersJSON := os.Getenv("OPENWEBUI_TOOL_SERVER_HEADERS_JSON")
	configJSON := os.Getenv("OPENWEBUI_TOOL_SERVER_CONFIG_JSON")

	authTypeLine := ""
	if authType != "" {
		authTypeLine = fmt.Sprintf("  auth_type = %q\n", authType)
	}

	keyLine := ""
	if key != "" {
		keyLine = fmt.Sprintf("  key = %q\n", key)
	}

	headersLine := ""
	if headersJSON != "" {
		headersLine = fmt.Sprintf("  headers_json = %q\n", headersJSON)
	}

	configLine := ""
	if configJSON != "" {
		configLine = fmt.Sprintf("  config_json = %q\n", configJSON)
	}

	return fmt.Sprintf(`%s
data "openwebui_tool_server_verify" "server" {
  url  = %q
  path = %q
%s%s%s%s}
`, testAccProviderConfig(), toolServerURL, toolServerPath, authTypeLine, keyLine, headersLine, configLine)
}
