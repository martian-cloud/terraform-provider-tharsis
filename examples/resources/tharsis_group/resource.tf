resource "tharsis_group" "top_level" {
  name        = "my-group"
  description = "A top-level group."
}

resource "tharsis_group" "nested" {
  name        = "my-sub-group"
  description = "A nested group."
  parent_id   = tharsis_group.top_level.id
}
