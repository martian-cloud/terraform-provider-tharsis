resource "tharsis_assigned_managed_identity" "this" {
  workspace_id        = tharsis_workspace.this.id
  managed_identity_id = tharsis_managed_identity.aws.id
}
