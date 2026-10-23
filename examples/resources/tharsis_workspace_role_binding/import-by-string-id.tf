# A role binding is identified by the workspace it applies to, so it is
# imported using the workspace's TRN.
import {
  to = tharsis_workspace_role_binding.wrb
  id = "trn:workspace:my-group/my-workspace"
}

# The workspace's ID also works as the id, shown by the "id" attribute of a
# tharsis_workspace resource or in the Tharsis UI.
