# Source site never appears in the repository

The repository is public, but the results database we collect from must not be identifiable from it, and its pages contain personal data of third parties. So the source address lives only in private configuration, downloaded pages live only in a git-ignored cache, and parser tests use hand-made synthetic HTML fixtures (fake names, fake ids, no address) instead of saved real pages. Anything committed to a public repo stays in its history forever, so "just commit one real page as a test fixture" must not happen.

## Consequences

- The source address is entered in the `Settings` sheet of the private workbook, never compiled into the binary — released binaries are public, and a compiled-in string is readable from them.
- Real-site checks run only as an opt-in local smoke test, never in CI. A layout change on the site is caught by the parser failing loudly on unexpected structure, not by CI.
- The participant list, workbook and generated reports are also git-ignored (personal data); only example files with fake rows are committed.
