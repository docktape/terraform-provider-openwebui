package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccSkillResourceConfig(skillID, name, extra string) string {
	return fmt.Sprintf(`%s
resource "openwebui_skill" "test" {
  skill_id = %q
  name     = %q
  content  = "Always greet the customer first."
%s
}
`, testAccProviderConfig(), skillID, name, extra)
}

func testAccSkillDataSourceConfig(skillID, name string) string {
	return fmt.Sprintf(`%s
resource "openwebui_skill" "seed" {
  skill_id    = %q
  name        = %q
  description = "DS lookup test"
  content     = "Skill body for the data source test."
  tags        = ["ds"]
}

data "openwebui_skill" "by_id" {
  skill_id = openwebui_skill.seed.skill_id
}

data "openwebui_skill" "by_name" {
  name = openwebui_skill.seed.name
}
`, testAccProviderConfig(), skillID, name)
}

func TestAccSkillResource(t *testing.T) {
	skillID := acctest.RandomWithPrefix("tfaccskill")
	name := acctest.RandomWithPrefix("Acc Skill")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				// Minimal config: description is null, tags default to [], is_active to true.
				Config: testAccSkillResourceConfig(skillID, name, ""),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_skill.test", "skill_id", skillID),
					resource.TestCheckResourceAttr("openwebui_skill.test", "id", skillID),
					resource.TestCheckResourceAttr("openwebui_skill.test", "name", name),
					resource.TestCheckResourceAttr("openwebui_skill.test", "content", "Always greet the customer first."),
					resource.TestCheckResourceAttr("openwebui_skill.test", "is_active", "true"),
					resource.TestCheckResourceAttr("openwebui_skill.test", "tags.#", "0"),
					resource.TestCheckNoResourceAttr("openwebui_skill.test", "description"),
					resource.TestCheckResourceAttrSet("openwebui_skill.test", "user_id"),
				),
			},
			{
				ResourceName:      "openwebui_skill.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// Populated tags, description, and is_active=false.
				Config: testAccSkillResourceConfig(skillID, name+" Updated", `
  description = "Terraform acc test skill"
  is_active   = false
  tags        = ["support", "playbook"]
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_skill.test", "name", name+" Updated"),
					resource.TestCheckResourceAttr("openwebui_skill.test", "description", "Terraform acc test skill"),
					resource.TestCheckResourceAttr("openwebui_skill.test", "is_active", "false"),
					resource.TestCheckResourceAttr("openwebui_skill.test", "tags.#", "2"),
					resource.TestCheckResourceAttr("openwebui_skill.test", "tags.0", "support"),
					resource.TestCheckResourceAttr("openwebui_skill.test", "tags.1", "playbook"),
				),
			},
			{
				ResourceName:      "openwebui_skill.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// Explicit empty tags and an empty-string description round-trip.
				Config: testAccSkillResourceConfig(skillID, name+" Updated", `
  description = ""
  tags        = []
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_skill.test", "description", ""),
					resource.TestCheckResourceAttr("openwebui_skill.test", "is_active", "true"),
					resource.TestCheckResourceAttr("openwebui_skill.test", "tags.#", "0"),
				),
			},
			{
				// Removing description and tags from the config resets them.
				Config: testAccSkillResourceConfig(skillID, name+" Updated", ""),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr("openwebui_skill.test", "description"),
					resource.TestCheckResourceAttr("openwebui_skill.test", "tags.#", "0"),
				),
			},
		},
	})
}

func TestAccSkillResourceRejectsUnnormalisedID(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      testAccSkillResourceConfig("Mixed Case", "Rejected Skill", ""),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`(?s)Invalid skill_id.*lowercases`),
			},
		},
	})
}

func TestAccSkillDataSource(t *testing.T) {
	skillID := acctest.RandomWithPrefix("tfaccskill")
	name := acctest.RandomWithPrefix("Acc DS Skill")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccSkillDataSourceConfig(skillID, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.openwebui_skill.by_id", "skill_id", skillID),
					resource.TestCheckResourceAttr("data.openwebui_skill.by_id", "name", name),
					resource.TestCheckResourceAttr("data.openwebui_skill.by_id", "content", "Skill body for the data source test."),
					resource.TestCheckResourceAttr("data.openwebui_skill.by_id", "tags.#", "1"),
					resource.TestCheckResourceAttr("data.openwebui_skill.by_id", "is_active", "true"),
					resource.TestCheckResourceAttr("data.openwebui_skill.by_name", "skill_id", skillID),
					resource.TestCheckResourceAttr("data.openwebui_skill.by_name", "id", skillID),
					resource.TestCheckResourceAttr("data.openwebui_skill.by_name", "description", "DS lookup test"),
				),
			},
		},
	})
}

func TestAccSkillResourceAccessGroups(t *testing.T) {
	skillID := acctest.RandomWithPrefix("tfaccskill")
	groupName := acctest.RandomWithPrefix("tf-acc-skill-group")

	config := func(groups string) string {
		return fmt.Sprintf(`%s
resource "openwebui_group" "g" {
  name        = %q
  description = "skill access test"
}

resource "openwebui_skill" "test" {
  skill_id = %q
  name     = %q
  content  = "grouped skill"
%s
}
`, testAccProviderConfig(), groupName, skillID, skillID, groups)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config(`
  read_groups  = [openwebui_group.g.name]
  write_groups = [openwebui_group.g.name]
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_skill.test", "read_groups.#", "1"),
					resource.TestCheckResourceAttr("openwebui_skill.test", "read_groups.0", groupName),
					resource.TestCheckResourceAttr("openwebui_skill.test", "write_groups.#", "1"),
					resource.TestCheckResourceAttr("openwebui_skill.test", "write_groups.0", groupName),
				),
			},
			{
				ResourceName:      "openwebui_skill.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
