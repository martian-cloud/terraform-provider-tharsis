## Unreleased

### Added

- Import support for `tharsis_assigned_managed_identity`. An assignment has no identifier of its
  own in Tharsis, so it is imported using a composite ID of the form
  `<workspace_id>/<managed_identity_id>`. Both halves must be IDs; TRNs contain forward slashes
  and are rejected with an explanatory error.
- Import documentation for every importable resource. Each resource page now has an Import
  section with an `import` block example, showing the ID or TRN to use. Previously no resource
  page mentioned import at all.
- Acceptance test coverage for `tharsis_workspace_vcs_provider_link`, which had an `ImportState`
  implementation but no tests. The test links to an existing VCS provider whose OAuth
  authorization has been completed, supplied through environment variables, and is skipped
  when they are not set.
- Acceptance test coverage for importing by TRN, alongside the existing import by ID.

### Changed

- Importing a `tharsis_apply_module` now fails with an explanation rather than Terraform's generic
  unsupported-import message. The resource represents the act of applying a module to a workspace
  rather than a stored Tharsis resource, so there is nothing for an import to read back.

## 0.0.1 (Unreleased)

Initial release of the Tharsis Provider for Terraform.

Added two data sources for Tharsis Workspace Outputs:
    - `tharsis_workspace_outputs`
        * Currently this data source only supports outputs that are strings. Once terraform supports dynamic schemas we'll add all types.
    - `tharsis_workspace_outputs_json`
        * Supports all output types but all output values are JSON encoded and need to be decoded to access the values.
