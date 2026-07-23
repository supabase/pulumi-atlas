package provider

import (
	_ "embed"

	pf "github.com/pulumi/pulumi-terraform-bridge/v3/pkg/pf/tfbridge"
	"github.com/pulumi/pulumi-terraform-bridge/v3/pkg/tfbridge"
	"github.com/supabase/pulumi-atlas/provider/version"
	tfp "github.com/supabase/terraform-provider-ripe-atlas/provider"
)

//go:embed cmd/pulumi-resource-ripe-atlas/bridge-metadata.json
var bridgeMetadata []byte

func Provider() tfbridge.ProviderInfo {
	return tfbridge.ProviderInfo{
		P:                pf.ShimProvider(tfp.New()),
		Name:             "ripe-atlas",
		Version:          version.Version,
		DisplayName:      "RIPE Atlas",
		Publisher:        "supabase",
		GitHubOrg:        "supabase",
		ResourcePrefix: "ripeatlas",
		Golang: &tfbridge.GolangInfo{
			ImportBasePath: "github.com/supabase/pulumi-atlas/sdk/go/ripeatlas",
			ModulePath:     "github.com/supabase/pulumi-atlas/sdk/go/ripeatlas",
		},
		JavaScript: &tfbridge.JavaScriptInfo{
			PackageName: "@supabase/ripe-atlas",
		},
		Config: map[string]*tfbridge.SchemaInfo{
			"namespace": {
				Default: &tfbridge.DefaultInfo{
					Value: "pulumi-atlas",
				},
			},
		},
		Resources: map[string]*tfbridge.ResourceInfo{
			"ripeatlas_measurement": {
				Tok: tfbridge.MakeResource("ripe-atlas", "index", "Measurement"),
			},
		},
		MetadataInfo: tfbridge.NewProviderMetadata(bridgeMetadata),
	}
}
