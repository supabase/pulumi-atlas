package main

import (
	"github.com/pulumi/pulumi-terraform-bridge/v3/pkg/pf/tfgen"
	provider "github.com/supabase/pulumi-atlas/provider"
)

func main() {
	tfgen.Main("ripe-atlas", provider.Provider())
}
