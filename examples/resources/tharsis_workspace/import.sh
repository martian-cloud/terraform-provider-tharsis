# A workspace can be imported by its full path, which is the group path plus the
# workspace name.
terraform import tharsis_workspace.this my-group/my-workspace

# The workspace's TRN also works.
terraform import tharsis_workspace.this trn:workspace:my-group/my-workspace

# So does the workspace's ID, shown by the "id" attribute or in the Tharsis UI.
terraform import tharsis_workspace.this <workspace-id>
