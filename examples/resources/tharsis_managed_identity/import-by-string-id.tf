# A managed identity can be imported by its TRN, which is "trn:managed_identity:"
# followed by the group path and the identity name.
import {
  to = tharsis_managed_identity.aws
  id = "trn:managed_identity:my-group/my-aws-identity"
}

# The identity's ID also works as the id, shown by the "id" attribute or in the
# Tharsis UI.
