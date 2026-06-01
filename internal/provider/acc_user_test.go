package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccUserDataSource(t *testing.T) {
	email := testAccRequireEnv(t, "OPENWEBUI_TEST_USER_EMAIL")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccUserDataSourceConfig(email),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.openwebui_user.test", "email", email),
					resource.TestCheckResourceAttrSet("data.openwebui_user.test", "id"),
					resource.TestCheckResourceAttrSet("data.openwebui_user.test", "role"),
				),
			},
		},
	})
}

func testAccUserDataSourceConfig(email string) string {
	return fmt.Sprintf(`%s
data "openwebui_user" "test" {
  query = %q
}
`, testAccProviderConfig(), email)
}
