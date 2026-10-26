# A link has no human-readable name, so it is imported by its ID, shown by the
# "id" attribute or in the Tharsis UI.
import {
  to = tharsis_workspace_vcs_provider_link.this
  id = "<link-id>"
}

# A TRN also works, but its last segment is the same link ID, so it offers no
# advantage: "trn:workspace_vcs_provider_link:my-group/my-workspace/<link-id>".
