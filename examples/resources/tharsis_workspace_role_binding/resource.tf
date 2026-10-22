resource "tharsis_workspace_role_binding" "wrb" {
  workspace_id = tharsis_workspace.this.id
  role_name    = "deployer"
}
