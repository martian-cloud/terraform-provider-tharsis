# A VCS provider can be imported by its path, which is the group path plus the
# provider name.
terraform import tharsis_vcs_provider.this my-group/my-vcs-provider

# The provider's TRN also works.
terraform import tharsis_vcs_provider.this trn:vcs_provider:my-group/my-vcs-provider

# So does its ID, shown by the "id" attribute or in the Tharsis UI.
terraform import tharsis_vcs_provider.this <vcs-provider-id>

# Note: oauth_client_id and oauth_client_secret are write-only and cannot be
# read back from Tharsis, and oauth_authorization_url is only returned when the
# provider is first created. After importing, set the two OAuth credentials in
# your configuration to match the values the provider was created with, or the
# next apply will update them.
