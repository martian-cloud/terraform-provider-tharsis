resource "tharsis_vcs_provider" "this" {
  name                 = "my-vcs-provider"
  description          = "GitLab VCS provider."
  type                 = "gitlab"
  group_id             = tharsis_group.top_level.id
  url                  = "https://gitlab.com"
  oauth_client_id      = "<oauth_client_id>"
  oauth_client_secret  = "<oauth_client_secret>"
  auto_create_webhooks = true
}
