# An access rule has no human-readable name, so import it by its ID, shown by
# the "id" attribute or in the Tharsis UI.
terraform import tharsis_managed_identity_access_rule.eligible_principals <access-rule-id>

# A TRN works too, but its last segment is the same rule ID, so it offers no
# advantage over using the ID on its own.
terraform import tharsis_managed_identity_access_rule.eligible_principals trn:managed_identity_access_rule:my-group/my-aws-identity/<access-rule-id>
