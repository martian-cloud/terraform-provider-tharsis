## Unreleased

### Added

- New resource `tharsis_workspace_role_binding`, which binds a role to a workspace by `workspace_id`
  and `role_name`. It can be imported by the workspace's ID or TRN. (#57)
- Import support for `tharsis_assigned_managed_identity`. An assignment has no identifier of its
  own in Tharsis, so it is imported using a composite ID of the form
  `<workspace_id>/<managed_identity_id>`. Both halves must be IDs; TRNs contain forward slashes
  and are rejected with an explanatory error. (#58)
- Import documentation for every importable resource. Each resource page now has an Import
  section with an `import` block example, showing the ID or TRN to use. Previously no resource
  page mentioned import at all. (#58)
- Usage examples and descriptive text on the resource, data source, and provider documentation
  pages. (#57)
- Acceptance test coverage for `tharsis_workspace_vcs_provider_link`, which had an `ImportState`
  implementation but no tests. The test links to an existing VCS provider whose OAuth
  authorization has been completed, supplied through environment variables, and is skipped
  when they are not set. (#58)
- Acceptance test coverage for importing by TRN, alongside the existing import by ID. (#58)

### Changed

- `last_updated` now records the last change made by Terraform instead of being refreshed on every
  read, which removes the drift it caused. After an import there is no earlier value, so it is set
  from the update time reported by Tharsis. (#57)
- Attributes that accept a group, workspace, or managed identity identifier now document that either
  an ID or a TRN may be given. (#57)
- Updated the `tharsis-api` client dependency from v0.95.0 to v0.106.0 and `grpc` from v1.80.0 to
  v1.83.1. `tharsis_apply_module` was adjusted to match the newer client's run status values. (#57)
- Importing a `tharsis_apply_module` now fails with an explanation rather than Terraform's generic
  unsupported-import message. The resource represents the act of applying a module to a workspace
  rather than a stored Tharsis resource, so there is nothing for an import to read back. (#58)

## 0.0.1 (Unreleased)

Initial release of the Tharsis Provider for Terraform.

Added two data sources for Tharsis Workspace Outputs:
    - `tharsis_workspace_outputs`
        * Currently this data source only supports outputs that are strings. Once terraform supports dynamic schemas we'll add all types.
    - `tharsis_workspace_outputs_json`
        * Supports all output types but all output values are JSON encoded and need to be decoded to access the values.
