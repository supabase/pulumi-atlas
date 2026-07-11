package provider

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pulumi/pulumi-go-provider/infer"

	"github.com/supabase/atlasctl/pkg/atlasapi"
	"github.com/supabase/atlasctl/pkg/plan"
)

// AtlasConfig holds provider-level configuration.
type AtlasConfig struct {
	APIKey    string `pulumi:"apiKey"    provider:"secret"`
	TagPrefix string `pulumi:"tagPrefix,optional"`

	// client is built during Configure and threaded into resource methods.
	client plan.ApplyClient
}

var _ infer.CustomConfigure = (*AtlasConfig)(nil)

func (c *AtlasConfig) Configure(ctx context.Context) error {
	if c.APIKey == "" {
		return fmt.Errorf("apiKey is required")
	}
	if c.TagPrefix == "" {
		c.TagPrefix = "[atlasctl:"
	}

	id, err := uuid.Parse(c.APIKey)
	if err != nil {
		return fmt.Errorf("apiKey is not a valid UUID: %w", err)
	}

	codec := plan.NewTagCodec(c.TagPrefix)
	c.client = atlasapi.NewApplyClient(&id, false, codec)
	return nil
}
