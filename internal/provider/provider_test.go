package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

var (
	testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
		"tharsis": providerserver.NewProtocol6WithError(New()),
	}
)

// TestProvider is a very simple preliminary test to connect to a provider.
func TestProvider(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testSharedProviderConfiguration(),
				Check:  resource.ComposeAggregateTestCheckFunc(
				// No check to do here.
				),
			},
		},
	})
}

// Provider configuration (used by several tests) uses environment variables:
//   THARSIS_ENDPOINT
//   THARSIS_STATIC_TOKEN
func testSharedProviderConfiguration() string {
	return `
provider "tharsis" {
}
	`
}

// The End.
