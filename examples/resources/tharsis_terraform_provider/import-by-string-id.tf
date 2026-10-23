# A Terraform provider can be imported by its TRN, which is
# "trn:terraform_provider:" followed by the group path and the provider name.
import {
  to = tharsis_terraform_provider.this
  id = "trn:terraform_provider:my-group/my-provider"
}

# The provider's ID also works as the id, shown by the "id" attribute or in the
# Tharsis UI.
