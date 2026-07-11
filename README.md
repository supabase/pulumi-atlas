# pulumi-atlas

A Pulumi provider for managing RIPE Atlas measurements declaratively. Define measurement targets and probe sets in a Pulumi program, and the provider handles the full lifecycle: create, update probe sets in place, and stop measurements on destroy.

Built for Supabase's external edge telemetry project. See [atlasctl](../atlasctl) for the underlying domain library and the CLI workflow that generates probe selections.

## How it fits together

```
atlasctl select   ---->   probe IDs per round
                                  |
                                  v
              pulumi up  (this provider)
                                  |
                                  v
                       RIPE Atlas measurement IDs
                                  |
                                  v
                        atlas_exporter  -->  Prometheus
```

`atlasctl select` scores and ranks the RIPE Atlas probe pool and outputs probe ID lists. Those lists are the input to this provider. The provider owns the measurement lifecycle. `atlas_exporter` subscribes to the resulting measurement IDs via the RIPE Atlas streaming WebSocket and exposes results as Prometheus metrics.

## Resource

One resource: `atlasctl:index:Measurement`

Each resource maps to one `(name, round)` pair and one RIPE Atlas measurement ID.

```typescript
import * as atlas from "@pulumi/atlas";

const canary = new atlas.Measurement("dns-canary-high", {
    name: "dns-canary",
    round: "high-freq",
    target: "canary.supabase.co",
    type: "dns",
    af: 4,
    intervalSeconds: 60,
    probeIds: [1001, 2002, 3003, 4004, 5005],
});

export const msmId = canary.msmId;
```

### ProbeSelection

```typescript
import * as atlas from "@pulumi/atlas";

// Round definitions, scoring weights, excludeTags, and geoDiversity all come
// from atlasctl.yaml — no duplication in the Pulumi program.
const selection = new atlas.ProbeSelection("probe-selection", {
    configPath:   "./atlasctl.yaml",
    // snapshotPath: "./probes/snapshot.json"  // optional override; defaults to config snapshot field
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

`ProbeSelection` hashes both `atlasctl.yaml` and the snapshot file on every `pulumi up`. Editing the config or running `atlasctl refresh` triggers re-selection automatically, propagating updated probe IDs to all dependent `Measurement` resources.

### Inputs

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Logical measurement name. Immutable. |
| `round` | string | Frequency tier (e.g. `high-freq`). Immutable. |
| `target` | string | DNS name or IP address. Immutable. |
| `type` | string | `dns`, `ping`, `tls`, or `traceroute`. Immutable. |
| `af` | int | Address family: `4` or `6`. Default `4`. Immutable. |
| `intervalSeconds` | int | Measurement interval in seconds (minimum 60). Immutable. |
| `probeIds` | []int | RIPE Atlas probe IDs. Mutable in place. |

Changing any immutable field stops the old measurement and creates a new one. Changing `probeIds` adds or removes participants without recreating the measurement.

### Outputs

| Field | Type | Description |
|-------|------|-------------|
| `msmId` | string | RIPE Atlas measurement ID assigned at creation. |

## Provider configuration

```typescript
const provider = new atlas.Provider("atlas", {
    apiKey: process.env.RIPE_ATLAS_API_KEY,
    tagPrefix: "[atlasctl:",  // optional, default shown
});
```

`apiKey` is marked secret and never appears in plain text in Pulumi state.

## Measurement types and credit costs

| Type | Credits per result | Notes |
|------|--------------------|-------|
| dns | 10 | UDP by default |
| tls | 10 | TCP handshake + certificate check |
| ping | 3 | ICMP |
| traceroute | 30 | |

One-off measurements cost 2x the periodic rate. All measurements created by this provider are periodic. Minimum interval is 60 seconds.

## Building

```bash
# Build the provider binary
make build

# Regenerate schema.json (commit the result)
make schema

# Generate language SDKs (not committed, generated in CI or locally)
make sdk

# Run tests
make test
```

Requires Go 1.26 or later and the Pulumi CLI.

## Release

Tag a commit with a semver tag to trigger GoReleaser:

```bash
git tag v0.1.0
git push origin v0.1.0
```

The release workflow builds binaries for Linux, macOS, and Windows (amd64 and arm64) and attaches them to the GitHub release.

## Requirements

- `RIPE_ATLAS_API_KEY` environment variable set to a valid RIPE Atlas API key UUID
- A RIPE Atlas account with sufficient credits for the measurements you intend to create
- Pulumi CLI
