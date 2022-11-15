package tharsis

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (

	// Provider configuration uses environment variables:
	//   THARSIS_ENDPOINT
	//   THARSIS_STATIC_TOKEN
	providerConfig = `
provider "tharsis" {
}
`
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
				Config: providerConfig,
				Check:  resource.ComposeAggregateTestCheckFunc(
				// No check to do here.
				),
			},
		},
	})
}

// The End.
