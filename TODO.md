# TODO: pre-publication blockers and follow-ups

Items that cannot be completed until the relevant repos are public, plus housekeeping
that should happen at publication time.

---

## go.work cleanup at publication time

`go.work` currently resolves `atlasctl` and `terraform-provider-ripe-atlas` from local
paths. When both modules are published, remove their entries from `go.work`:

```
# Remove these two use entries:
use ../atlasctl
use ../terraform-provider-ripe-atlas

# Remove these two replace directives:
replace github.com/supabase/atlasctl v0.1.2 => ../atlasctl
replace github.com/supabase/terraform-provider-ripe-atlas v0.1.2 => ../terraform-provider-ripe-atlas
```

`go.mod` already requires both at `v0.1.2` with no local replace. Once published at that
tag the build resolves them from the module proxy without go.work.

Also remove the two commented-out replace lines at the bottom of `go.mod` (cosmetic cleanup):

```go
// replace github.com/supabase/terraform-provider-ripe-atlas => ../terraform-provider-ripe-atlas
// replace github.com/supabase/atlasctl => ../atlasctl
```

Keep permanently (Pulumi bridge requirement, not a local path):

```
replace github.com/hashicorp/terraform-plugin-sdk/v2 => github.com/pulumi/terraform-plugin-sdk/v2 ...
```

### terraform-provider-ripe-atlas/go.work

Same situation: remove the `use` and `replace` entries for atlasctl once it is published.

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

## platform/pulumi/monitoring-ripe-atlas provider resolution

`Pulumi.yaml` currently declares the provider via `path: ../../../pulumi-atlas`, which only
works with both repos checked out side by side. The stack cannot run in platform CI until
this is changed.

Once `pulumi-atlas` has GitHub releases, update `Pulumi.yaml`:

```yaml
plugins:
  providers:
    - name: atlas
      version: X.Y.Z
      server: github://api.github.com/supabase/pulumi-atlas
```

Pulumi will then download `pulumi-resource-ripe-atlas` from the release assets. The shared
`_pulumi_workflow.yaml` already caches `~/.pulumi/plugins` keyed on `pnpm-lock.yaml`, so
the binary is cached across runs without any extra workflow changes.

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

| Language | Registry   | Package name                                            | Status  |
|----------|------------|---------------------------------------------------------|---------|
| Go       | pkg.go.dev | `github.com/supabase/pulumi-atlas/sdk/go/ripe-atlas`    | not set |
| Node.js  | npm        | `@supabase/ripe-atlas`                                  | done    |
| Python   | PyPI       | `supabase-pulumi-ripe-atlas`                            | not set |
| .NET     | NuGet      | `Supabase.Pulumi.RipeAtlas`                             | not set |

Set remaining names in `provider/resources.go` under the `Python` and `CSharp` fields of
`ProviderInfo` before the first published release. Changing them later is a breaking change.

---

## npm publish for the Node.js SDK

### Current state (pre-publication)

`platform/pulumi/ripe-atlas-sdk/` is a vendored copy of `sdk/nodejs/` checked into the
platform repo. `monitoring-ripe-atlas` depends on it via `"@supabase/ripe-atlas": "workspace:*"`.

To sync after a schema change: run `make sync-sdk` in this repo. That re-runs `make generate`
and rsyncs the result to `../platform/pulumi/ripe-atlas-sdk/`, patching the version to `0.0.1`
so pnpm accepts it.

### One-time setup for npm publishing

1. Done. `JavaScript.PackageName` is set to `@supabase/ripe-atlas` in `provider/resources.go`.
2. Create the `@supabase/ripe-atlas` package on npmjs.com under the `@supabase` org scope.
3. Configure OIDC trusted publishing on npmjs.com for this repo (Granular Access Token or
   Provenance). This avoids a static `NPM_TOKEN` secret that needs rotation.

### Release workflow addition

Add a step to `.github/workflows/release.yml` after GoReleaser completes:

```yaml
- uses: actions/setup-node@v4
  with:
    node-version: '20'
    registry-url: 'https://registry.npmjs.org'

- name: Publish Node.js SDK
  working-directory: sdk/nodejs
  run: |
    VERSION="${{ github.ref_name }}"
    VERSION="${VERSION#v}"
    sed -i "s/\${VERSION}/$VERSION/" package.json
    npm install
    npm run build
    npm publish --access public --provenance
  env:
    NODE_AUTH_TOKEN: ${{ secrets.NPM_TOKEN }}
```

The `sed` substitutes the git tag version into the `${VERSION}` placeholder that `make
generate` writes into `package.json`. This mirrors what GoReleaser does for the Go binary
via ldflags.

### Migration from workspace to npm (do after first published release)

Once `@supabase/ripe-atlas` is live on npm:

1. In `platform/pulumi/pnpm-workspace.yaml`, add to the catalog:
   ```yaml
   "@supabase/ripe-atlas": "^X.Y.Z"
   ```
2. In `platform/pulumi/monitoring-ripe-atlas/package.json`, change:
   ```json
   "@supabase/ripe-atlas": "workspace:*"
   ```
   to:
   ```json
   "@supabase/ripe-atlas": "catalog:"
   ```
3. Delete `platform/pulumi/ripe-atlas-sdk/` and run `pnpm install`.
4. The `sync-sdk` Makefile target can be removed or kept for local development against
   unreleased schema changes.

---

## Version

Done. `provider/version/version.go` holds the `Version` variable; `provider/resources.go`
reads it; `.goreleaser.yml` injects it at build time via ldflags. In local dev builds the
version defaults to `0.0.1`.
