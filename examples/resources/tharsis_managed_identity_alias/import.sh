# An alias can be imported by its path, which is the path of the group holding
# the alias plus the alias name.
terraform import tharsis_managed_identity_alias.this my-group/my-sub-group/my-identity-alias

# An alias is a managed identity on the Tharsis side, so it carries a
# managed_identity TRN rather than one of its own.
terraform import tharsis_managed_identity_alias.this trn:managed_identity:my-group/my-sub-group/my-identity-alias

# The alias ID, shown by the "id" attribute or in the Tharsis UI, also works.
terraform import tharsis_managed_identity_alias.this <alias-id>
