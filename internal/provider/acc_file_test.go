package provider

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccFileResource(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(filePath, []byte("hello from terraform acc test"), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccFileResourceConfig(filePath),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("openwebui_file.test", "id"),
					resource.TestCheckResourceAttr("openwebui_file.test", "source_path", filePath),
					resource.TestCheckResourceAttrSet("openwebui_file.test", "filename"),
					resource.TestCheckResourceAttrSet("openwebui_file.test", "user_id"),
				),
			},
			{
				ResourceName:      "openwebui_file.test",
				ImportState:       true,
				ImportStateVerify: true,
				// source_path and metadata_json are local-only (not stored by the API).
				// process and process_in_background are plan-time values not returned by the API.
				// data_json, meta_json, hash, and updated_at may change asynchronously as
				// file processing completes, so they can differ between the original Create
				// read and the post-import Read.
				ImportStateVerifyIgnore: []string{
					"source_path",
					"metadata_json",
					"process",
					"process_in_background",
					"data_json",
					"meta_json",
					"hash",
					"updated_at",
				},
			},
		},
	})
}

func TestAccFileDataSource(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "ds_test.txt")
	if err := os.WriteFile(filePath, []byte("data source test content"), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccFileDataSourceConfig(filePath),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.openwebui_file.test", "id"),
					resource.TestCheckResourceAttrSet("data.openwebui_file.test", "filename"),
				),
			},
		},
	})
}

func TestAccFilesDataSource(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "files_ds_test.txt")
	if err := os.WriteFile(filePath, []byte("files list test"), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccFilesDataSourceConfig(filePath),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.openwebui_files.all", "files.0.id"),
				),
			},
		},
	})
}

func testAccFileResourceConfig(filePath string) string {
	return fmt.Sprintf(`%s
resource "openwebui_file" "test" {
  source_path = %q
}
`, testAccProviderConfig(), filePath)
}

func testAccFileDataSourceConfig(filePath string) string {
	return fmt.Sprintf(`%s
resource "openwebui_file" "seed" {
  source_path = %q
}

data "openwebui_file" "test" {
  file_id = openwebui_file.seed.id
}
`, testAccProviderConfig(), filePath)
}

func testAccFilesDataSourceConfig(filePath string) string {
	return fmt.Sprintf(`%s
resource "openwebui_file" "seed" {
  source_path = %q
}

data "openwebui_files" "all" {
  depends_on = [openwebui_file.seed]
}
`, testAccProviderConfig(), filePath)
}
