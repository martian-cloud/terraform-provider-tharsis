# An access rule has no human-readable name, so it is imported by its ID, shown
# by the "id" attribute or in the Tharsis UI.
import {
  to = tharsis_managed_identity_access_rule.eligible_principals
  id = "<access-rule-id>"
}

# A TRN also works, but its last segment is the same rule ID, so it offers no
# advantage: "trn:managed_identity_access_rule:my-group/my-aws-identity/<access-rule-id>".
