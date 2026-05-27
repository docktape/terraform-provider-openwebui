# openwebui_function

Manages an Open WebUI function — a Pipe, Filter, or Action plugin defined in Python.

The function `type` (`pipe`/`filter`/`action`) and `manifest_json` are derived by Open WebUI
from the source you provide; they are read-only. Activation (`is_active`) and global
application (`is_global`) default to `false` and are managed declaratively.

## Example Usage

```hcl
resource "openwebui_function" "example_filter" {
  function_id = "example_filter"
  name        = "Example Filter"
  description = "Adjusts requests before they reach the model"
  is_active   = true

  content = file("${path.module}/functions/example_filter.py")
}
```

## Argument Reference

* `function_id` (Required, forces replacement) — Function identifier. Must be a valid
  identifier (letters, digits, underscores); Open WebUI lowercases it.
* `name` (Required) — Display name.
* `content` (Required) — Python source. The class you define (`Pipe`, `Filter`, or `Action`)
  determines the function type.
* `description` (Optional) — Human-readable description (`meta.description`).
* `is_active` (Optional) — Whether the function is enabled. Defaults to `false`.
* `is_global` (Optional) — Whether the function applies to every chat automatically.
  Defaults to `false`.

## Attribute Reference

* `id` — Mirrors `function_id`.
* `type` — Derived function type: `pipe`, `filter`, or `action`.
* `manifest_json` — JSON manifest derived from the source frontmatter.
* `user_id` — Owner user identifier.
* `created_at` / `updated_at` — Unix timestamps.

## Import

```bash
terraform import openwebui_function.example_filter example_filter
```
