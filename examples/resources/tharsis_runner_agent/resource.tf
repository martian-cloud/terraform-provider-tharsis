resource "tharsis_runner_agent" "this" {
  name              = "my-runner-agent"
  description       = "A runner agent for a Kubernetes cluster."
  group_id          = tharsis_group.top_level.id
  run_untagged_jobs = false
  tags              = ["kubernetes", "us-east-2"]
}
