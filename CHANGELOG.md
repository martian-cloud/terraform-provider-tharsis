## Unreleased

### Added

- Import support for `tharsis_assigned_managed_identity`. An assignment has no identifier of its
  own in Tharsis, so it is imported using a composite ID of the form
  `<workspace_id>/<managed_identity_id>`. Both halves must be IDs; TRNs contain forward slashes
  and are rejected with an explanatory error.
- Import documentation for every importable resource. Each resource now has an `examples/resources`
  entry with both a usage example and an `import.sh` showing each accepted form of the import ID,
  which `tfplugindocs` renders into an Import section on the resource's documentation page. No
  resource page previously mentioned import at all, which left the existing support undiscoverable.
- Acceptance test coverage for `tharsis_workspace_vcs_provider_link`, which had an `ImportState`
  implementation but no tests.

### Fixed

- Import by resource path now works for `tharsis_gpg_key`, `tharsis_managed_identity`,
  `tharsis_managed_identity_access_rule`, `tharsis_managed_identity_alias`,
  `tharsis_service_account`, `tharsis_terraform_module`, `tharsis_terraform_provider`,
  `tharsis_variable`, `tharsis_vcs_provider`, `tharsis_workspace_role_binding`, and
  `tharsis_workspace_vcs_provider_link`.

  These resources passed the supplied import ID straight through to the API, which accepts only
  TRNs and GIDs. Importing by a human-readable path failed with `invalid identifier format: value
  is neither a valid TRN nor GID`. All resources now normalize the import ID, so a path, a GID, or
  a TRN are accepted interchangeably. Only `tharsis_group` and `tharsis_workspace` previously did
  this, which is why import appeared to work for some resources but not others.

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
