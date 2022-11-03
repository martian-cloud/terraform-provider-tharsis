package tharsis

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// WorkspacesOutputsDataSourceData represents the outputs for a workspace in Tharsis.
type WorkspacesOutputsDataSourceData struct {
	Path           types.String      `tfsdk:"path"`
	FullPath       types.String      `tfsdk:"full_path"`
	WorkspaceID    types.String      `tfsdk:"workspace_id"`
	StateVersionID types.String      `tfsdk:"state_version_id"`
	Outputs        map[string]string `tfsdk:"outputs"`
}
