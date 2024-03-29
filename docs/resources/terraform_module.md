---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "tharsis_terraform_module Resource - terraform-provider-tharsis"
subcategory: ""
description: |-
  Defines and manages a Terraform module.
---

# tharsis_terraform_module (Resource)

Defines and manages a Terraform module.



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `group_path` (String) The group path for this module.
- `name` (String) The name of the Terraform module.
- `system` (String) The target system for the module (e.g. aws, azure, etc.).

### Optional

- `private` (Boolean) Whether other groups are blocked from seeing this module.
- `repository_url` (String) The URL in a repository where this module is found.

### Read-Only

- `id` (String) String identifier of the Terraform module.
- `last_updated` (String) Timestamp when this terraform module was most recently updated.
- `registry_namespace` (String) The top-level group in which this module resides.
- `resource_path` (String) The path of the parent namespace plus the name of the terraform module.
