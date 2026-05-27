package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccFunctionResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccFunctionResourceConfig("Acc Filter", true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_function.test", "function_id", "tf_acc_filter"),
					resource.TestCheckResourceAttr("openwebui_function.test", "name", "Acc Filter"),
					resource.TestCheckResourceAttr("openwebui_function.test", "type", "filter"),
					resource.TestCheckResourceAttr("openwebui_function.test", "is_active", "true"),
					resource.TestCheckResourceAttr("openwebui_function.test", "is_global", "false"),
				),
			},
			{
				ResourceName:            "openwebui_function.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"content"},
			},
			{
				Config: testAccFunctionResourceConfig("Acc Filter Renamed", false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_function.test", "name", "Acc Filter Renamed"),
					resource.TestCheckResourceAttr("openwebui_function.test", "is_active", "false"),
				),
			},
		},
	})
}

func testAccFunctionResourceConfig(name string, active bool) string {
	content := `"""
title: Acc Filter
author: docktape
version: 0.1.0
"""

from pydantic import BaseModel


class Filter:
    class Valves(BaseModel):
        priority: int = 0

    def __init__(self):
        self.valves = self.Valves()

    def inlet(self, body: dict) -> dict:
        return body
`
	return fmt.Sprintf(`%s
resource "openwebui_function" "test" {
  function_id = "tf_acc_filter"
  name        = %q
  is_active   = %t
  content     = <<-PY
%s
  PY
}
`, testAccProviderConfig(), name, active, content)
}
