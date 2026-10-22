# A Terraform module can be imported by its path, which is the group path plus
# the module name plus the target system.
terraform import tharsis_terraform_module.this my-group/my-module/aws

# The module's TRN also works.
terraform import tharsis_terraform_module.this trn:terraform_module:my-group/my-module/aws

# So does its ID, shown by the "id" attribute or in the Tharsis UI.
terraform import tharsis_terraform_module.this <terraform-module-id>
