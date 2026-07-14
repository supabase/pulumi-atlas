package provider

import (
	_ "embed"

	pf      "github.com/pulumi/pulumi-terraform-bridge/v3/pkg/pf/tfbridge"
	"github.com/pulumi/pulumi-terraform-bridge/v3/pkg/tfbridge"
	tfp     "github.com/supabase/terraform-provider-ripe-atlas/provider"
	"github.com/supabase/pulumi-atlas/provider/version"
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
		ResourcePrefix:   "ripeatlas",
		UpstreamRepoPath: "../terraform-provider-ripe-atlas",
		Resources: map[string]*tfbridge.ResourceInfo{
			"ripeatlas_measurement": {
				Tok: tfbridge.MakeResource("ripe-atlas", "index", "Measurement"),
			},
		},
		DataSources: map[string]*tfbridge.DataSourceInfo{
			"ripeatlas_probe_selection": {
				Tok: tfbridge.MakeDataSource("ripe-atlas", "index", "getProbeSelection"),
			},
		},
		MetadataInfo: tfbridge.NewProviderMetadata(bridgeMetadata),
	}
}
