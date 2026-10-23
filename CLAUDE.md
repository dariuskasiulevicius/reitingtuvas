# Reitingtuvas

Orienteering season rating tool. Go CLI that reads/writes an xlsx workbook.

## Key docs

- `CONTEXT.md` — domain glossary, data model, workbook structure
- `docs/ci-and-releases.md` — CI pipeline, release process, important gotchas

## Security

This is a **public repo**. The source site address, path words, and real personal names must NEVER appear in any committed file.

- `.forbidden-strings` (gitignored) lists banned strings — scan tracked files against it before pushing
- Source base URL and path templates live only in the workbook's Nustatymai sheet (never compiled into binary)
- `participants.csv`/xlsx contains personal data — never committed
- Test fixtures use synthetic HTML with fake names only

## Build & test

```bash
go test ./...
go vet ./...
go build -o dist/reitingtuvas ./cmd/reitingtuvas
```

## Release

See `docs/ci-and-releases.md`. Short version:

```bash
git tag v0.X.Y && git push origin v0.X.Y
```

GHA builds 3 platforms and creates a GitHub Release automatically.
