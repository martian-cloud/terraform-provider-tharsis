resource "tharsis_service_account" "this" {
  name        = "my-service-account"
  description = "A service account for CI."
  group_id    = tharsis_group.top_level.id

  oidc_trust_policies = [
    {
      issuer = "https://gitlab.com"
      bound_claims = {
        "sub" = "project_path:my-org/my-project:ref_type:branch:ref:main"
      }
    }
  ]
}
