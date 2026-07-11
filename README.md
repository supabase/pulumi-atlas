# pulumi-atlas

A Pulumi provider for [RIPE Atlas](https://atlas.ripe.net) measurements. It wraps the [atlasctl](../atlasctl) domain library to bring RIPE Atlas measurement management into the Pulumi resource model: declare your measurements and probe selections in code, preview changes before applying, and let Pulumi track state.

## Resources

### `atlas:index:Measurement`

One Pulumi resource per RIPE Atlas measurement. Each maps to a single `(name, round)` pair and one measurement ID on the RIPE Atlas platform.

```typescript
import * as atlas from "@pulumi/atlas";

const dnsHigh = new atlas.Measurement("dns-canary-high", {
    name:            "dns-canary",
    round:           "high-freq",
    target:          "canary.supabase.co",
    type:            "dns",
    intervalSeconds: 60,
    probeIds:        [1001, 2002, 3003],
});

export const msmId = dnsHigh.msmId;
```

**Inputs**

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Logical measurement name. Immutable. |
| `round` | string | Frequency tier name. Immutable. |
| `target` | string | DNS name or IP address. Immutable. |
| `type` | string | `dns`, `ping`, `tls`, or `traceroute`. Immutable. |
| `af` | int | Address family: `4` or `6`. Default `4`. Immutable. |
| `intervalSeconds` | int | Measurement interval in seconds (minimum 60). Immutable. |
| `probeIds` | []int | RIPE Atlas probe IDs. Mutable in place. |

Changing any immutable field stops the old measurement and creates a new one. Changing `probeIds` adds or removes participants on the running measurement without recreating it.

**Outputs**

| Field | Type | Description |
|-------|------|-------------|
| `msmId` | string | RIPE Atlas measurement ID assigned at creation. |

---

### `atlas:index:ProbeSelection`

Runs the atlasctl probe selection algorithm and stores the results in Pulumi state. Round definitions, scoring weights, exclude tags, and geographic diversity are all read from an `atlasctl.yaml` config file — no duplication in the Pulumi program.

```typescript
import * as atlas from "@pulumi/atlas";

const selection = new atlas.ProbeSelection("probe-selection", {
    configPath: "./atlasctl.yaml",
    // snapshotPath: "./probes/snapshot.json"  // overrides the path in atlasctl.yaml if needed
});

const dnsHigh = new atlas.Measurement("dns-canary-high", {
    name:            "dns-canary",
    round:           "high-freq",
    target:          "canary.supabase.co",
    type:            "dns",
    intervalSeconds: 60,
    probeIds:        selection.roundProbeIds["high-freq"],
});
```

On every `pulumi up`, the provider hashes both `atlasctl.yaml` and the probe snapshot. If either has changed, selection re-runs and updated probe IDs propagate automatically to dependent `Measurement` resources.

**Inputs**

| Field | Type | Description |
|-------|------|-------------|
| `configPath` | string | Path to `atlasctl.yaml`. |
| `snapshotPath` | string | Optional. Overrides the snapshot path in the config. |

**Outputs**

| Field | Type | Description |
|-------|------|-------------|
| `roundProbeIds` | map[string][]int | Probe IDs per round name. |
| `snapshotHash` | string | SHA-256 of the snapshot at last selection. |
| `configHash` | string | SHA-256 of the config at last selection. |
| `selectedAt` | string | RFC3339 timestamp of the last selection run. |

## Provider configuration

```typescript
const provider = new atlas.Provider("atlas", {
    apiKey: process.env.RIPE_ATLAS_API_KEY,
    tagPrefix: "[atlasctl:",  // optional, default shown
});
```

`apiKey` is marked secret and never appears in plain text in Pulumi state.

## Credit costs

| Type | Credits per result |
|------|--------------------|
| dns | 10 |
| tls | 10 |
| ping | 3 |
| traceroute | 30 |

Minimum measurement interval is 60 seconds. All measurements created by this provider are periodic.

## Building

```bash
make build    # build the provider binary
make schema   # regenerate schema.json (commit the result)
make sdk      # generate language SDKs (not committed)
make test     # run tests
```

Requires Go 1.26 or later and the Pulumi CLI.

## Release

Tag a commit to trigger GoReleaser:

```bash
git tag v0.1.0 && git push origin v0.1.0
```

## Requirements

- Go 1.26+
- Pulumi CLI
- `RIPE_ATLAS_API_KEY` set to a valid RIPE Atlas API key
