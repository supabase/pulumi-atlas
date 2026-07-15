# SDK snippets

All examples declare one `Measurement` resource with two cohorts: `high-freq` (30 probes,
every 60 s) and `low-freq` (100 probes, every 15 min). Both cohorts are managed by the
same resource; probe selection runs in declaration order, with each cohort drawing from
the remaining pool after earlier cohorts have claimed their probes.

The snapshot path is set once at the provider level. Run `atlasctl refresh` to update it
before running `pulumi preview`.

---

## TypeScript

```typescript
import * as pulumi from "@pulumi/pulumi";
import * as ripeAtlas from "@supabase/ripe-atlas";

const provider = new ripeAtlas.Provider("ripe-atlas", {
    apiKey:   process.env.RIPE_ATLAS_API_KEY,
    snapshot: "./snapshot.json",
});

const dnsCanary = new ripeAtlas.Measurement("dns-canary", {
    name:        "dns-canary",
    target:      "canary.supabase.co",
    msmType:     "dns",
    excludeTags: ["broken", "system-flakey-connection"],
    cohorts: [
        {
            name:             "high-freq",
            probeCount:       30,
            maxProbesPerCell: 1,
            intervalSeconds:  60,
            cfg: {
                asn:       { "7018": 10, "7922": 8 },
                stability: { "system-ipv4-stable-90d": 5 },
            },
        },
        {
            name:             "low-freq",
            probeCount:       100,
            maxProbesPerCell: 3,
            intervalSeconds:  900,
        },
    ],
}, { provider });

// msmId and probeIds are computed per cohort, available after pulumi up.
export const highFreqMsmId    = dnsCanary.cohorts.apply(cs => cs[0].msmId);
export const highFreqProbeIds = dnsCanary.cohorts.apply(cs => cs[0].probeIds);
export const lowFreqMsmId     = dnsCanary.cohorts.apply(cs => cs[1].msmId);
```

---

## Python

```python
import os
import pulumi
import pulumi_ripe_atlas as ripe_atlas

provider = ripe_atlas.Provider("ripe-atlas",
    api_key=os.environ["RIPE_ATLAS_API_KEY"],
    snapshot="./snapshot.json",
)

dns_canary = ripe_atlas.Measurement("dns-canary",
    name="dns-canary",
    target="canary.supabase.co",
    msm_type="dns",
    exclude_tags=["broken", "system-flakey-connection"],
    cohorts=[
        ripe_atlas.MeasurementCohortArgs(
            name="high-freq",
            probe_count=30,
            max_probes_per_cell=1,
            interval_seconds=60,
            cfg=ripe_atlas.MeasurementCohortCfgArgs(
                asn={"7018": 10, "7922": 8},
                stability={"system-ipv4-stable-90d": 5},
            ),
        ),
        ripe_atlas.MeasurementCohortArgs(
            name="low-freq",
            probe_count=100,
            max_probes_per_cell=3,
            interval_seconds=900,
        ),
    ],
    opts=pulumi.ResourceOptions(provider=provider),
)

pulumi.export("high_freq_msm_id",    dns_canary.cohorts[0].msm_id)
pulumi.export("high_freq_probe_ids", dns_canary.cohorts[0].probe_ids)
pulumi.export("low_freq_msm_id",     dns_canary.cohorts[1].msm_id)
```

---

## Go

```go
package main

import (
    "github.com/pulumi/pulumi/sdk/v3/go/pulumi"
    ripeatlas "github.com/supabase/pulumi-atlas/sdk/go/ripeatlas"
)

func main() {
    pulumi.Run(func(ctx *pulumi.Context) error {
        provider, err := ripeatlas.NewProvider(ctx, "ripe-atlas", &ripeatlas.ProviderArgs{
            ApiKey:   pulumi.String(pulumi.String(ctx.MustGetConfig("apiKey"))),
            Snapshot: pulumi.String("./snapshot.json"),
        })
        if err != nil {
            return err
        }

        dnsCanary, err := ripeatlas.NewMeasurement(ctx, "dns-canary", &ripeatlas.MeasurementArgs{
            Name:        pulumi.String("dns-canary"),
            Target:      pulumi.String("canary.supabase.co"),
            MsmType:     pulumi.String("dns"),
            ExcludeTags: pulumi.StringArray{pulumi.String("broken"), pulumi.String("system-flakey-connection")},
            Cohorts: ripeatlas.MeasurementCohortArray{
                ripeatlas.MeasurementCohortArgs{
                    Name:             pulumi.String("high-freq"),
                    ProbeCount:       pulumi.Int(30),
                    MaxProbesPerCell: pulumi.Int(1),
                    IntervalSeconds:  pulumi.Int(60),
                    Cfg: ripeatlas.MeasurementCohortCfgArgs{
                        Asn:       pulumi.IntMap{"7018": pulumi.Int(10), "7922": pulumi.Int(8)},
                        Stability: pulumi.IntMap{"system-ipv4-stable-90d": pulumi.Int(5)},
                    }.ToMeasurementCohortCfgPtrOutput(),
                },
                ripeatlas.MeasurementCohortArgs{
                    Name:             pulumi.String("low-freq"),
                    ProbeCount:       pulumi.Int(100),
                    MaxProbesPerCell: pulumi.Int(3),
                    IntervalSeconds:  pulumi.Int(900),
                },
            },
        }, pulumi.ProviderID(provider.ID()))
        if err != nil {
            return err
        }

        ctx.Export("highFreqMsmId", dnsCanary.Cohorts.Index(pulumi.Int(0)).MsmId())
        ctx.Export("lowFreqMsmId",  dnsCanary.Cohorts.Index(pulumi.Int(1)).MsmId())
        return nil
    })
}
```

---

## Notes

**One resource, multiple RIPE Atlas measurements.** Each cohort element in the list
creates one RIPE Atlas measurement ID. Probe selection runs across all cohorts in order:
each cohort draws from the pool of probes not already claimed by earlier cohorts in the
same resource.

**Snapshot is provider-level.** Configure it once via the provider block or the
`RIPE_ATLAS_SNAPSHOT` environment variable. It does not appear on individual resources.

**`msmId` and `probeIds` are per-cohort computed outputs.** They are unknown during
`pulumi preview` and resolve after `pulumi up`. Access them by index on the `cohorts`
output array.

**Reusing cohort configs.** Because `cohorts` is a plain TypeScript array, cohort
objects can be defined as constants or factory functions and referenced across multiple
`Measurement` resources. Each resource still runs its own independent selection against
the full probe pool.
