resource "tharsis_workspace_vcs_provider_link" "this" {
  workspace_id          = tharsis_workspace.this.id
  vcs_provider_id       = tharsis_vcs_provider.this.id
  repository_path       = "my-org/my-project"
  branch                = "main"
  module_directory      = "terraform"
  glob_patterns         = ["terraform/**"]
  auto_speculative_plan = true
  webhook_disabled      = false
}
