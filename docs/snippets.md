# SDK snippets

All examples declare two DNS canary measurements sharing a single probe selection call:
`high-freq` (30 probes, every 60 s) and `low-freq` (100 probes, every 15 min). The probe
IDs come from `atlasctl.yaml` and a local `snapshot.json` updated by `atlasctl refresh`.

---

## TypeScript

```typescript
import * as pulumi from "@pulumi/pulumi";
import * as ripeAtlas from "@supabase/ripe-atlas";

// getProbeSelectionOutput returns an Output<T>, which can be passed directly
// into resource arguments without an explicit apply() call.
const selected = ripeAtlas.getProbeSelectionOutput({
    snapshot: "./snapshot.json",
    config:   "./atlasctl.yaml",
});

const dnsCanaryHigh = new ripeAtlas.Measurement("dns-canary-high", {
    name:            "dns-canary",
    cohort:          "high-freq",
    target:          "canary.supabase.co",
    msmType:         "dns",
    intervalSeconds: 60,
    probeIds:        selected.apply(s => s.probeIds["high-freq"]),
});

const dnsCanaryLow = new ripeAtlas.Measurement("dns-canary-low", {
    name:            "dns-canary",
    cohort:          "low-freq",
    target:          "canary.supabase.co",
    msmType:         "dns",
    intervalSeconds: 900,
    probeIds:        selected.apply(s => s.probeIds["low-freq"]),
});

export const highFreqMsmId = dnsCanaryHigh.msmId;
export const lowFreqMsmId  = dnsCanaryLow.msmId;
```

---

## Python

```python
import pulumi
import pulumi_ripe_atlas as ripe_atlas

# The synchronous variant returns a plain GetProbeSelectionResult, which is
# simpler when probe IDs are used only as resource inputs.
selected = ripe_atlas.get_probe_selection(
    snapshot="./snapshot.json",
    config="./atlasctl.yaml",
)

dns_canary_high = ripe_atlas.Measurement(
    "dns-canary-high",
    name="dns-canary",
    cohort="high-freq",
    target="canary.supabase.co",
    msm_type="dns",
    interval_seconds=60,
    probe_ids=selected.probe_ids["high-freq"],
)

dns_canary_low = ripe_atlas.Measurement(
    "dns-canary-low",
    name="dns-canary",
    cohort="low-freq",
    target="canary.supabase.co",
    msm_type="dns",
    interval_seconds=900,
    probe_ids=selected.probe_ids["low-freq"],
)

pulumi.export("high_freq_msm_id", dns_canary_high.msm_id)
pulumi.export("low_freq_msm_id",  dns_canary_low.msm_id)
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
        selected, err := ripeatlas.GetProbeSelection(ctx, &ripeatlas.GetProbeSelectionArgs{
            Snapshot: "./snapshot.json",
            Config:   "./atlasctl.yaml",
        }, nil)
        if err != nil {
            return err
        }

        // Convert []int to pulumi.IntArray for use as resource input.
        toIntArray := func(ids []int) pulumi.IntArray {
            out := make(pulumi.IntArray, len(ids))
            for i, id := range ids {
                out[i] = pulumi.Int(id)
            }
            return out
        }

        dnsCanaryHigh, err := ripeatlas.NewMeasurement(ctx, "dns-canary-high", &ripeatlas.MeasurementArgs{
            Name:            pulumi.String("dns-canary"),
            Cohort:          pulumi.String("high-freq"),
            Target:          pulumi.String("canary.supabase.co"),
            MsmType:         pulumi.String("dns"),
            IntervalSeconds: pulumi.Int(60),
            ProbeIds:        toIntArray(selected.ProbeIds["high-freq"]),
        })
        if err != nil {
            return err
        }

        dnsCanaryLow, err := ripeatlas.NewMeasurement(ctx, "dns-canary-low", &ripeatlas.MeasurementArgs{
            Name:            pulumi.String("dns-canary"),
            Cohort:          pulumi.String("low-freq"),
            Target:          pulumi.String("canary.supabase.co"),
            MsmType:         pulumi.String("dns"),
            IntervalSeconds: pulumi.Int(900),
            ProbeIds:        toIntArray(selected.ProbeIds["low-freq"]),
        })
        if err != nil {
            return err
        }

        ctx.Export("highFreqMsmId", dnsCanaryHigh.MsmId)
        ctx.Export("lowFreqMsmId",  dnsCanaryLow.MsmId)
        return nil
    })
}
```

---

## Notes

**Probe selection variant choice.** TypeScript uses `getProbeSelectionOutput` (returns
`Output<T>`) because TypeScript resource inputs are `Input<T>` and the bridge is smooth.
Python and Go use the synchronous variant because `probe_ids` is immediately available as
a plain value and the conversion to Pulumi input types is more readable that way.

**`msmId` availability.** The `msmId` output is assigned by the RIPE Atlas API at creation
time. It is unknown during `pulumi preview` and resolves after `pulumi up` completes.

**Snapshot freshness.** Run `atlasctl refresh` before `pulumi preview` whenever you want
probe selection to reflect the current connected probe pool. Committing `snapshot.json`
gives reproducible selection until you explicitly refresh.
