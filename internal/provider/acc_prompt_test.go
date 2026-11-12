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

// TestAccPromptDataSource_Inactive reads a prompt that the resource has
// deactivated. GET /prompts/ hides inactive prompts, so the data source
// searches GET /prompts/list instead.
func TestAccPromptDataSource_Inactive(t *testing.T) {
	suffix := acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum)
	command := "tfaccdi" + suffix

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`%s
resource "openwebui_prompt" "seed" {
  command   = %q
  name      = "Acc Inactive DS Prompt"
  content   = "Inactive data source lookup test."
  is_active = false
}

data "openwebui_prompt" "test" {
  command = openwebui_prompt.seed.command
}
`, testAccProviderConfig(), command),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.openwebui_prompt.test", "id", "openwebui_prompt.seed", "id"),
					resource.TestCheckResourceAttr("data.openwebui_prompt.test", "name", "Acc Inactive DS Prompt"),
					resource.TestCheckResourceAttr("data.openwebui_prompt.test", "is_active", "false"),
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
			// Step 1: create with is_active=true and multiple tags. The
			// inactive path is covered by TestAccPromptResource_Inactive.
			{
				Config: fmt.Sprintf(`%s
resource "openwebui_prompt" "test" {
  command   = %q
  name      = "New Fields Test"
  content   = "Test content."
  is_active = true
  tags      = ["test", "acc"]
}
`, testAccProviderConfig(), command),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_prompt.test", "is_active", "true"),
					resource.TestCheckResourceAttr("openwebui_prompt.test", "tags.#", "2"),
					resource.TestCheckResourceAttr("openwebui_prompt.test", "tags.0", "test"),
				),
			},
			// Step 2: update tags only; is_active stays true.
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
					resource.TestCheckResourceAttr("openwebui_prompt.test", "tags.0", "updated"),
				),
			},
		},
	})
}

// TestAccPromptResource_Inactive covers is_active = false. Open WebUI's create
// and update forms have no is_active (create always yields true), so the
// provider flips it through POST /prompts/id/{id}/toggle.
func TestAccPromptResource_Inactive(t *testing.T) {
	suffix := acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum)
	command := "tfaccia" + suffix

	cfg := func(content string, active bool) string {
		return fmt.Sprintf(`%s
resource "openwebui_prompt" "test" {
  command   = %q
  name      = "Inactive Prompt"
  content   = %q
  is_active = %t
}
`, testAccProviderConfig(), command, content, active)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			// Create inactive: the server creates it active and the provider toggles it.
			{
				Config: cfg("v1", false),
				Check:  resource.TestCheckResourceAttr("openwebui_prompt.test", "is_active", "false"),
			},
			// A refresh and plan against the live (inactive) prompt shows no diff.
			{
				Config:   cfg("v1", false),
				PlanOnly: true,
			},
			// Import of an inactive prompt reads is_active from the server.
			{
				ResourceName:      "openwebui_prompt.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// An update that leaves is_active alone must not re-activate it.
			{
				Config: cfg("v2", false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_prompt.test", "content", "v2"),
					resource.TestCheckResourceAttr("openwebui_prompt.test", "is_active", "false"),
				),
			},
			// Activate.
			{
				Config: cfg("v2", true),
				Check:  resource.TestCheckResourceAttr("openwebui_prompt.test", "is_active", "true"),
			},
			// Deactivate again together with a content change.
			{
				Config: cfg("v3", false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_prompt.test", "content", "v3"),
					resource.TestCheckResourceAttr("openwebui_prompt.test", "is_active", "false"),
				),
			},
			{
				Config:   cfg("v3", false),
				PlanOnly: true,
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
