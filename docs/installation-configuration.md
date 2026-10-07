---
title: RIPE Atlas Installation & Configuration
meta_desc: Information on how to install and configure the RIPE Atlas Pulumi provider.
layout: package
---

## Installation

The RIPE Atlas provider is available as a package in all Pulumi languages:

* JavaScript / TypeScript: [`@supabase/ripe-atlas`](https://www.npmjs.com/package/@supabase/ripe-atlas)
* Python: [`pulumi-ripe-atlas`](https://pypi.org/project/pulumi-ripe-atlas/)
* Go: [`github.com/supabase/pulumi-atlas/sdk/go/ripeatlas`](https://pkg.go.dev/github.com/supabase/pulumi-atlas/sdk/go/ripeatlas)
* .NET: [`Supabase.RipeAtlas`](https://www.nuget.org/packages/Supabase.RipeAtlas)

## Provider binary

The provider binary must be present in `~/.pulumi/plugins/` for Pulumi to find it. Install it with:

```bash
pulumi plugin install resource ripe-atlas --server github://api.github.com/supabase/pulumi-atlas
```

Or build and install locally:

```bash
make install
```

## Configuration

The following configuration attributes are supported on the `Provider` resource. Sensitive values can also be set via environment variables.

| Attribute | Env var | Required | Description |
|-----------|---------|----------|-------------|
| `apiKey` | `RIPE_ATLAS_API_KEY` | yes | RIPE Atlas API key. Marked sensitive. |
| `snapshot` | `RIPE_ATLAS_SNAPSHOT` | one of snapshot or snapshotCachePath | Path to a pre-fetched probe snapshot JSON file. Probes are read directly from this file with no freshness check. |
| `snapshotCachePath` | — | one of snapshot or snapshotCachePath | Path for the auto-managed probe cache file. Probes are refreshed from the RIPE Atlas API when stale. Defaults to `/tmp/atlasctl-probes.json`. |
| `snapshotTtl` | — | no | Maximum age of the probe cache before refresh. Go duration string, e.g. `"2h"`. Defaults to `2h`. |
| `namespace` | — | no | Prefix embedded in each measurement's RIPE Atlas tags to distinguish measurements across stacks. Defaults to `pulumi-atlas`. |

### Setting configuration values

{{< chooser language "typescript,python,go,csharp" >}}

{{% choosable language typescript %}}

```typescript
const provider = new ripeAtlas.Provider("ripe-atlas", {
    apiKey:            process.env.RIPE_ATLAS_API_KEY,
    snapshotCachePath: "/var/cache/atlas-probes.json",
    snapshotTtl:       "2h",
    namespace:         "my-stack",
});
```

{{% /choosable %}}

{{% choosable language python %}}

```python
provider = ripe_atlas.Provider("ripe-atlas",
    api_key=os.environ["RIPE_ATLAS_API_KEY"],
    snapshot_cache_path="/var/cache/atlas-probes.json",
    snapshot_ttl="2h",
    namespace="my-stack",
)
```

{{% /choosable %}}

{{% choosable language go %}}

```go
provider, err := ripeatlas.NewProvider(ctx, "ripe-atlas", &ripeatlas.ProviderArgs{
    ApiKey:            pulumi.String(os.Getenv("RIPE_ATLAS_API_KEY")),
    SnapshotCachePath: pulumi.String("/var/cache/atlas-probes.json"),
    SnapshotTtl:       pulumi.String("2h"),
    Namespace:         pulumi.String("my-stack"),
})
```

{{% /choosable %}}

{{% choosable language csharp %}}

```csharp
var provider = new RipeAtlas.Provider("ripe-atlas", new RipeAtlas.ProviderArgs
{
    ApiKey            = System.Environment.GetEnvironmentVariable("RIPE_ATLAS_API_KEY"),
    SnapshotCachePath = "/var/cache/atlas-probes.json",
    SnapshotTtl       = "2h",
    Namespace         = "my-stack",
});
```

{{% /choosable %}}

{{< /chooser >}}

### Avoid the default provider

Always pass the provider instance explicitly to every resource. If omitted, Pulumi silently uses the default provider, which reads only from stack config variables and ignores constructor arguments.

To make this a hard error, add the following to your `Pulumi.<stack>.yaml`:

```yaml
config:
  pulumi:disable-default-providers:
    - ripeatlas
```

### Probe snapshot

The provider needs a probe list to run selection at plan time. Two modes are supported:

**Managed cache (recommended).** Set `snapshotCachePath` to a writable file path. The provider fetches probes from the RIPE Atlas API on first use and refreshes automatically when the cache is older than `snapshotTtl`.

**Pre-fetched file.** Set `snapshot` (or `RIPE_ATLAS_SNAPSHOT`) to a snapshot file produced by `atlasctl refresh`. The file is read as-is with no freshness check.
