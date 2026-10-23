# A VCS provider can be imported by its TRN, which is "trn:vcs_provider:"
# followed by the group path and the provider name.
import {
  to = tharsis_vcs_provider.this
  id = "trn:vcs_provider:my-group/my-vcs-provider"
}

# The provider's ID also works as the id, shown by the "id" attribute or in the
# Tharsis UI.
#
# Note: oauth_client_id and oauth_client_secret are write-only and cannot be read
# back from Tharsis, and oauth_authorization_url is only returned when the
# provider is first created. After importing, set the two OAuth credentials in
# your configuration to match the values the provider was created with, or the
# next apply will update them.
