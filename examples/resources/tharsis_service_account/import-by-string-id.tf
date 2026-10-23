# A service account can be imported by its TRN, which is "trn:service_account:"
# followed by the group path and the service account name.
import {
  to = tharsis_service_account.this
  id = "trn:service_account:my-group/my-service-account"
}

# The service account's ID also works as the id, shown by the "id" attribute or
# in the Tharsis UI.
