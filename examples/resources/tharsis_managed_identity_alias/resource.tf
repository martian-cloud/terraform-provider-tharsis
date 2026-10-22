resource "tharsis_managed_identity_alias" "this" {
  name            = "my-identity-alias"
  group_id        = tharsis_group.nested.id
  alias_source_id = tharsis_managed_identity.aws.id
}
