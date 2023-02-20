---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "tharsis_workspace_vcs_provider_link Resource - terraform-provider-tharsis"
subcategory: ""
description: |-
  Defines and manages a workspace VCS provider link.
---

# tharsis_workspace_vcs_provider_link (Resource)

Defines and manages a workspace VCS provider link.



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `auto_speculative_plan` (Boolean) Whether to create speculative plans automatically for PRs.
- `glob_patterns` (List of String) Glob patterns to use for monitoring changes.
- `module_directory` (String) The module's directory path.
- `repository_path` (String) The path portion of the repository URL.
- `vcs_provider_id` (String) The string identifier of the  VCS provider.
- `webhook_disabled` (Boolean) Whether to disable the webhook.
- `workspace_path` (String) The resource path of the workspace.

### Optional

- `branch` (String) The repository branch.
- `tag_regex` (String) A regular expression that specifies which tags trigger runs.

### Read-Only

- `id` (String) String identifier of the workspace VCS provider link.
- `last_updated` (String) Timestamp when this workspace VCS provider link was most recently updated.
- `webhook_id` (String) String identifier of the webhook.
- `workspace_id` (String) The ID of the workspace.

