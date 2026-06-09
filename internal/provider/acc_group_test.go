package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGroupResource(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-group")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGroupResourceConfig(name, "Initial description"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_group.test", "name", name),
					resource.TestCheckResourceAttr("openwebui_group.test", "description", "Initial description"),
					resource.TestCheckResourceAttrSet("openwebui_group.test", "id"),
				),
			},
			{
				ResourceName:      "openwebui_group.test",
				ImportState:       true,
				ImportStateVerify: true,
				// permissions sub-maps are computed and may have server defaults not in config.
				ImportStateVerifyIgnore: []string{"permissions"},
			},
			{
				Config: testAccGroupResourceConfig(name, "Updated description"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_group.test", "name", name),
					resource.TestCheckResourceAttr("openwebui_group.test", "description", "Updated description"),
				),
			},
		},
	})
}

func TestAccGroupDataSource(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-group-ds")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGroupDataSourceConfig(name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.openwebui_group.test", "name", name),
					resource.TestCheckResourceAttrSet("data.openwebui_group.test", "id"),
					resource.TestCheckResourceAttrSet("data.openwebui_group.test", "group_id"),
				),
			},
		},
	})
}

func testAccGroupResourceConfig(name, description string) string {
	return fmt.Sprintf(`%s
resource "openwebui_group" "test" {
  name        = %q
  description = %q
}
`, testAccProviderConfig(), name, description)
}

// TestAccGroupResource_EmptyUsers verifies that setting users = [] (an explicit
// empty list) does not trigger "provider produced inconsistent result after
// apply".  This was broken before the users attribute became Optional+Computed
// because fetchUsernamesForIDs returned a nil slice, which types.ListValueFrom
// mapped to a null list — inconsistent with the planned empty list.
func TestAccGroupResource_EmptyUsers(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-group-empty-users")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`%s
resource "openwebui_group" "test" {
  name        = %q
  description = "empty users test"
  users       = []
}
`, testAccProviderConfig(), name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_group.test", "name", name),
					resource.TestCheckResourceAttr("openwebui_group.test", "users.#", "0"),
				),
			},
		},
	})
}

// TestAccGroupResource_WithUser verifies that a named user can be added to a
// group and that the state round-trips correctly.  Skipped when
// OPENWEBUI_TEST_USER_EMAIL is not set.
func TestAccGroupResource_WithUser(t *testing.T) {
	email := os.Getenv("OPENWEBUI_TEST_USER_EMAIL")
	if email == "" {
		t.Skip("OPENWEBUI_TEST_USER_EMAIL not set")
	}
	name := acctest.RandomWithPrefix("tf-acc-group-with-user")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			// Create with user.
			{
				Config: fmt.Sprintf(`%s
resource "openwebui_group" "test" {
  name        = %q
  description = "with user test"
  users       = [%q]
}
`, testAccProviderConfig(), name, email),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_group.test", "users.#", "1"),
					resource.TestCheckResourceAttr("openwebui_group.test", "users.0", email),
				),
			},
			// Remove user — should not error and users should become empty.
			{
				Config: fmt.Sprintf(`%s
resource "openwebui_group" "test" {
  name        = %q
  description = "with user test"
  users       = []
}
`, testAccProviderConfig(), name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_group.test", "users.#", "0"),
				),
			},
		},
	})
}

func TestAccGroupResource_WithPermissions(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-group-perms")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`%s
resource "openwebui_group" "test" {
  name        = %q
  description = "permissions test"
  permissions = {
    access_grants = { allow_users = true }
    settings      = { interface = true }
  }
}
`, testAccProviderConfig(), name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_group.test", "name", name),
					resource.TestCheckResourceAttr("openwebui_group.test", "permissions.access_grants.allow_users", "true"),
					resource.TestCheckResourceAttr("openwebui_group.test", "permissions.settings.interface", "true"),
				),
			},
			{
				Config: fmt.Sprintf(`%s
resource "openwebui_group" "test" {
  name        = %q
  description = "permissions test"
  permissions = {
    access_grants = { allow_users = false }
    settings      = { interface = false }
  }
}
`, testAccProviderConfig(), name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_group.test", "permissions.access_grants.allow_users", "false"),
					resource.TestCheckResourceAttr("openwebui_group.test", "permissions.settings.interface", "false"),
				),
			},
			{
				ResourceName:            "openwebui_group.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"permissions"},
			},
		},
	})
}

func testAccGroupDataSourceConfig(name string) string {
	return fmt.Sprintf(`%s
resource "openwebui_group" "seed" {
  name        = %q
  description = "Data source lookup test"
}

data "openwebui_group" "test" {
  name = openwebui_group.seed.name
}
`, testAccProviderConfig(), name)
}
