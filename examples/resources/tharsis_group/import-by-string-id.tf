# A group can be imported by its TRN, which is "trn:group:" followed by the
# group's full path. A top-level group is just its name, for example
# "trn:group:my-group".
import {
  to = tharsis_group.nested
  id = "trn:group:my-group/my-sub-group"
}

# The group's ID also works as the id, shown by the "id" attribute or in the
# Tharsis UI.
