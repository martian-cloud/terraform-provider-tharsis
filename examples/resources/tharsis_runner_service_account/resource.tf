resource "tharsis_runner_service_account" "this" {
  runner_id          = tharsis_runner.this.id
  service_account_id = tharsis_service_account.this.id
}
