resource "tharsis_runner" "this" {
  name              = "my-runner"
  description       = "A runner for a Kubernetes cluster."
  group_id          = tharsis_group.top_level.id
  run_untagged_jobs = false
  tags              = ["kubernetes", "us-east-2"]
}
