terraform {
  required_providers {
    tharsis = {
      source = "registry.terraform.io/martian-cloud/tharsis"
    }
  }
}

provider "tharsis" {
  host         = "<tharsis_api_host>"
  static_token = "<static_token>"
}

data "tharsis_workspace_outputs" "this" {
  path = "group/sub-group/workspace"
}

# When running via a Tharsis executor, in a workspace,
# the path can be relative to the workspace.
#
# For instance, if you had the following structure where
# you are operating from myworkspace:
#   group
#   |- sub-group
#   |--|- workspace
#   |--my-group
#   |--|- myworkspace  <- this is the current workspace
#
#  You can access `workspace` relative to your `myworkspace`
#  by using the relative path `../sub-group/workspace`
#
# data "tharsis_workspace_outputs" "this" {
#   path = "../sub-group/workspace"
# }

output "str" {
  value = data.tharsis_workspace_outputs.this.outputs.output_name
}
