# pulumi-atlas

A Pulumi provider for [RIPE Atlas](https://atlas.ripe.net) measurements. It exposes the same resources as [terraform-provider-ripe-atlas](https://github.com/supabase/terraform-provider-ripe-atlas) through the Pulumi resource model, in Go, TypeScript, Python, and .NET.

## How this provider is built

This repo contains no resource logic. All CRUD operations, probe selection, drift detection, and API access live in two upstream packages:

- [atlasctl](https://github.com/supabase/atlasctl): the domain library. Probe scoring, selection, the RIPE Atlas API client, and the measurement lifecycle all live here.
- [terraform-provider-ripe-atlas](https://github.com/supabase/terraform-provider-ripe-atlas): the Terraform provider. It implements `ripeatlas_measurement` using the atlasctl library.

This repo is a bridge layer using [pulumi-terraform-bridge](https://github.com/pulumi/pulumi-terraform-bridge). The bridge works in two phases:

**Build time.** The `pulumi-tfgen-ripe-atlas` binary imports the Terraform provider as a Go library, reads its schema (resources, attribute types, validation rules), and generates typed Pulumi SDKs for each target language into `sdk/`. The only configuration required is `provider/resources.go`, which maps Terraform resource names to Pulumi token paths and sets provider metadata.

**Runtime.** The `pulumi-resource-ripe-atlas` binary embeds the generated schema and runs the Terraform provider in-process via the Pulumi Terraform bridge. When `pulumi up` runs, Pulumi calls the bridge, which translates each lifecycle call (preview, create, read, update, delete) into the equivalent Terraform plugin protocol call, executed against the provider implementation in `terraform-provider-ripe-atlas`.

The only file in this repo that requires editing when resources change upstream is `provider/resources.go`.

## Resources

### `Measurement` (resource)

Manages a group of RIPE Atlas measurements sharing a common target and measurement type. Each cohort in the `cohorts` list creates one RIPE Atlas measurement ID from a distinct, non-overlapping slice of the probe pool. Cohorts are selected in declaration order.

Immutable attributes (`name`, `target`, `msmType`, `af`, `cohorts[*].name`, `cohorts[*].intervalSeconds`) trigger replacement on change. All other attributes are mutable in place: changes to scoring weights or probe counts re-run selection on the next plan, and the resulting diff drives `AddParticipants` or `RemoveParticipants` on the running measurements without recreating them.

**Inputs**

| Attribute | Type | Required | Immutable | Description |
|-----------|------|----------|-----------|-------------|
| `name` | string | yes | yes | Logical measurement name. |
| `target` | string | yes | yes | DNS name or IP address. |
| `msmType` | string | yes | yes | One of `dns`, `ping`, `tls`, `traceroute`. |
| `af` | int | no | yes | Address family: `4` or `6`. Default `4`. |
| `excludeTags` | list(string) | no | no | Probe tags that hard-exclude a probe from all cohort selection. |
| `cohorts` | list(object) | yes | partial | Ordered list of cohort configs (see below). |

**`cohorts[*]` fields**

| Attribute | Type | Required | Immutable | Description |
|-----------|------|----------|-----------|-------------|
| `name` | string | yes | yes | Cohort tier name, e.g. `high-freq`. |
| `probeCount` | int | yes | no | Number of probes to select. |
| `maxProbesPerCell` | int | yes | no | Maximum probes per H3 geographic cell. |
| `intervalSeconds` | int | yes | yes | Measurement interval in seconds. Minimum 60. |
| `includeProbeIds` | list(int) | no | no | Probes always included regardless of scoring or H3 cap. |
| `excludeProbeIds` | list(int) | no | no | Probes never selected in this cohort. |
| `cfg` | object | no | no | Additive scoring weights: `asn`, `tags`, `countries`, `stability`. |

**Per-cohort computed outputs** (accessed via `cohorts[*]`)

| Attribute | Type | Description |
|-----------|------|-------------|
| `msmId` | int | RIPE Atlas measurement ID assigned to this cohort. |
| `probeIds` | list(int) | Probe IDs selected for this cohort. |

## Provider configuration

| Attribute | Env var | Description |
|-----------|---------|-------------|
| `apiKey` | `RIPE_ATLAS_API_KEY` | RIPE Atlas API key. Marked sensitive. |
| `snapshot` | `RIPE_ATLAS_SNAPSHOT` | Path to `snapshot.json` produced by `atlasctl refresh`. |

The snapshot is configured once at the provider level and shared across all resources.

Required API key permissions are documented in the [atlasctl README](https://github.com/supabase/atlasctl#required-api-key-permissions).

## Usage (TypeScript)

For Go, Python, and TypeScript examples see [docs/snippets.md](docs/snippets.md).

```typescript
import * as ripeAtlas from "@supabase/ripe-atlas";

const provider = new ripeAtlas.Provider("ripe-atlas", {
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

export const highFreqMsmId = dnsCanary.cohorts.apply(cs => cs[0].msmId);
export const lowFreqMsmId  = dnsCanary.cohorts.apply(cs => cs[1].msmId);
```

`msmId` and `probeIds` are per-cohort computed outputs, available after `pulumi up`
but unknown during preview.

## Credit costs

| Type | Credits per result |
|------|--------------------|
| dns | 10 |
| tls | 10 |
| ping | 3 |
| traceroute | 30 |

Total hourly cost per cohort: `(3600 / intervalSeconds) * creditsPerResult * probeCount`.

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

`make generate` is not needed for runtime changes that do not touch the schema (for example, bug fixes inside the Terraform provider that leave resource attributes unchanged). A schema change is any addition, removal, or type change of a resource or attribute.

## Local development

To use a locally built provider with a Pulumi program, run `make install` and then point the program at the local binary:

```bash
make install
pulumi up   # Pulumi will find the installed binary automatically
```

Auth: set `RIPE_ATLAS_API_KEY` and `RIPE_ATLAS_SNAPSHOT` in the environment, or configure them via the provider block.

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
