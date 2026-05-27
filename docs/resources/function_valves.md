# openwebui_function_valves

Configures the Valves (settings) of an Open WebUI function.

## Example Usage

```hcl
resource "openwebui_function_valves" "example" {
  function_id = openwebui_function.example_filter.id

  valves_json = jsonencode({
    priority = 5
  })
}
```

## Argument Reference

* `function_id` (Required, forces replacement) — Function identifier to configure.
* `valves_json` (Optional) — JSON object of valve settings.

## Attribute Reference

* `id` — Mirrors `function_id`.
* `spec_json` — JSON schema describing the available valve settings.

## Import

```bash
terraform import openwebui_function_valves.example example_filter
```
