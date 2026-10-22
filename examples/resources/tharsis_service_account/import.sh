# A service account can be imported by its path, which is the group path plus
# the service account name.
terraform import tharsis_service_account.this my-group/my-service-account

# The service account's TRN also works.
terraform import tharsis_service_account.this trn:service_account:my-group/my-service-account

# So does its ID, shown by the "id" attribute or in the Tharsis UI.
terraform import tharsis_service_account.this <service-account-id>
