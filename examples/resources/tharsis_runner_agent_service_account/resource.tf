resource "tharsis_runner_agent_service_account" "this" {
  runner_agent_id    = tharsis_runner_agent.this.id
  service_account_id = tharsis_service_account.this.id
}
