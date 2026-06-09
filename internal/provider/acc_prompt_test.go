package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPromptResource(t *testing.T) {
	suffix := acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum)
	command := "tfacc" + suffix

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccPromptResourceConfig(command, "Initial Prompt", "Say hello."),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_prompt.test", "name", "Initial Prompt"),
					resource.TestCheckResourceAttrSet("openwebui_prompt.test", "id"),
					resource.TestCheckResourceAttrSet("openwebui_prompt.test", "created_at"),
				),
			},
			{
				ResourceName:      "openwebui_prompt.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccPromptResourceConfig(command, "Updated Prompt", "Say goodbye."),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_prompt.test", "name", "Updated Prompt"),
					resource.TestCheckResourceAttr("openwebui_prompt.test", "content", "Say goodbye."),
				),
			},
		},
	})
}

func TestAccPromptDataSource(t *testing.T) {
	suffix := acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum)
	command := "tfaccds" + suffix

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccPromptDataSourceConfig(command),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.openwebui_prompt.test", "id"),
					resource.TestCheckResourceAttr("data.openwebui_prompt.test", "name", "Acc DS Prompt"),
				),
			},
		},
	})
}

func TestAccPromptResource_WithNewFields(t *testing.T) {
	suffix := acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum)
	command := "tfaccnf" + suffix

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`%s
resource "openwebui_prompt" "test" {
  command   = %q
  name      = "New Fields Test"
  content   = "Test content."
  is_active = false
  tags      = ["test", "acc"]
}
`, testAccProviderConfig(), command),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_prompt.test", "is_active", "false"),
					resource.TestCheckResourceAttr("openwebui_prompt.test", "tags.#", "2"),
					resource.TestCheckResourceAttr("openwebui_prompt.test", "tags.0", "test"),
				),
			},
			{
				Config: fmt.Sprintf(`%s
resource "openwebui_prompt" "test" {
  command   = %q
  name      = "New Fields Test"
  content   = "Test content."
  is_active = true
  tags      = ["updated"]
}
`, testAccProviderConfig(), command),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_prompt.test", "is_active", "true"),
					resource.TestCheckResourceAttr("openwebui_prompt.test", "tags.#", "1"),
				),
			},
		},
	})
}

func testAccPromptResourceConfig(command, name, content string) string {
	return fmt.Sprintf(`%s
resource "openwebui_prompt" "test" {
  command = %q
  name    = %q
  content = %q
}
`, testAccProviderConfig(), command, name, content)
}

func testAccPromptDataSourceConfig(command string) string {
	return fmt.Sprintf(`%s
resource "openwebui_prompt" "seed" {
  command = %q
  name    = "Acc DS Prompt"
  content = "Data source lookup test."
}

data "openwebui_prompt" "test" {
  command = openwebui_prompt.seed.command
}
`, testAccProviderConfig(), command)
}
