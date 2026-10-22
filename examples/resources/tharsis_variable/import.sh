# A namespace variable can be imported by its path, which is the namespace path
# plus the category ("terraform" or "environment") plus the key.
terraform import tharsis_variable.region my-group/my-workspace/terraform/region

# A variable on a group rather than a workspace uses that group's path.
terraform import tharsis_variable.log_level my-group/environment/TF_LOG

# The variable's TRN also works.
terraform import tharsis_variable.region trn:variable:my-group/my-workspace/terraform/region

# So does the variable's ID, shown by the "id" attribute.
terraform import tharsis_variable.region <variable-id>
