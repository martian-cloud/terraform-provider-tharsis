# A workspace can be imported by its TRN, which is "trn:workspace:" followed by
# the group path and the workspace name.
import {
  to = tharsis_workspace.this
  id = "trn:workspace:my-group/my-workspace"
}

# The workspace's ID also works as the id, shown by the "id" attribute or in the
# Tharsis UI.
