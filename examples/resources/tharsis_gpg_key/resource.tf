resource "tharsis_gpg_key" "this" {
  group_id    = tharsis_group.top_level.id
  ascii_armor = file("${path.module}/public-key.asc")
}
