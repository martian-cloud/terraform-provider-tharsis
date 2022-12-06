package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
	ttypes "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

const (
	tharsisGroupPathEnvVar = "THARSIS_GROUP_PATH"
)

// WorkspacesOutputsDataSourceData represents the outputs for a workspace in Tharsis.
type WorkspacesOutputsDataSourceData struct {
	Outputs        map[string]string `tfsdk:"outputs"`
	Path           types.String      `tfsdk:"path"`
	FullPath       types.String      `tfsdk:"full_path"`
	WorkspaceID    types.String      `tfsdk:"workspace_id"`
	StateVersionID types.String      `tfsdk:"state_version_id"`
}

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ datasource.DataSource = workspaceOutputsDataSource{}
)

// Metadata effectively replaces the DataSourceType (and thus workspaceOutputsDataSourceType)
// It returns the full name of the data source.
func (t workspaceOutputsDataSource) Metadata(_ context.Context,
	req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	typeName := "tharsis_workspace_outputs"
	if t.isJSONEncoded {
		typeName += "_json"
	}
	resp.TypeName = typeName
}

func (t workspaceOutputsDataSource) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Version: 1,

		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Tharsis Workspace Outputs data source is used to retrieve outputs from workspace under a given path.",
		Description:         "Tharsis Workspace Outputs data source is used to retrieve outputs from workspace under a given path.",

		Attributes: map[string]tfsdk.Attribute{
			"path": {
				MarkdownDescription: "The path of the workspace to retrieve outputs.",
				Description:         "The path of the workspace to retrieve outputs.",
				Optional:            false,
				Required:            true,
				Type:                types.StringType,
			},
			"full_path": {
				MarkdownDescription: "The full path of the workspace.",
				Description:         "The full path of the workspace.",
				Type:                types.StringType,
				Computed:            true,
			},
			"workspace_id": {
				MarkdownDescription: "The ID of the workspace.",
				Description:         "The ID of the workspace.",
				Type:                types.StringType,
				Computed:            true,
			},
			"state_version_id": {
				MarkdownDescription: "The ID of the workspace's current state version.",
				Description:         "The ID of the workspace's current state version.",
				Type:                types.StringType,
				Computed:            true,
			},
			"outputs": {
				MarkdownDescription: "The outputs of the workspace specified by the path.",
				Description:         "The outputs of the workspace specified by the path.",
				Type: types.MapType{
					ElemType: types.StringType,
				},
				Computed: true,
			},
		},
	}, nil
}

type workspaceOutputsDataSource struct {
	provider      tharsisProvider
	isJSONEncoded bool
}

func (t workspaceOutputsDataSource) Read(ctx context.Context,
	req datasource.ReadRequest, resp *datasource.ReadResponse) {
	defer func() {
		if r := recover(); r != nil {
			resp.Diagnostics.AddError("Oops! Something went wrong", fmt.Sprintf("%v\n%v", r, string(debug.Stack())))
			return
		}
	}()

	var data WorkspacesOutputsDataSourceData
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.Path.Unknown || data.Path.Null {
		resp.Diagnostics.AddError(
			"Path is required",
			"Path cannot be null or unknown",
		)
		return
	}

	path, err := resolvePath(data.Path.Value)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error resolving full path of workspace",
			err.Error(),
		)
		return
	}

	// For later dereference, input.Path is known to not be nil.
	input := &ttypes.GetWorkspaceInput{
		Path: &path,
	}

	workspace, err := t.provider.client.Workspaces.GetWorkspace(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error retrieving workspace",
			err.Error(),
		)
		return
	}

	if workspace == nil {
		resp.Diagnostics.AddError(
			"Couldn't find workspace",
			fmt.Sprintf("Workspace '%s' could not be found. Either the workspace doesn't exist or you don't have access.", *input.Path),
		)
		return
	}

	if workspace.CurrentStateVersion == nil {
		resp.Diagnostics.AddError(
			"Workspace doesn't have a current state version",
			fmt.Sprintf("Workspace '%s' does not have a current state version.", *input.Path),
		)
		return
	}

	data.Outputs = map[string]string{}
	for _, output := range workspace.CurrentStateVersion.Outputs {
		if !t.isJSONEncoded {
			switch output.Type {
			// Currently Strings are only supported
			case cty.String:
			default:
				// Unsupported types for non-json encoded provider need to be skipped
				continue
			}
		}

		b, err := ctyjson.Marshal(output.Value, output.Type)
		if err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Fail to parse value from output \"%s\"", output.Name),
				err.Error(),
			)
		}

		if !t.isJSONEncoded {
			var s string
			if err := json.Unmarshal(b, &s); err != nil {
				resp.Diagnostics.AddError(
					fmt.Sprintf("Failed to parse value from output \"%s\"", output.Name),
					err.Error(),
				)
				return
			}
			data.Outputs[output.Name] = s
		} else {
			data.Outputs[output.Name] = string(b)
		}
	}

	// Add additional attributes
	data.FullPath = types.String{Value: path}
	data.WorkspaceID = types.String{Value: workspace.Metadata.ID}
	data.StateVersionID = types.String{Value: workspace.CurrentStateVersion.Metadata.ID}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func resolvePath(path string) (string, error) {
	// If the path contains a forward slash but no relative paths, return as it is a full path
	// We only need to check for `./` as `../` contains `./`
	if strings.Contains(path, "/") && !strings.Contains(path, "./") {
		return path, nil
	}

	val, present := os.LookupEnv(tharsisGroupPathEnvVar)
	// If the environment variable isn't present, we need to error
	// because relative paths cannot be resolved.
	if !present {
		return "", fmt.Errorf("Relative path was provided but the environment variable %s was undefined", tharsisGroupPathEnvVar)
	}

	// If the environment variable is an empty string, it is invalid
	if val == "" {
		return "", fmt.Errorf("Received an invalid Tharsis Group Path value")
	}

	// Add a leading '/' to the beginning so that it resolves to a full path and not relative
	// for the Clean function, then we remove the leading path to get the Tharsis path.
	path = filepath.Clean(filepath.Join("/", val, path))[1:]

	if !strings.Contains(path, "/") {
		return "", fmt.Errorf("Workspace must exist under at least one parent group")
	}

	return path, nil
}
