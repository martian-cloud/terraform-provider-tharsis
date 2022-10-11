package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccExampleDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testAccExampleDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.scaffolding_example.test", "id", "example-id"),
				),
			},
		},
	})
}

const testAccExampleDataSourceConfig = `
data "scaffolding_example" "test" {
  configurable_attribute = "example"
}
`

func Test_resolvePath(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name      string
		groupPath *string
		args      args
		want      string
		wantErr   bool
	}{
		{
			name:      "Not providing a relative path but containing a slash is treated as a full path",
			groupPath: strPtr("group/subgroup"),
			args: args{
				path: "deepgroup/workspace",
			},
			want:    "deepgroup/workspace",
			wantErr: false,
		},
		{
			name:      "Tharsis Group Path isn't set, returns error with relative path",
			groupPath: nil,
			args: args{
				path: "../subgroup/workspace",
			},
			want:    "",
			wantErr: true,
		},
		{
			name:      "Tharsis Group Path isn't set, returns original path when its a full path",
			groupPath: nil,
			args: args{
				path: "group/subgroup/workspace",
			},
			want:    "group/subgroup/workspace",
			wantErr: false,
		},
		{
			name:      "Tharsis Group Path is empty, returns an error",
			groupPath: strPtr(""),
			args: args{
				path: "../workspace",
			},
			want:    "",
			wantErr: true,
		},
		{
			name:      "Too many relative paths up can result in an invalid path",
			groupPath: strPtr("group/subgroup"),
			args: args{
				path: "../../workspace",
			},
			want:    "",
			wantErr: true,
		},
		{
			name:      "Relative paths up can result in a valid workspace",
			groupPath: strPtr("group/subgroup"),
			args: args{
				path: "../workspace",
			},
			want:    "group/workspace",
			wantErr: false,
		},
		{
			name:      "Relative paths up and down can result in a valid workspace",
			groupPath: strPtr("group/subgroup"),
			args: args{
				path: "../../group2/workspace",
			},
			want:    "group2/workspace",
			wantErr: false,
		},
		{
			name:      "Providing only a workspace, results in the full path",
			groupPath: strPtr("group/subgroup"),
			args: args{
				path: "workspace",
			},
			want:    "group/subgroup/workspace",
			wantErr: false,
		},
		{
			name:      "Providing a dot slash in path results in a subgroup of the current group",
			groupPath: strPtr("group/subgroup"),
			args: args{
				path: "./deepgroup/workspace",
			},
			want:    "group/subgroup/deepgroup/workspace",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		prevValue, ok := os.LookupEnv(tharsisGroupPathEnvVar)
		if tt.groupPath != nil {
			if err := os.Setenv(tharsisGroupPathEnvVar, *tt.groupPath); err != nil {
				t.Fatalf("cannot set environment variable: %v", err)
			}
		}
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolvePath(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("resolvePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("resolvePath() = %v, want %v", got, tt.want)
			}
		})

		if ok {
			os.Setenv(tharsisGroupPathEnvVar, prevValue)
		} else {
			os.Unsetenv(tharsisGroupPathEnvVar)
		}
	}
}

func strPtr(str string) *string {
	return &str
}
