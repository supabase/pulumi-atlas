package main

import (
	"context"
	"os"

	p "github.com/pulumi/pulumi-go-provider"

	"github.com/supabase/pulumi-atlas/provider"
)

func main() {
	err := p.RunProvider(context.Background(), "atlas", provider.Version, provider.NewProvider())
	if err != nil {
		p.GetLogger(context.Background()).Error(err.Error())
		os.Exit(1)
	}
}
