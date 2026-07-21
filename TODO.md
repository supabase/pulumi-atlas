# TODO: pre-publication blockers and follow-ups

Items that cannot be completed until the relevant repos are public, plus housekeeping
that should happen at publication time.

---

## Replace directives to remove

### pulumi-atlas/go.mod

Remove when the listed module is published:

```
replace github.com/supabase/terraform-provider-ripe-atlas => ../terraform-provider-ripe-atlas
replace github.com/supabase/atlasctl => ../atlasctl
```

Keep permanently (Pulumi bridge requirement, not a local path):

```
replace github.com/hashicorp/terraform-plugin-sdk/v2 => github.com/pulumi/terraform-plugin-sdk/v2 ...
```

### terraform-provider-ripe-atlas/go.mod

Remove when atlasctl is published:

```
replace github.com/supabase/atlasctl => ../atlasctl
```

---

## provider/resources.go

Remove `UpstreamRepoPath` once the Terraform provider repo is public. The bridge will
then resolve docs by downloading the module via `go mod download`, using `GitHubOrg` and
`Name` to locate the repo:

```go
// Remove this line:
UpstreamRepoPath: "../terraform-provider-ripe-atlas",
```

---

## Publish order

The repos have a dependency chain. Publish in this order:

1. `atlasctl` — no external local deps
2. `terraform-provider-ripe-atlas` — depends on atlasctl
3. `pulumi-atlas` — depends on both

Publishing terraform-provider-ripe-atlas to the Terraform registry is a prerequisite for
the Pulumi provider's HCL example conversion to work during `make generate`.

---

## HCL example conversion

`make generate` currently warns that examples cannot be converted to Go/TypeScript/Python/etc.
because the Terraform provider is not yet in the registry. Once it is, rerun `make generate`
and verify the converted examples look correct in the generated SDK docs.

If example conversion is still undesirable after publication (e.g. the HCL examples use
local file paths that do not translate cleanly), suppress it per-resource with:

```go
"ripeatlas_measurement": {
    Tok:  tfbridge.MakeResource("ripe-atlas", "index", "Measurement"),
    Docs: &tfbridge.DocInfo{Source: "measurement.html.markdown"},
},
```

---

## Release infrastructure

Neither repo has release CI yet.

- `terraform-provider-ripe-atlas` has a `.goreleaser.yml` already. It needs a GitHub
  Actions workflow that triggers on tag push and calls GoReleaser, plus a
  `RELEASES_GITHUB_TOKEN` secret and Terraform Registry API key.

- `pulumi-atlas` has `.github/workflows/release.yml` and `.goreleaser.yml`. The release
  workflow builds and archives `pulumi-resource-ripe-atlas` for all platforms and attaches
  archives to the GitHub release. What is not yet wired: publishing SDK packages to
  npm/PyPI/NuGet. Add those as separate steps in the release workflow once SDK package
  names are decided (see below).

---

## SDK package names

Before publishing the generated SDKs, decide on package names for each language registry:

| Language | Registry | Package name (suggested) |
|----------|----------|--------------------------|
| Go       | pkg.go.dev | `github.com/supabase/pulumi-atlas/sdk/go/ripe-atlas` |
| Node.js  | npm | `@supabase/ripe-atlas` |
| Python   | PyPI | `supabase-pulumi-ripe-atlas` |
| .NET     | NuGet | `Supabase.Pulumi.RipeAtlas` |

Set these in `provider/resources.go` under `JavaScript`, `Python`, and `CSharp` fields of
`ProviderInfo` before the first published release, as changing them later is a breaking change.

---

## Version

Done. `provider/version/version.go` holds the `Version` variable; `provider/resources.go`
reads it; `.goreleaser.yml` injects it at build time via ldflags. In local dev builds the
version is an empty string.
