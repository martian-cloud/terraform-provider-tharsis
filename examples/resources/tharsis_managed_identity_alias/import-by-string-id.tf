# An alias is a managed identity on the Tharsis side, so its TRN is
# "trn:managed_identity:" followed by the path of the group that holds the alias
# and the alias name.
import {
  to = tharsis_managed_identity_alias.this
  id = "trn:managed_identity:my-group/my-sub-group/my-identity-alias"
}

# The alias's ID also works as the id, shown by the "id" attribute or in the
# Tharsis UI.
