package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPipelineResource(t *testing.T) {
	pipelineURL := testAccRequireEnv(t, "OPENWEBUI_PIPELINE_URL")
	pipelineKey := testAccRequireEnv(t, "OPENWEBUI_PIPELINE_KEY")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccPipelineResourceConfig(pipelineURL, pipelineKey),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("openwebui_pipeline.test", "id"),
					resource.TestCheckResourceAttrSet("openwebui_pipeline.test", "pipeline_id"),
					resource.TestCheckResourceAttr("openwebui_pipeline.test", "url", pipelineURL),
				),
			},
			{
				ResourceName:            "openwebui_pipeline.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"url", "source_path", "key"},
			},
		},
	})
}

func TestAccPipelineDataSource(t *testing.T) {
	pipelineURL := testAccRequireEnv(t, "OPENWEBUI_PIPELINE_URL")
	pipelineKey := testAccRequireEnv(t, "OPENWEBUI_PIPELINE_KEY")
	pipelineID := testAccRequireEnv(t, "OPENWEBUI_PIPELINE_ID")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccPipelineDataSourceConfig(pipelineURL, pipelineKey, pipelineID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.openwebui_pipeline.test", "id"),
					resource.TestCheckResourceAttr("data.openwebui_pipeline.test", "pipeline_id", pipelineID),
				),
			},
		},
	})
}

func TestAccPipelineValvesResource(t *testing.T) {
	pipelineURL := testAccRequireEnv(t, "OPENWEBUI_PIPELINE_URL")
	pipelineKey := testAccRequireEnv(t, "OPENWEBUI_PIPELINE_KEY")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccPipelineValvesResourceConfig(pipelineURL, pipelineKey),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("openwebui_pipeline_valves.test", "id"),
					resource.TestCheckResourceAttrSet("openwebui_pipeline_valves.test", "pipeline_id"),
				),
			},
			{
				Config:   testAccPipelineValvesResourceConfig(pipelineURL, pipelineKey),
				PlanOnly: true,
			},
		},
	})
}

func testAccPipelineResourceConfig(pipelineURL, pipelineKey string) string {
	return fmt.Sprintf(`%s
resource "openwebui_pipeline" "test" {
  url = %q
  key = %q
}
`, testAccProviderConfig(), pipelineURL, pipelineKey)
}

func testAccPipelineDataSourceConfig(pipelineURL, pipelineKey, pipelineID string) string {
	return fmt.Sprintf(`%s
resource "openwebui_pipeline" "seed" {
  url = %q
  key = %q
}

data "openwebui_pipeline" "test" {
  pipeline_id = %q
  depends_on  = [openwebui_pipeline.seed]
}
`, testAccProviderConfig(), pipelineURL, pipelineKey, pipelineID)
}

func testAccPipelineValvesResourceConfig(pipelineURL, pipelineKey string) string {
	return fmt.Sprintf(`%s
resource "openwebui_pipeline" "valves_pipeline" {
  url = %q
  key = %q
}

resource "openwebui_pipeline_valves" "test" {
  pipeline_id = openwebui_pipeline.valves_pipeline.pipeline_id
  url_idx     = openwebui_pipeline.valves_pipeline.url_idx
  # valves_json omitted: server defaults flow in and are preserved on subsequent plans
}
`, testAccProviderConfig(), pipelineURL, pipelineKey)
}
