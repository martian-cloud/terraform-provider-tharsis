# A managed identity can be imported by its path, which is the group path plus
# the identity name.
terraform import tharsis_managed_identity.aws my-group/my-aws-identity

# The identity's TRN also works.
terraform import tharsis_managed_identity.aws trn:managed_identity:my-group/my-aws-identity

# So does the identity's ID, shown by the "id" attribute or in the Tharsis UI.
terraform import tharsis_managed_identity.aws <managed-identity-id>
