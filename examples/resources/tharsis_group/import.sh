# A group can be imported by its full path, which is the parent path plus the
# group name.
terraform import tharsis_group.nested my-group/my-sub-group

# A top-level group has no parent, so its full path is just its name.
terraform import tharsis_group.top_level my-group

# The group's TRN also works.
terraform import tharsis_group.nested trn:group:my-group/my-sub-group

# So does the group's ID, shown by the "id" attribute or in the Tharsis UI.
terraform import tharsis_group.nested <group-id>
