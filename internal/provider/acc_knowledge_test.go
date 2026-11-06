package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccKnowledgeResource(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-knowledge")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccKnowledgeResourceConfig(name, "Initial description"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_knowledge.test", "name", name),
					resource.TestCheckResourceAttr("openwebui_knowledge.test", "description", "Initial description"),
					resource.TestCheckResourceAttrSet("openwebui_knowledge.test", "id"),
					resource.TestCheckResourceAttrSet("openwebui_knowledge.test", "created_at"),
					resource.TestCheckNoResourceAttr("openwebui_knowledge.test", "data_json"),
					resource.TestCheckNoResourceAttr("openwebui_knowledge.test", "meta_json"),
				),
			},
			{
				ResourceName:      "openwebui_knowledge.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccKnowledgeResourceConfig(name, "Updated description"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_knowledge.test", "description", "Updated description"),
				),
			},
		},
	})
}

// TestAccKnowledgeResource_UpgradeFrom1x creates a knowledge base with provider
// 1.2.0, then plans with the current provider. The 1.2.0 state carries the
// removed data_json and meta_json attributes (null: the server never returns
// them), so the refresh must drop them from state without an error and without
// a diff.
//
// Provider 1.2.0 cannot be applied with data_json configured: the server
// ignores the field, so 1.2.0 fails with "Provider produced inconsistent
// result after apply". The step 1 config therefore leaves it unset. The
// non-null state case is covered by
// TestKnowledgeResource_UpgradeStateDropsRemovedAttributes.
func TestAccKnowledgeResource_UpgradeFrom1x(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-knowledge-upgrade")

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"openwebui": {
						Source:            "docktape/openwebui",
						VersionConstraint: "1.2.0",
					},
				},
				Config: fmt.Sprintf(`%s
resource "openwebui_knowledge" "test" {
  name        = %q
  description = "Upgrade from 1.x"
}
`, testAccProviderConfig(), name),
				Check: resource.TestCheckResourceAttr("openwebui_knowledge.test", "name", name),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
				Config: fmt.Sprintf(`%s
resource "openwebui_knowledge" "test" {
  name        = %q
  description = "Upgrade from 1.x"
}
`, testAccProviderConfig(), name),
				PlanOnly: true,
			},
		},
	})
}

func TestAccKnowledgeDataSource(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-knowledge-ds")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccKnowledgeDataSourceConfig(name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.openwebui_knowledge.test", "name", name),
					resource.TestCheckResourceAttrSet("data.openwebui_knowledge.test", "id"),
					resource.TestCheckResourceAttrSet("data.openwebui_knowledge.test", "knowledge_id"),
					resource.TestCheckNoResourceAttr("data.openwebui_knowledge.test", "data_json"),
				),
			},
		},
	})
}

func testAccKnowledgeResourceConfig(name, description string) string {
	return fmt.Sprintf(`%s
resource "openwebui_knowledge" "test" {
  name        = %q
  description = %q
}
`, testAccProviderConfig(), name, description)
}

func testAccKnowledgeDataSourceConfig(name string) string {
	return fmt.Sprintf(`%s
resource "openwebui_knowledge" "seed" {
  name        = %q
  description = "Data source lookup test"
}

data "openwebui_knowledge" "test" {
  name = openwebui_knowledge.seed.name
}
`, testAccProviderConfig(), name)
}
