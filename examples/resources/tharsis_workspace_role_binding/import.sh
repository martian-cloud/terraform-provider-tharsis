# A role binding is identified by the workspace it applies to, so import it
# using the workspace's path.
terraform import tharsis_workspace_role_binding.wrb my-group/my-workspace

# The workspace's TRN also works.
terraform import tharsis_workspace_role_binding.wrb trn:workspace:my-group/my-workspace

# So does the workspace's ID, shown by the "id" attribute of a tharsis_workspace
# resource or in the Tharsis UI.
terraform import tharsis_workspace_role_binding.wrb <workspace-id>
