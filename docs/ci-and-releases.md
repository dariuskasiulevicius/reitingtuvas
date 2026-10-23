# CI and Releases

## CI (`ci.yml`)

Runs on every push to `main` and every pull request.

| Step | What |
|------|------|
| `go vet ./...` | Static analysis |
| `go test ./...` | All tests |

Runs on 3 OS: ubuntu, windows, macos.

## Release (`release.yml`)

Triggered by pushing a `v*` tag.

### How to release

```bash
git checkout main
git pull origin main
git tag v0.X.Y
git push origin v0.X.Y
```

### What happens

1. Builds 3 binaries (cross-compiled, CGO_ENABLED=0):
   - `windows/amd64` (.exe)
   - `darwin/amd64`
   - `darwin/arm64`
2. Version baked via `-ldflags "-X main.version=${VERSION}"`
3. Each binary zipped as `reitingtuvas-{os}-{arch}.zip`
4. `docs/naudotojo-gidas.md` included in zip if it exists
5. GitHub Release created automatically via `softprops/action-gh-release`

### Build artifacts

| File | Target |
|------|--------|
| `reitingtuvas-windows-amd64.zip` | Windows users (main audience) |
| `reitingtuvas-darwin-amd64.zip` | Mac Intel |
| `reitingtuvas-darwin-arm64.zip` | Mac Apple Silicon |

## Important to remember

- **Tag goes on `main`** — always pull latest main before tagging.
- **Merge PRs first** — the tag must include all intended changes.
- **Version format**: `v0.X.Y` — semver, prefixed with `v`.
- **Manual releases** (via `gh release create`) create a release but skip the CI build matrix. The GHA workflow builds from the tag, so both approaches produce the same binaries — but only the GHA path runs on all 3 OS runners.
- **No linux binary** — not in the matrix. Add if needed.
- **The v0.2.0 release was created manually** with locally cross-compiled binaries. Starting from v0.2.1, releases are built by GHA.
- **`.forbidden-strings`** — always check `git diff` against it before committing. The leak check doesn't run in CI (it needs the local-only `.forbidden-strings` file).
- **Binary is public** — source URL and paths must stay in the workbook only, never compiled into the binary. The `-X main.version` flag is the only linker injection.
