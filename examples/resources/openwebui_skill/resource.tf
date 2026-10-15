resource "openwebui_skill" "example" {
  skill_id    = "customer-support-playbook"
  name        = "Customer Support Playbook"
  description = "Step-by-step instructions for handling support escalations"
  content     = "When a customer reports an outage, first check the status page, then..."
  is_active   = true
  tags        = ["support", "playbook"]

  read_groups  = ["Support"]
  write_groups = ["Support"]
}
