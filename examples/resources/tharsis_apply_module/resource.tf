resource "tharsis_apply_module" "this" {
  workspace_id   = tharsis_workspace.this.id
  module_source  = "tharsis.example.com/my-group/my-module/aws"
  module_version = "1.0.0"

  variables = [
    {
      category = "terraform"
      key      = "region"
      value    = "us-east-1"
    }
  ]
}
