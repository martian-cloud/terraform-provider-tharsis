resource "tharsis_variable" "region" {
  namespace_path = "my-group/my-workspace"
  category       = "terraform"
  key            = "region"
  value          = "us-east-1"
}

resource "tharsis_variable" "log_level" {
  namespace_path = "my-group"
  category       = "environment"
  key            = "TF_LOG"
  value          = "INFO"
}
