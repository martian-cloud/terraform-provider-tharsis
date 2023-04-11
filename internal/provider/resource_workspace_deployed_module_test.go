package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	tharsis "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/config"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

func TestWorkspaceDeployedModule(t *testing.T) {
	ws1Name := "workspace-1"
	ws1Desc := "this is workspace 1"
	ws2Name := "workspace-2"
	ws2Desc := "this is workspace 2"

	// Don't leave the pre-config resources around after this function is finished.
	// Must defer in case any test steps fail.
	PreConfigForTestWorkspaceDeployedModule()
	defer PostDestroyForTestWorkspaceDeployedModule()

	resource.Test(t, resource.TestCase{

		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{

			// Create two workspaces and perhaps other resources.
			{
				Config: testWorkspaceDeployedModuleConfigurationCreate(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify values that should be known.
					resource.TestCheckResourceAttr("tharsis_workspace.tw1", "name", ws1Name),
					resource.TestCheckResourceAttr("tharsis_workspace.tw1", "description", ws1Desc),
					resource.TestCheckResourceAttr("tharsis_workspace.tw2", "name", ws2Name),
					resource.TestCheckResourceAttr("tharsis_workspace.tw2", "description", ws2Desc),
				),
			},

			// FIXME: Write the tests.

			// Planned steps:
			// 1. Create the remote workspace, uploaded module, etc.; then do the apply/create run.
			// 2. Repeat the apply/create run with no changes.
			// 3. Do an apply/create run with changes to a variable's value.
			// 4. Do a destroy/delete run.

			// Destroy should mostly be covered automatically by TestCase.
			// The leftovers are handled by the deferred PostDestroy... function.

		},
	})
}

// PreConfigForTestWorkspaceDeployedModule pre-configures some resources that our provider does
// not support creating from HCL, including any pre-requisites (like the root group).
func PreConfigForTestWorkspaceDeployedModule() {

	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("ERROR 1: %s", err)
		return
	}

	client, err := tharsis.NewClient(cfg)
	if err != nil {
		fmt.Printf("ERROR 2: %s", err)
		return
	}

	// Create the root group.
	ctx := context.Background()
	_, err = client.Group.CreateGroup(ctx,
		&types.CreateGroupInput{
			Name:        testGroupPath,
			Description: "This is the root group for testing workspace deployed module.",
		},
	)
	if err != nil {
		fmt.Printf("ERROR 3: %s", err)
		return
	}

	fmt.Printf("Function PreConfigForTestWorkspaceDeployedModule succeeded.\n")
}

// PostDestroyForTestWorkspaceDeployedModule destroys the resources created by the pre-config function.
func PostDestroyForTestWorkspaceDeployedModule() {

	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("ERROR 11: %s", err)
		return
	}

	client, err := tharsis.NewClient(cfg)
	if err != nil {
		fmt.Printf("ERROR 12: %s", err)
		return
	}

	// Create the root group.
	ctx := context.Background()
	err = client.Group.DeleteGroup(ctx,
		&types.DeleteGroupInput{
			GroupPath: ptr.String(testGroupPath),
		},
	)
	if err != nil {
		fmt.Printf("ERROR 13: %s", err)
		return
	}

	fmt.Printf("Function PostDestroyForTestWorkspaceDeployedModule succeeded.\n")
}

func testWorkspaceDeployedModuleConfigurationCreate() string {
	ws1Name := "workspace-1"
	ws1Desc := "this is workspace 1"
	ws2Name := "workspace-2"
	ws2Desc := "this is workspace 2"
	wsPreventDestroyPlan := false

	return fmt.Sprintf(`

# Root group has already been created by pre-config.

resource "tharsis_workspace" "tw1" {
	name = "%s"
	description = "%s"
	group_path = "%s"
	prevent_destroy_plan = "%v"
}

resource "tharsis_workspace" "tw2" {
	name = "%s"
	description = "%s"
	group_path = "%s"
	prevent_destroy_plan = "%v"
}

	`,
		ws1Name, ws1Desc, testGroupPath, wsPreventDestroyPlan,
		ws2Name, ws2Desc, testGroupPath, wsPreventDestroyPlan,
	)
}

// The End.
