# A Terraform module can be imported by its TRN, which is "trn:terraform_module:"
# followed by the group path, the module name, and the target system.
import {
  to = tharsis_terraform_module.this
  id = "trn:terraform_module:my-group/my-module/aws"
}

# The module's ID also works as the id, shown by the "id" attribute or in the
# Tharsis UI.
