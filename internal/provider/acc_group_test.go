package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGroupResource(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-group")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGroupResourceConfig(name, "Initial description"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_group.test", "name", name),
					resource.TestCheckResourceAttr("openwebui_group.test", "description", "Initial description"),
					resource.TestCheckResourceAttrSet("openwebui_group.test", "id"),
				),
			},
			{
				ResourceName:      "openwebui_group.test",
				ImportState:       true,
				ImportStateVerify: true,
				// users is optional in config but always populated as empty list on Read;
				// permissions sub-maps are computed and may have server defaults not in config.
				ImportStateVerifyIgnore: []string{"users", "permissions"},
			},
			{
				Config: testAccGroupResourceConfig(name, "Updated description"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_group.test", "name", name),
					resource.TestCheckResourceAttr("openwebui_group.test", "description", "Updated description"),
				),
			},
		},
	})
}

func TestAccGroupDataSource(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-group-ds")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGroupDataSourceConfig(name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.openwebui_group.test", "name", name),
					resource.TestCheckResourceAttrSet("data.openwebui_group.test", "id"),
					resource.TestCheckResourceAttrSet("data.openwebui_group.test", "group_id"),
				),
			},
		},
	})
}

func testAccGroupResourceConfig(name, description string) string {
	return fmt.Sprintf(`%s
resource "openwebui_group" "test" {
  name        = %q
  description = %q
}
`, testAccProviderConfig(), name, description)
}

func testAccGroupDataSourceConfig(name string) string {
	return fmt.Sprintf(`%s
resource "openwebui_group" "seed" {
  name        = %q
  description = "Data source lookup test"
}

data "openwebui_group" "test" {
  name = openwebui_group.seed.name
}
`, testAccProviderConfig(), name)
}
