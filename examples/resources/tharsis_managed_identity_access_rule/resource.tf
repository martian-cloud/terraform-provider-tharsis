resource "tharsis_managed_identity_access_rule" "eligible_principals" {
  managed_identity_id         = tharsis_managed_identity.aws.id
  type                        = "eligible_principals"
  run_stage                   = "plan"
  allowed_service_account_ids = [tharsis_service_account.this.id]
}

resource "tharsis_managed_identity_access_rule" "module_attestation" {
  managed_identity_id  = tharsis_managed_identity.aws.id
  type                 = "module_attestation"
  run_stage            = "apply"
  verify_state_lineage = true

  module_attestation_policies = [
    {
      public_key     = file("${path.module}/attestation-key.pem")
      predicate_type = "https://slsa.dev/provenance/v1"
    }
  ]
}
