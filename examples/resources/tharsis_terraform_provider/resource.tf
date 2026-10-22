resource "tharsis_terraform_provider" "this" {
  name           = "my-provider"
  group_id       = tharsis_group.top_level.id
  private        = true
  repository_url = "https://github.com/my-org/terraform-provider-my-provider"
}
