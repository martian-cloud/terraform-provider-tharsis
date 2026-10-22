resource "tharsis_workspace" "this" {
  name                 = "my-workspace"
  description          = "A workspace managed by Terraform."
  group_id             = tharsis_group.top_level.id
  terraform_version    = "1.9.0"
  max_job_duration     = 720
  prevent_destroy_plan = false
}
