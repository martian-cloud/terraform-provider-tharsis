# Tharsis provider definition with static token
provider "tharsis" {
  host         = "<tharsis_api_host>"
  static_token = "<static_token>"
}

# # Tharsis provider using a service account
# provider "tharsis" {
#   host                  = "<tharsis_api_host>"
#   service_account_path  = "<service_account_path>"
#   service_account_token = "<service_account_token>"
# }
