---
title: RIPE Atlas
meta_desc: Manage RIPE Atlas measurements from Pulumi. Select probe cohorts, track credit burn, and integrate active network monitoring into your infrastructure as code.
layout: package
---

The RIPE Atlas provider for Pulumi lets you declare and manage [RIPE Atlas](https://atlas.ripe.net) measurements as infrastructure. Probe selection, cohort drawdown, and participant updates are handled at plan time — `pulumi preview` shows which probes will be selected and the projected credit cost before anything is applied.

Each `Measurement` resource manages one or more cohorts. Each cohort draws probes from the remaining pool after earlier cohorts have claimed theirs, so a single resource can express a tiered monitoring strategy (high-frequency anchors, lower-frequency broad coverage) without probe overlap.

## Example

{{< chooser language "typescript,python,go,csharp" >}}

{{% choosable language typescript %}}

```typescript
import * as pulumi from "@pulumi/pulumi";
import * as ripeAtlas from "@supabase/ripe-atlas";

const provider = new ripeAtlas.Provider("ripe-atlas", {
    apiKey:   process.env.RIPE_ATLAS_API_KEY,
    snapshot: "./snapshot.json",
});

const dnsCanary = new ripeAtlas.Measurement("dns-canary", {
    name:    "dns-canary",
    target:  "canary.supabase.co",
    msmType: "dns",
    cohorts: [
        {
            name:             "high-freq",
            probeCount:       30,
            maxProbesPerCell: 1,
            intervalSeconds:  60,
            cfg: {
                asn:         { "7018": 10, "7922": 8 },
                stability:   { "system-ipv4-stable-90d": 5 },
                excludeTags: ["broken", "system-flakey-connection"],
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

export const highFreqMsmId     = dnsCanary.cohorts.apply(cs => cs[0].msmId);
export const totalDailyCredits = dnsCanary.totalDailyCredits;
```

{{% /choosable %}}

{{% choosable language python %}}

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
    cohorts=[
        ripe_atlas.MeasurementCohortArgs(
            name="high-freq",
            probe_count=30,
            max_probes_per_cell=1,
            interval_seconds=60,
            cfg=ripe_atlas.MeasurementCohortCfgArgs(
                asn={"7018": 10, "7922": 8},
                stability={"system-ipv4-stable-90d": 5},
                exclude_tags=["broken", "system-flakey-connection"],
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

pulumi.export("high_freq_msm_id",     dns_canary.cohorts[0].msm_id)
pulumi.export("total_daily_credits",  dns_canary.total_daily_credits)
```

{{% /choosable %}}

{{% choosable language go %}}

```go
package main

import (
    "github.com/pulumi/pulumi/sdk/v3/go/pulumi"
    ripeatlas "github.com/supabase/pulumi-atlas/sdk/go/ripeatlas"
)

func main() {
    pulumi.Run(func(ctx *pulumi.Context) error {
        provider, err := ripeatlas.NewProvider(ctx, "ripe-atlas", &ripeatlas.ProviderArgs{
            ApiKey:   pulumi.String(ctx.MustGetConfig("apiKey")),
            Snapshot: pulumi.String("./snapshot.json"),
        })
        if err != nil {
            return err
        }

        dnsCanary, err := ripeatlas.NewMeasurement(ctx, "dns-canary", &ripeatlas.MeasurementArgs{
            Name:    pulumi.String("dns-canary"),
            Target:  pulumi.String("canary.supabase.co"),
            MsmType: pulumi.String("dns"),
            Cohorts: ripeatlas.MeasurementCohortArray{
                ripeatlas.MeasurementCohortArgs{
                    Name:             pulumi.String("high-freq"),
                    ProbeCount:       pulumi.Int(30),
                    MaxProbesPerCell: pulumi.Int(1),
                    IntervalSeconds:  pulumi.Int(60),
                    Cfg: ripeatlas.MeasurementCohortCfgArgs{
                        Asn:         pulumi.IntMap{"7018": pulumi.Int(10), "7922": pulumi.Int(8)},
                        Stability:   pulumi.IntMap{"system-ipv4-stable-90d": pulumi.Int(5)},
                        ExcludeTags: pulumi.StringArray{pulumi.String("broken"), pulumi.String("system-flakey-connection")},
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

        ctx.Export("highFreqMsmId",     dnsCanary.Cohorts.Index(pulumi.Int(0)).MsmId())
        ctx.Export("totalDailyCredits", dnsCanary.TotalDailyCredits())
        return nil
    })
}
```

{{% /choosable %}}

{{% choosable language csharp %}}

```csharp
using System.Collections.Generic;
using Pulumi;
using RipeAtlas = Supabase.RipeAtlas;

return await Deployment.RunAsync(() =>
{
    var provider = new RipeAtlas.Provider("ripe-atlas", new RipeAtlas.ProviderArgs
    {
        ApiKey   = System.Environment.GetEnvironmentVariable("RIPE_ATLAS_API_KEY"),
        Snapshot = "./snapshot.json",
    });

    var dnsCanary = new RipeAtlas.Measurement("dns-canary", new RipeAtlas.MeasurementArgs
    {
        Name    = "dns-canary",
        Target  = "canary.supabase.co",
        MsmType = "dns",
        Cohorts =
        {
            new RipeAtlas.Inputs.MeasurementCohortArgs
            {
                Name             = "high-freq",
                ProbeCount       = 30,
                MaxProbesPerCell = 1,
                IntervalSeconds  = 60,
                Cfg = new RipeAtlas.Inputs.MeasurementCohortCfgArgs
                {
                    Asn       = { { "7018", 10 }, { "7922", 8 } },
                    Stability = { { "system-ipv4-stable-90d", 5 } },
                    ExcludeTags = { "broken", "system-flakey-connection" },
                },
            },
            new RipeAtlas.Inputs.MeasurementCohortArgs
            {
                Name             = "low-freq",
                ProbeCount       = 100,
                MaxProbesPerCell = 3,
                IntervalSeconds  = 900,
            },
        },
    }, new CustomResourceOptions { Provider = provider });

    return new Dictionary<string, object?>
    {
        ["highFreqMsmId"]     = dnsCanary.Cohorts.Apply(cs => cs[0].MsmId),
        ["totalDailyCredits"] = dnsCanary.TotalDailyCredits,
    };
});
```

{{% /choosable %}}

{{< /chooser >}}
