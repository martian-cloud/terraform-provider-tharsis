resource "tharsis_terraform_module" "this" {
  name           = "my-module"
  system         = "aws"
  group_id       = tharsis_group.top_level.id
  private        = true
  repository_url = "https://gitlab.com/my-org/terraform-aws-my-module"
}
