# TODO: post-publication follow-ups

All three repos (atlasctl, terraform-provider-ripe-atlas, pulumi-atlas) are now public.

---

## go.work cleanup (when cross-repo development is done)

`go.work` still has `use` and `replace` entries for the sibling repos — useful while
actively developing across all three. Remove them when you no longer need local edits to
resolve:

```
# Remove:
use ../atlasctl
use ../terraform-provider-ripe-atlas

replace github.com/supabase/atlasctl v0.1.2 => ../atlasctl
replace github.com/supabase/terraform-provider-ripe-atlas v0.1.2 => ../terraform-provider-ripe-atlas
```

`go.mod` already requires both at `v0.1.2` with no local replace, so once removed the
build resolves them from the module proxy.

### terraform-provider-ripe-atlas/go.work

Same: remove `use` and `replace` for atlasctl when done with cross-repo work.

---

## HCL example conversion

`make generate` warns that examples cannot be converted because the Terraform provider is
not yet in the **Terraform Registry** (GitHub-public is not sufficient — the registry
requires a separate submission). Once it is registered, rerun `make generate` and verify
the converted examples.

If example conversion is still undesirable after registration (e.g. HCL examples use local
file paths that do not translate cleanly), suppress it per-resource with:

```go
"ripeatlas_measurement": {
    Tok:  tfbridge.MakeResource("ripe-atlas", "index", "Measurement"),
    Docs: &tfbridge.DocInfo{Source: "measurement.html.markdown"},
},
```

---

## platform/pulumi/monitoring-ripe-atlas provider resolution

Done. `Pulumi.yaml` now pins `v0.1.0` from GitHub releases. Update the version here when
a new release is cut.

---

## Release infrastructure

Neither repo has release CI yet.

- `terraform-provider-ripe-atlas` has a `.goreleaser.yml` already. It needs a GitHub
  Actions workflow that triggers on tag push and calls GoReleaser, plus a
  `RELEASES_GITHUB_TOKEN` secret and Terraform Registry API key.

- `pulumi-atlas` has `.github/workflows/release.yml` and `.goreleaser.yml`. The release
  workflow builds and archives `pulumi-resource-ripe-atlas` for all platforms and attaches
  archives to the GitHub release. What is not yet wired: publishing SDK packages to
  npm/PyPI/NuGet.

---

## SDK package names

| Language | Registry   | Package name                                            | Status  |
|----------|------------|---------------------------------------------------------|---------|
| Go       | pkg.go.dev | `github.com/supabase/pulumi-atlas/sdk/go/ripe-atlas`    | not set |
| Node.js  | npm        | `@supabase/ripe-atlas`                                  | done    |
| Python   | PyPI       | `supabase-pulumi-ripe-atlas`                            | not set |
| .NET     | NuGet      | `Supabase.Pulumi.RipeAtlas`                             | not set |

Set remaining names in `provider/resources.go` under the `Python` and `CSharp` fields
before the first published release. Changing them later is a breaking change.

---

## npm publish for the Node.js SDK

### Current state

`platform/pulumi/ripe-atlas-sdk/` is a vendored copy of `sdk/nodejs/` checked into the
platform repo. `monitoring-ripe-atlas` depends on it via `"@supabase/ripe-atlas": "workspace:*"`.

To sync after a schema change: run `make sync-sdk` in this repo.

### One-time setup for npm publishing

1. Done. `JavaScript.PackageName` is set to `@supabase/ripe-atlas` in `provider/resources.go`.
2. Create the `@supabase/ripe-atlas` package on npmjs.com under the `@supabase` org scope.
3. Configure OIDC trusted publishing on npmjs.com for this repo.

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

### Migration from workspace to npm (do after first published release)

Once `@supabase/ripe-atlas` is live on npm:

1. In `platform/pulumi/pnpm-workspace.yaml`, add to the catalog:
   ```yaml
   "@supabase/ripe-atlas": "^X.Y.Z"
   ```
2. In `platform/pulumi/monitoring-ripe-atlas/package.json`, change to `"catalog:"`.
3. Delete `platform/pulumi/ripe-atlas-sdk/` and run `pnpm install`.

---

## Version

Done. `provider/version/version.go` holds the `Version` variable; `provider/resources.go`
reads it; `.goreleaser.yml` injects it at build time via ldflags. In local dev builds the
version defaults to `0.0.1`.
