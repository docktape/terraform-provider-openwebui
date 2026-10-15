# Look up by name (skill names are unique)...
data "openwebui_skill" "by_name" {
  name = "Customer Support Playbook"
}

# ...or by identifier.
data "openwebui_skill" "by_id" {
  skill_id = "customer-support-playbook"
}
