# A Terraform provider can be imported by its path, which is the group path plus
# the provider name.
terraform import tharsis_terraform_provider.this my-group/my-provider

# The provider's TRN also works.
terraform import tharsis_terraform_provider.this trn:terraform_provider:my-group/my-provider

# So does its ID, shown by the "id" attribute or in the Tharsis UI.
terraform import tharsis_terraform_provider.this <terraform-provider-id>
