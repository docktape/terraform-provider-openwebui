package provider

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccKnowledgeFileResource(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(filePath, []byte("hello from terraform acc test"), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	name := acctest.RandomWithPrefix("tf-acc-kf")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccKnowledgeFileResourceConfig(name, filePath),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("openwebui_knowledge_file.test", "id"),
					resource.TestCheckResourceAttrSet("openwebui_knowledge_file.test", "knowledge_id"),
					resource.TestCheckResourceAttrSet("openwebui_knowledge_file.test", "file_id"),
				),
			},
			{
				ResourceName:            "openwebui_knowledge_file.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"delete_file"},
			},
		},
	})
}

func testAccKnowledgeFileResourceConfig(name, filePath string) string {
	return fmt.Sprintf(`%s
resource "openwebui_knowledge" "kb" {
  name        = %q
  description = "Knowledge file acc test"
}

resource "openwebui_file" "doc" {
  source_path = %q
  process     = true
}

resource "openwebui_knowledge_file" "test" {
  knowledge_id = openwebui_knowledge.kb.id
  file_id      = openwebui_file.doc.id
  delete_file  = true
}
`, testAccProviderConfig(), name, filePath)
}
