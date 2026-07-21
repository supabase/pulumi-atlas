# pulumi-atlas

A Pulumi provider for [RIPE Atlas](https://atlas.ripe.net) measurements. It exposes the same resources as [terraform-provider-ripe-atlas](https://github.com/supabase/terraform-provider-ripe-atlas) through the Pulumi resource model, in Go, TypeScript, Python, and .NET.

## How this provider is built

This repo contains no resource logic. All CRUD operations, probe selection, drift detection, and API access live in two upstream packages:

- [atlasctl](https://github.com/supabase/atlasctl): the domain library. Probe scoring, selection, the RIPE Atlas API client, and the measurement lifecycle all live here.
- [terraform-provider-ripe-atlas](https://github.com/supabase/terraform-provider-ripe-atlas): the Terraform provider. It implements `ripeatlas_measurement` and `ripeatlas_probe_selection` using the atlasctl library.

This repo is a bridge layer using [pulumi-terraform-bridge](https://github.com/pulumi/pulumi-terraform-bridge). The bridge works in two phases:

**Build time.** The `pulumi-tfgen-ripe-atlas` binary imports the Terraform provider as a Go library, reads its schema (resources, data sources, attribute types, validation rules), and generates typed Pulumi SDKs for each target language into `sdk/`. The only configuration required is `provider/resources.go`, which maps Terraform resource names to Pulumi token paths and sets provider metadata.

**Runtime.** The `pulumi-resource-ripe-atlas` binary embeds the generated schema and runs the Terraform provider in-process via the Pulumi Terraform bridge. When `pulumi up` runs, Pulumi calls the bridge, which translates each lifecycle call (preview, create, read, update, delete) into the equivalent Terraform plugin protocol call, executed against the provider implementation in `terraform-provider-ripe-atlas`.

The only file in this repo that requires editing when resources change upstream is `provider/resources.go`.

## Resources

### `Measurement` (resource)

One resource per RIPE Atlas measurement. Each maps to a single `(name, cohort)` pair and one measurement ID on the RIPE Atlas platform.

Structural attributes (`name`, `cohort`, `target`, `msmType`, `af`, `intervalSeconds`) are immutable. Changing any of them stops the old measurement and creates a new one. `probeIds` is mutable in place: adding or removing probe IDs calls `AddParticipants` or `RemoveParticipants` on the running measurement without recreating it.

**Inputs**

| Attribute | Type | Description |
|-----------|------|-------------|
| `name` | string | Logical measurement name. Immutable. |
| `cohort` | string | Probe group name. Immutable. |
| `target` | string | DNS name or IP address. Immutable. |
| `msmType` | string | `dns`, `ping`, `tls`, or `traceroute`. Immutable. |
| `af` | int | Address family: `4` or `6`. Default `4`. Immutable. |
| `intervalSeconds` | int | Measurement interval in seconds (minimum 60). Immutable. |
| `probeIds` | list(int) | RIPE Atlas probe IDs. Mutable in place. |

**Outputs**

| Attribute | Type | Description |
|-----------|------|-------------|
| `msmId` | int | RIPE Atlas measurement ID assigned at creation. |

### `getProbeSelection` (data source / function)

Runs the atlasctl probe selection algorithm and returns probe IDs grouped by cohort. Cohort definitions, scoring weights, exclude tags, and geographic diversity are read from an `atlasctl.yaml` config file.

**Inputs**

| Attribute | Type | Description |
|-----------|------|-------------|
| `snapshot` | string | Path to a local `snapshot.json` produced by `atlasctl refresh`. |
| `config` | string | Path to `atlasctl.yaml`. |

**Outputs**

| Attribute | Type | Description |
|-----------|------|-------------|
| `probeIds` | map(list(int)) | Probe IDs per cohort name. |

## Usage (TypeScript)

For Go, Python, and TypeScript examples see [docs/snippets.md](docs/snippets.md).

Run `atlasctl refresh` to update your local probe snapshot before previewing, then use
`getProbeSelectionOutput` to feed probe IDs directly into `Measurement` resources.

```typescript
import * as pulumi from "@pulumi/pulumi";
import * as ripeAtlas from "@supabase/ripe-atlas";

// Run probe selection locally during `pulumi preview`.
// Update snapshot.json beforehand with: atlasctl refresh
const selected = ripeAtlas.getProbeSelectionOutput({
    snapshot: "./snapshot.json",
    config:   "./atlasctl.yaml",
});

// High-frequency DNS canary: 30 probes, every 60s
const dnsCanaryHigh = new ripeAtlas.Measurement("dns-canary-high", {
    name:            "dns-canary",
    cohort:          "high-freq",
    target:          "canary.supabase.co",
    msmType:         "dns",
    intervalSeconds: 60,
    probeIds:        selected.apply(s => s.probeIds["high-freq"]),
});

// Low-frequency DNS canary: 100 probes, every 15m
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

Use `getProbeSelectionOutput` (the `Output`-returning variant) rather than
`getProbeSelection` when passing results directly into resource arguments. `msmId` is
a computed output, available after `pulumi up` but not during preview.

## Provider configuration

Set `RIPE_ATLAS_API_KEY` in the environment, or pass `apiKey` explicitly. The key is marked sensitive and never appears in plain text in Pulumi state.

Required API key permissions are documented in the [atlasctl README](https://github.com/supabase/atlasctl#required-api-key-permissions).

## Credit costs

| Type | Credits per result |
|------|--------------------|
| dns | 10 |
| tls | 10 |
| ping | 3 |
| traceroute | 30 |

Total hourly cost per measurement: `(3600 / intervalSeconds) * creditsPerResult * len(probeIds)`.

Full background on RIPE Atlas, probe selection, and credit accounting is in the [atlasctl README](https://github.com/supabase/atlasctl/blob/main/README.md).

## Building

Requires Go 1.26 or later.

```bash
make build      # compile both binaries (tfgen and resource provider)
make generate   # regenerate SDKs from the current provider schema
make install    # install the provider binary into ~/.pulumi/plugins for local use
```

The typical workflow when the upstream Terraform provider schema changes:

```bash
make build      # rebuild tfgen with updated provider import
make generate   # regenerate schema.json, bridge-metadata.json, and all SDKs
make build      # rebuild the resource binary with the new schema embedded
```

`make generate` is not needed for runtime changes that do not touch the schema (for example, bug fixes inside the Terraform provider that leave resource attributes unchanged). A schema change is any addition, removal, or type change of a resource, data source, or attribute.

## Local development

To use a locally built provider with a Pulumi program, run `make install` and then point the program at the local binary:

```bash
make install
pulumi up   # Pulumi will find the installed binary automatically
```

Auth: set `RIPE_ATLAS_API_KEY` in the environment.

The `goat` CLI is useful for inspecting live measurements independently of Pulumi:

```bash
goat fm -my -status ong   # list running managed measurements
goat fp -asn4 7018 -status C -limit 20   # list connected AT&T probes
```

## Release

See `TODO.md` for the release infrastructure still to be wired up. Once it is in place, tagging a commit triggers the release workflow:

```bash
git tag v0.1.0 && git push origin v0.1.0
```
