## 0.0.1 (Unreleased)

Initial release of the Tharsis Provider for Terraform.

Added two data sources for Tharsis Workspace Outputs:
    - `tharsis_workspace_outputs`
        * Currently this data source only supports outputs that are strings. Once terraform supports dynamic schemas we'll add all types.
    - `tharsis_workspace_outputs_json`
        * Supports all output types but all output values are JSON encoded and need to be decoded to access the values.
