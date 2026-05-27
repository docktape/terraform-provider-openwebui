package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccConfigExportDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccConfigExportDataSourceConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.openwebui_config_export.current", "config_json"),
				),
			},
		},
	})
}

func TestAccConfigImportResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccRequireEnv(t, "OPENWEBUI_TEST_CONFIG_IMPORT")
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccConfigImportResourceConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_config_import.restore", "id", "config_import"),
					resource.TestCheckResourceAttrSet("openwebui_config_import.restore", "config_json"),
				),
			},
		},
	})
}

func testAccConfigExportDataSourceConfig() string {
	return testAccProviderConfig() + `
data "openwebui_config_export" "current" {}
`
}

func testAccConfigImportResourceConfig() string {
	return testAccProviderConfig() + `
data "openwebui_config_export" "current" {}

resource "openwebui_config_import" "restore" {
  config_json = data.openwebui_config_export.current.config_json
}
`
}
