package main

import (
	"context"
	_ "embed"

	pfbridge "github.com/pulumi/pulumi-terraform-bridge/v3/pkg/pf/tfbridge"
	provider  "github.com/supabase/pulumi-atlas/provider"
)

//go:embed schema.json
var schema []byte

func main() {
	meta := pfbridge.ProviderMetadata{PackageSchema: schema}
	pfbridge.Main(context.Background(), "ripe-atlas", provider.Provider(), meta)
}
