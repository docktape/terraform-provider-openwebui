package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// testAccAccessGroups exercises read_groups/write_groups on a resource that
// uses buildAccessControl: create with only write_groups, distinct read and
// write groups, read_groups left
// unset while write_groups is set (read_groups is extended at plan time),
// a write group kept from state that a configured read_groups no longer
// lists (plan-time error), and revoking via [].
func testAccAccessGroups(t *testing.T, resourceType, body string) {
	t.Helper()
	groupA := acctest.RandomWithPrefix("tf-acc-ag-a")
	groupB := acctest.RandomWithPrefix("tf-acc-ag-b")
	addr := resourceType + ".test"

	config := func(groups string) string {
		return fmt.Sprintf(`%s
resource "openwebui_group" "a" {
  name        = %q
  description = "access groups test A"
}

resource "openwebui_group" "b" {
  name        = %q
  description = "access groups test B"
}

resource %q "test" {
%s
%s
}
`, testAccProviderConfig(), groupA, groupB, resourceType, body, groups)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				// A configured write group missing from the configured read_groups is
				// rejected before anything is created.
				Config: config(`
  read_groups  = [openwebui_group.a.name]
  write_groups = [openwebui_group.b.name]
`),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`(?s)write_groups must also be listed in\s+read_groups`),
			},
			{
				// Create with only write_groups: read_groups is unknown at plan time
				// and comes back from Open WebUI as the write group.
				Config: config(`
  write_groups = [openwebui_group.b.name]
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "read_groups.#", "1"),
					resource.TestCheckResourceAttr(addr, "read_groups.0", groupB),
					resource.TestCheckResourceAttr(addr, "write_groups.#", "1"),
					resource.TestCheckResourceAttr(addr, "write_groups.0", groupB),
				),
			},
			{
				// Distinct read and write groups (read is a superset of write).
				Config: config(`
  read_groups  = [openwebui_group.a.name, openwebui_group.b.name]
  write_groups = [openwebui_group.b.name]
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "read_groups.#", "2"),
					resource.TestCheckResourceAttr(addr, "read_groups.0", groupA),
					resource.TestCheckResourceAttr(addr, "read_groups.1", groupB),
					resource.TestCheckResourceAttr(addr, "write_groups.#", "1"),
					resource.TestCheckResourceAttr(addr, "write_groups.0", groupB),
				),
			},
			{
				// Removing a group from read_groups together with revoking write via [].
				Config: config(`
  read_groups  = [openwebui_group.a.name]
  write_groups = []
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "read_groups.#", "1"),
					resource.TestCheckResourceAttr(addr, "read_groups.0", groupA),
					resource.TestCheckResourceAttr(addr, "write_groups.#", "0"),
				),
			},
			{
				// read_groups unset, write_groups set: the planned read_groups is the
				// current value extended with the new write group.
				Config: config(`
  write_groups = [openwebui_group.b.name]
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "read_groups.#", "2"),
					resource.TestCheckResourceAttr(addr, "read_groups.0", groupA),
					resource.TestCheckResourceAttr(addr, "read_groups.1", groupB),
					resource.TestCheckResourceAttr(addr, "write_groups.#", "1"),
					resource.TestCheckResourceAttr(addr, "write_groups.0", groupB),
				),
			},
			{
				// The same configuration plans no further changes.
				Config: config(`
  write_groups = [openwebui_group.b.name]
`),
				PlanOnly: true,
			},
			{
				// write_groups unset keeps b from state, but the configured read_groups
				// drops it: rejected at plan time instead of failing after apply.
				Config: config(`
  read_groups = [openwebui_group.a.name]
`),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`(?s)kept\s+from\s+the\s+current\s+state`),
			},
			{
				// Following the error's advice: revoke write explicitly.
				Config: config(`
  read_groups  = [openwebui_group.a.name]
  write_groups = []
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "read_groups.#", "1"),
					resource.TestCheckResourceAttr(addr, "read_groups.0", groupA),
					resource.TestCheckResourceAttr(addr, "write_groups.#", "0"),
				),
			},
			{
				// Revoke everything.
				Config: config(`
  read_groups  = []
  write_groups = []
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "read_groups.#", "0"),
					resource.TestCheckResourceAttr(addr, "write_groups.#", "0"),
				),
			},
		},
	})
}

func TestAccToolResourceAccessGroups(t *testing.T) {
	toolID := regexp.MustCompile(`-`).ReplaceAllString(acctest.RandomWithPrefix("tfacctoolag"), "_")
	testAccAccessGroups(t, "openwebui_tool", fmt.Sprintf(`  tool_id     = %q
  name        = "Acc Access Groups Tool"
  description = "Terraform acc access groups tool"
  content     = <<-PY
%s
  PY`, toolID, accToolContent()))
}

func TestAccSkillResourceAccessGroupsPlan(t *testing.T) {
	skillID := acctest.RandomWithPrefix("tfaccskillag")
	testAccAccessGroups(t, "openwebui_skill", fmt.Sprintf(`  skill_id = %q
  name     = %q
  content  = "grouped skill"`, skillID, skillID))
}

func TestAccPromptResourceAccessGroups(t *testing.T) {
	command := acctest.RandomWithPrefix("tfaccpromptag")
	testAccAccessGroups(t, "openwebui_prompt", fmt.Sprintf(`  command = %q
  name    = "Acc Access Groups Prompt"
  content = "Access groups prompt."`, command))
}

func TestAccModelResourceAccessGroups(t *testing.T) {
	modelID := acctest.RandomWithPrefix("tf-acc-model-ag")
	testAccAccessGroups(t, "openwebui_model", fmt.Sprintf(`  model_id      = %q
  name          = "Acc Access Groups Model"
  base_model_id = "llama3.2"
  params        = {}`, modelID))
}

func TestAccKnowledgeResourceAccessGroups(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-knowledge-ag")
	testAccAccessGroups(t, "openwebui_knowledge", fmt.Sprintf(`  name        = %q
  description = "Access groups knowledge"`, name))
}
