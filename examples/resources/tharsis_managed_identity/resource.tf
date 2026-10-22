resource "tharsis_managed_identity" "aws" {
  name        = "my-aws-identity"
  description = "Federated AWS identity."
  type        = "aws_federated"
  group_id    = tharsis_group.top_level.id
  aws_role    = "arn:aws:iam::123456789012:role/my-role"
}

# Azure and Tharsis identities use their own attributes instead of aws_role:
#
# resource "tharsis_managed_identity" "azure" {
#   name            = "my-azure-identity"
#   type            = "azure_federated"
#   group_id        = tharsis_group.top_level.id
#   azure_client_id = "<client_id>"
#   azure_tenant_id = "<tenant_id>"
# }
#
# resource "tharsis_managed_identity" "tharsis" {
#   name                         = "my-tharsis-identity"
#   type                         = "tharsis_federated"
#   group_id                     = tharsis_group.top_level.id
#   tharsis_service_account_path = "my-group/my-service-account"
# }
