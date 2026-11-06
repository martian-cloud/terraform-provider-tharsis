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
- New resource `tharsis_runner`, a group runner, with `name`, `description`, `group_id`, `tags`,
  `disabled` and `run_untagged_jobs`. It can be imported by its ID. (#59)
- New resource `tharsis_runner_service_account_assignment`, which assigns a service account to a
  runner by `runner_id` and `service_account_id`, letting the runner authenticate as it. It is
  imported using a composite ID of the form `<runner_id>/<service_account_id>`; both halves must
  be IDs. (#59)
- The `kubernetes_federated` managed identity type, with an `audience` attribute that must match the
  client ID of the cluster's OIDC identity provider configuration. (#59)

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
- Reference attributes (`group_id`, `parent_id`, `workspace_id`, `managed_identity_id`,
  `alias_source_id`, `vcs_provider_id`, and the IDs on the assignment resources) keep the form written
  in the configuration, an ID or a TRN. Switching between the two forms of the same object is an
  in-place update rather than a replacement; moving a reference to a different object still
  replaces the resource. (#59)
- `tharsis_workspace_vcs_provider_link`'s deprecated `workspace_path` is now computed, so it is
  filled in when the link is configured by `workspace_id`. (#59)

### Fixed

- `tharsis_assigned_managed_identity` was removed from state on every refresh, and re-created on every
  apply, when `managed_identity_id` was a TRN. (#59)
- `tharsis_workspace_vcs_provider_link` could not be created with `workspace_id`: every apply failed
  with "Provider produced inconsistent result after apply" for `workspace_path`. (#59)
- Opt-in acceptance tests for `tharsis_workspace_vcs_provider_link` against a GitLab stand-in
  (`internal/provider/testdata/fake_vcs`), run when `THARSIS_TEST_FAKE_VCS_URL` is set. (#59)

## 0.0.1 (Unreleased)

Initial release of the Tharsis Provider for Terraform.

Added two data sources for Tharsis Workspace Outputs:
    - `tharsis_workspace_outputs`
        * Currently this data source only supports outputs that are strings. Once terraform supports dynamic schemas we'll add all types.
    - `tharsis_workspace_outputs_json`
        * Supports all output types but all output values are JSON encoded and need to be decoded to access the values.
