# A link has no human-readable name, so import it by its ID, shown by the "id"
# attribute or in the Tharsis UI.
terraform import tharsis_workspace_vcs_provider_link.this <link-id>

# A TRN works too, but its last segment is the same link ID, so it offers no
# advantage over using the ID on its own.
terraform import tharsis_workspace_vcs_provider_link.this trn:workspace_vcs_provider_link:my-group/my-workspace/<link-id>
