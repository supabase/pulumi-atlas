package provider

import (
	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// Version is set at build time via ldflags.
var Version string

// NewProvider returns a configured pulumi-go-provider for atlas.
func NewProvider() p.Provider {
	return infer.Provider(infer.Options{
		Config: infer.Config(&AtlasConfig{}),
		Resources: []infer.InferredResource{
			infer.Resource[*Measurement, MeasurementArgs, MeasurementState](&Measurement{}),
			infer.Resource[*ProbeSelection, ProbeSelectionArgs, ProbeSelectionState](&ProbeSelection{}),
		},
	})
}
