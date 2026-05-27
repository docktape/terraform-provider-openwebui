package provider

import (
	"fmt"
	"regexp"
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

// TestAccFunctionResourceDefaults verifies that omitting is_active and is_global
// leaves the function at Open WebUI's defaults (both false). This exercises the
// convergeToggles branch where the planned value is null and the current value
// is kept (no toggle call is made).
func TestAccFunctionResourceDefaults(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccFunctionDefaultsConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_function.defaults", "type", "filter"),
					resource.TestCheckResourceAttr("openwebui_function.defaults", "is_active", "false"),
					resource.TestCheckResourceAttr("openwebui_function.defaults", "is_global", "false"),
				),
			},
		},
	})
}

// TestAccFunctionResourceGlobal exercises the is_global toggle path: create with
// is_global = true (verifying ToggleFunctionGlobal is called), then flip it back
// to false.
func TestAccFunctionResourceGlobal(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccFunctionGlobalConfig(true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_function.global", "is_active", "true"),
					resource.TestCheckResourceAttr("openwebui_function.global", "is_global", "true"),
				),
			},
			{
				Config: testAccFunctionGlobalConfig(false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_function.global", "is_global", "false"),
				),
			},
		},
	})
}

// TestAccFunctionResourceContentChange verifies that changing the source content
// recomputes the server-derived type (and manifest) without producing an
// "inconsistent result after apply" error — these attributes are
// known-after-apply rather than pinned to prior state.
func TestAccFunctionResourceContentChange(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccFunctionContentConfig(accFunctionFilterContent()),
				Check:  resource.TestCheckResourceAttr("openwebui_function.content", "type", "filter"),
			},
			{
				Config: testAccFunctionContentConfig(accFunctionPipeContent()),
				Check:  resource.TestCheckResourceAttr("openwebui_function.content", "type", "pipe"),
			},
		},
	})
}

// TestAccFunctionValvesResource verifies the openwebui_function_valves resource:
// applying valve values, updating them, and importing.
func TestAccFunctionValvesResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccFunctionValvesConfig(7),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_function_valves.test", "function_id", "tf_acc_valves"),
					resource.TestCheckResourceAttrSet("openwebui_function_valves.test", "spec_json"),
					resource.TestMatchResourceAttr("openwebui_function_valves.test", "valves_json", regexp.MustCompile(`"priority":\s*7`)),
				),
			},
			{
				Config: testAccFunctionValvesConfig(3),
				Check:  resource.TestMatchResourceAttr("openwebui_function_valves.test", "valves_json", regexp.MustCompile(`"priority":\s*3`)),
			},
			{
				ResourceName:      "openwebui_function_valves.test",
				ImportState:       true,
				ImportStateId:     "tf_acc_valves",
				ImportStateVerify: true,
			},
		},
	})
}

// accFunctionFilterContent returns a minimal valid Filter function source.
func accFunctionFilterContent() string {
	return `"""
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
}

// accFunctionPipeContent returns a minimal valid Pipe function source.
func accFunctionPipeContent() string {
	return `"""
title: Acc Pipe
author: docktape
version: 0.2.0
"""


class Pipe:
    def __init__(self):
        pass

    def pipe(self, body: dict):
        return "ok"
`
}

func testAccFunctionResourceConfig(name string, active bool) string {
	return fmt.Sprintf(`%s
resource "openwebui_function" "test" {
  function_id = "tf_acc_filter"
  name        = %q
  is_active   = %t
  content     = <<-PY
%s
  PY
}
`, testAccProviderConfig(), name, active, accFunctionFilterContent())
}

func testAccFunctionDefaultsConfig() string {
	return fmt.Sprintf(`%s
resource "openwebui_function" "defaults" {
  function_id = "tf_acc_defaults"
  name        = "Acc Defaults"
  content     = <<-PY
%s
  PY
}
`, testAccProviderConfig(), accFunctionFilterContent())
}

func testAccFunctionGlobalConfig(global bool) string {
	return fmt.Sprintf(`%s
resource "openwebui_function" "global" {
  function_id = "tf_acc_global"
  name        = "Acc Global"
  is_active   = true
  is_global   = %t
  content     = <<-PY
%s
  PY
}
`, testAccProviderConfig(), global, accFunctionFilterContent())
}

func testAccFunctionContentConfig(content string) string {
	return fmt.Sprintf(`%s
resource "openwebui_function" "content" {
  function_id = "tf_acc_content"
  name        = "Acc Content"
  content     = <<-PY
%s
  PY
}
`, testAccProviderConfig(), content)
}

func testAccFunctionValvesConfig(priority int) string {
	return fmt.Sprintf(`%s
resource "openwebui_function" "valves_fn" {
  function_id = "tf_acc_valves"
  name        = "Acc Valves"
  content     = <<-PY
%s
  PY
}

resource "openwebui_function_valves" "test" {
  function_id = openwebui_function.valves_fn.id

  valves_json = jsonencode({
    priority = %d
  })
}
`, testAccProviderConfig(), accFunctionFilterContent(), priority)
}
