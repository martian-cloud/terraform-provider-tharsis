# A namespace variable can be imported by its TRN, which is "trn:variable:"
# followed by the namespace path, the category ("terraform" or "environment"),
# and the key.
import {
  to = tharsis_variable.region
  id = "trn:variable:my-group/my-workspace/terraform/region"
}

# A variable set on a group rather than a workspace uses the group's path.
import {
  to = tharsis_variable.log_level
  id = "trn:variable:my-group/environment/TF_LOG"
}

# The variable's ID also works as the id, shown by the "id" attribute.
