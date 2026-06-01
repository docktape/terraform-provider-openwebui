package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func accToolContent() string {
	return `"""
title: Acc Tool
description: Terraform acceptance test tool
author: docktape
version: 0.1.0
"""


class Tools:
    def __init__(self):
        pass

    def hello(self) -> str:
        """Returns a greeting."""
        return "Hello from acc test!"
`
}

func accToolWithValvesContent() string {
	return `"""
title: Acc Valves Tool
description: Tool with configurable valves for acc test
author: docktape
version: 0.1.0
"""

from pydantic import BaseModel

class Tools:
    class Valves(BaseModel):
        max_pages: int = 5

    def __init__(self):
        self.valves = self.Valves()

    def get_tools(self) -> list[dict]:
        return []
`
}

func testAccToolResourceConfig(toolID, name string) string {
	return fmt.Sprintf(`%s
resource "openwebui_tool" "test" {
  tool_id     = %q
  name        = %q
  description = "Terraform acc test tool"
  content     = <<-PY
%s
  PY
}
`, testAccProviderConfig(), toolID, name, accToolContent())
}

func testAccToolDataSourceConfig(toolID, name string) string {
	return fmt.Sprintf(`%s
resource "openwebui_tool" "seed" {
  tool_id     = %q
  name        = %q
  description = "DS lookup test"
  content     = <<-PY
%s
  PY
}

data "openwebui_tool" "test" {
  tool_id = openwebui_tool.seed.tool_id
}
`, testAccProviderConfig(), toolID, name, accToolContent())
}

func testAccToolValvesResourceConfig(toolID string, maxPages int) string {
	return fmt.Sprintf(`%s
resource "openwebui_tool" "valves_tool" {
  tool_id     = %q
  name        = "Acc Valves Tool"
  description = "Tool with valves for acc test"
  content     = <<-PY
%s
  PY
}

resource "openwebui_tool_valves" "test" {
  tool_id = openwebui_tool.valves_tool.id

  valves_json = jsonencode({
    max_pages = %d
  })
}
`, testAccProviderConfig(), toolID, accToolWithValvesContent(), maxPages)
}

func TestAccToolResource(t *testing.T) {
	toolID := acctest.RandomWithPrefix("tfacctool")
	toolID = regexp.MustCompile(`-`).ReplaceAllString(toolID, "_")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccToolResourceConfig(toolID, "Initial Tool"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_tool.test", "tool_id", toolID),
					resource.TestCheckResourceAttr("openwebui_tool.test", "name", "Initial Tool"),
					resource.TestCheckResourceAttrSet("openwebui_tool.test", "id"),
				),
			},
			{
				ResourceName:      "openwebui_tool.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccToolResourceConfig(toolID, "Updated Tool"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_tool.test", "name", "Updated Tool"),
				),
			},
		},
	})
}

func TestAccToolDataSource(t *testing.T) {
	toolID := acctest.RandomWithPrefix("tfacctool")
	toolID = regexp.MustCompile(`-`).ReplaceAllString(toolID, "_")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccToolDataSourceConfig(toolID, "Acc DS Tool"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.openwebui_tool.test", "tool_id", toolID),
					resource.TestCheckResourceAttr("data.openwebui_tool.test", "name", "Acc DS Tool"),
					resource.TestCheckResourceAttrSet("data.openwebui_tool.test", "id"),
				),
			},
		},
	})
}

func TestAccToolValvesResource(t *testing.T) {
	toolID := acctest.RandomWithPrefix("tfaccvalve")
	toolID = regexp.MustCompile(`-`).ReplaceAllString(toolID, "_")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccToolValvesResourceConfig(toolID, 10),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_tool_valves.test", "tool_id", toolID),
					resource.TestCheckResourceAttrSet("openwebui_tool_valves.test", "spec_json"),
				),
			},
			{
				Config: testAccToolValvesResourceConfig(toolID, 20),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("openwebui_tool_valves.test", "valves_json", regexp.MustCompile(`"max_pages":\s*20`)),
				),
			},
			{
				ResourceName:      "openwebui_tool_valves.test",
				ImportState:       true,
				ImportStateId:     toolID,
				ImportStateVerify: true,
			},
		},
	})
}
