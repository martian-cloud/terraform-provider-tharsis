# Inside a Tharsis job, no arguments are needed: the API host and an
# authentication token are provided to the job automatically.
#
# A job is read-only by default. To create or change resources in the parent
# group of the workspace running the job, assign the workspace a role with a
# workspace role binding.
provider "tharsis" {}

# # Outside Tharsis, using a static token
# provider "tharsis" {
#   host         = "<tharsis_api_host>"
#   static_token = "<static_token>"
# }

# # Outside Tharsis, using a service account
# provider "tharsis" {
#   host                  = "<tharsis_api_host>"
#   service_account_id    = "<service_account_id>"
#   service_account_token = "<service_account_token>"
# }
