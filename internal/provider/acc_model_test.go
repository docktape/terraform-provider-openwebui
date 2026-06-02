package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccModelResource(t *testing.T) {
	modelID := acctest.RandomWithPrefix("tf-acc-model")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccModelResourceConfig(modelID, "Initial Name", false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_model.test", "model_id", modelID),
					resource.TestCheckResourceAttr("openwebui_model.test", "name", "Initial Name"),
					resource.TestCheckResourceAttr("openwebui_model.test", "is_active", "false"),
					resource.TestCheckResourceAttrSet("openwebui_model.test", "id"),
				),
			},
			{
				ResourceName:      "openwebui_model.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccModelResourceConfig(modelID, "Updated Name", true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_model.test", "name", "Updated Name"),
					resource.TestCheckResourceAttr("openwebui_model.test", "is_active", "true"),
				),
			},
		},
	})
}

func TestAccModelDataSource(t *testing.T) {
	modelID := acctest.RandomWithPrefix("tf-acc-model-ds")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccModelDataSourceConfig(modelID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.openwebui_model.test", "model_id", modelID),
					resource.TestCheckResourceAttrSet("data.openwebui_model.test", "id"),
				),
			},
		},
	})
}

// TestAccModelResourceNoCapabilities verifies that omitting the capabilities
// block entirely does not produce an "unknown value" conversion error.
func TestAccModelResourceNoCapabilities(t *testing.T) {
	modelID := acctest.RandomWithPrefix("tf-acc-model-nocaps")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccModelResourceConfigNoCaps(modelID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_model.test", "model_id", modelID),
					resource.TestCheckResourceAttrSet("openwebui_model.test", "id"),
				),
			},
		},
	})
}

func testAccModelResourceConfigNoCaps(modelID string) string {
	return fmt.Sprintf(`%s
resource "openwebui_model" "test" {
  model_id      = %q
  name          = "No Caps Model"
  base_model_id = "llama3.2"

  params = {}
}
`, testAccProviderConfig(), modelID)
}

func testAccModelResourceConfig(modelID, name string, active bool) string {
	return fmt.Sprintf(`%s
resource "openwebui_model" "test" {
  model_id      = %q
  name          = %q
  base_model_id = "llama3.2"
  is_active     = %t

  params = {}

  capabilities = {}
}
`, testAccProviderConfig(), modelID, name, active)
}

func testAccModelDataSourceConfig(modelID string) string {
	return fmt.Sprintf(`%s
resource "openwebui_model" "seed" {
  model_id      = %q
  name          = "Acc DS Model"
  base_model_id = "llama3.2"

  params = {}

  capabilities = {}
}

data "openwebui_model" "test" {
  model_id = openwebui_model.seed.model_id
}
`, testAccProviderConfig(), modelID)
}
