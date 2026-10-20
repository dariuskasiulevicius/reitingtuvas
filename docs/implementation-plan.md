# Implementation plan — phase 1: collect results, produce Report

Terms are defined in [CONTEXT.md](../CONTEXT.md). Decisions: [ADR 0001](adr/0001-go-single-binary-double-click.md), [ADR 0002](adr/0002-source-site-never-in-repo.md).

**In scope:** resolve **Participants**, collect every **Event** in the **Period** with **Day Results**, **Day Leader Results**, **Place**, write **Events**/**Days** sheets and a new **Report** sheet.
**Out of scope:** the **Season Rating** formula. **Participants** without a person id are reported and skipped.

All examples below use made-up names, hosts and paths. Nothing real goes into the repo (ADR 0002).

---

## M0. Prerequisites

- Install Go (current stable): `brew install go`.
- Dependencies (keep to these two):
  - `github.com/xuri/excelize/v2`: read/write xlsx.
  - `github.com/PuerkitoBio/goquery`: HTML parsing.
- Everything else is the standard library (`net/http`, `time`, `os`, `crypto/sha256`).

## M1. Repo skeleton and hygiene (first commit)

```
cmd/reitingtuvas/main.go      entry point, double-click flow
internal/domain/              types from CONTEXT.md, sums, no I/O
internal/source/              HTTP client: delay, back-off, disk cache, URL templates
internal/parse/               HTML → raw rows (search, person results, class, relay)
internal/collect/             orchestration: resolve → collect → build domain model
internal/workbook/            xlsx: template, read settings/participants, upsert events/days, write report, backup
internal/report/              domain model → report rows/columns (Lithuanian headers)
internal/texts/               all user-facing Lithuanian strings in one place
testdata/                     synthetic HTML fixtures only
.github/workflows/            ci.yml, release.yml
```

`.gitignore`:
```
*.xlsx
cache/
dist/
.forbidden-strings
```

Leak guard: `make check-leaks` (or `go run ./internal/tools/leakcheck`) reads `.forbidden-strings`, which is git-ignored and lists the real host, path words and real names one per line. It fails if any tracked file contains one of them. It runs locally before pushing. It cannot run in CI, because CI must not know the strings.

## M2. Source address as URL templates (closes the path leak)

The site's path words are distinctive enough to identify it, so they cannot live in code. Settings holds the base address **and** the path templates with placeholders:

| Setting | Example (fake) |
|---|---|
| Šaltinio adresas | `https://results.example.org` |
| Paieškos kelias | `/p/find?q={query}` |
| Dalyvio rezultatų kelias | `/p/find?person={person}` |
| Klasės rezultatų kelias | `/e/{event}/class/{class}?day={day}` |
| Varžybų kelias | `/e/{event}` |

`internal/source` builds request URLs from the templates. It also compiles each template into a regex so that the parsers can pull ids out of `href`s (`{event}` → `(?P<event>\d+)`, and so on). Code and fixtures only ever use the fake templates. The admin fills in the real ones once in the workbook they hand out.

## M3. HTTP client and cache (`internal/source`)

- Requests are strictly sequential. Delay between requests comes from Settings (`Pauzė tarp užklausų, ms`, default **200**). User-Agent: `reitingtuvas/<version>`.
- Back-off on network error, 429 or 5xx: wait 5 s, then 30 s, then give up on that page. The page is recorded as a warning and the run continues.
- Disk cache in `cache/` next to the executable. File name = sha256 of path+query. Stores body + fetched-at.
  - Search and person results pages: always re-fetched.
  - Class pages: if the **Event Date** is older than `Rezultatai galutiniai po, d.` (default **14**) and a cached copy exists, use the cache. Otherwise re-fetch.
- Progress line: `[123/540] Pavyzdžio taurė M21 …`. It counts planned fetches, so the total is known once the person pages are read.

## M4. Parsers (`internal/parse`) with synthetic fixtures

Each parser returns raw rows with no domain logic. If an expected header or table is missing, it fails with a "page structure changed" error rather than guessing.

| Parser | Input | Output |
|---|---|---|
| Search | search results page | candidates: display name, person id (may be empty), events count |
| Person results | person results page | per row: Event Date, event id, event name, Sport, Class, day selector from class link, total result text, total place, day/leg labels, per Day `time-or-status` + Day Place |
| Class (total view) | class page, multi-Day | Day names; per competitor per Day: person id, time-or-status, Day Place |
| Class (single Day) | class page, one Day | per competitor: person id, time-or-status, place |
| Relay class | relay class page | per team per Leg: leg number, person id, time-or-status, Leg place |

Time parsing: `h:mm:ss` and `m:ss` → `time.Duration`. Anything else that isn't a time → **Day Status** = lowercase raw text (`dsq`, `dns`, `dnf`, `mp`, …).

Fixtures in `testdata/`: hand-written HTML that keeps only the table structure the parser depends on. They use fake names (`Pavardenis Vardenis`), fake ids, fake paths matching the fake templates, and no head/scripts. Each fixture covers one edge case:
- single-Day Event
- 2-Day Event with total
- 3-Day Event with dsq on Day 3 (no total, no Place)
- Participant skipped Day 2
- Class changed between Days (two person rows, same Event)
- Relay
- search with 0, 1, many hits, and a hit without a person id

## M5. Domain and collection (`internal/domain`, `internal/collect`)

The types mirror CONTEXT.md: `Participant`, `Event`, `Day`, `DayResult{Status, Time, Place, Class}`, `EventResult{ParticipantResult, Place, LeaderResult, Days[]}`, `Relay/Leg`.

Flow:
1. **Resolution:** for each `Dalyviai` row with an empty ID, search `"{surname} {name}"`.
   - Exactly 1 hit with an id → write the ID back, status "rastas".
   - 0 hits → "nerastas".
   - More than 1 hit → "keli kandidatai: Vardas (id, N varž.), …".
   - The run carries on with the resolved ones.
2. For each resolved **Participant**: fetch their person results page and keep the rows whose **Event Date** is within the **Period**.
3. Merge rows with the same event id (Class changed between Days).
4. Plan the class page fetches: one per unique (event, class, view) across all Participants. This gives the progress total.
5. Fetch the class pages. For each Day, the **Day Leader Result** = time of the place-1 row. For relays, it is the place-1 time on the same Leg.
6. Build each Participant × Event:
   - Days come from the class page's Day list.
   - The Participant's Day Result comes from their row; if they have none, status is `not entered`.
   - **Participant Result** = sum of `ok` times.
   - **Leader Result** = sum of all Day Leader Results.
   - **Place** = published total (single Day: the place).
7. Domain tests cover all the CONTEXT.md relationships: dsq has no time, the Leader Result covers all Days, class change, relay mapping.

## M6. Workbook (`internal/workbook`)

One file, `reitingtuvas.xlsx`, next to the executable. Columns are found by header text, not position, so users can reorder columns or add their own.

| Sheet | Columns | Who writes |
|---|---|---|
| `Nustatymai` | Raktas, Reikšmė (rows from M2/M3 + Laikotarpis nuo, Laikotarpis iki) | user (admin) |
| `Dalyviai` | Pavardė, Vardas, ID, Būsena | user; tool fills ID if empty and always overwrites Būsena |
| `Varžybos` | ID, Pavadinimas, Data, Sportas, Estafetė, Dienų sk., Koeficientas | tool appends new rows only; Koeficientas = 1.00 on insert, never overwritten |
| `Dienos` | Varžybų ID, Varžybos, Diena, Dienos pavadinimas, Data | tool appends only; Data = Event Date + (N−1) on insert, never overwritten |
| `Ataskaita YYYY-MM-DD HH.MM` | see M7 | tool, new sheet each run |

Day Dates live in their own `Dienos` sheet, one row per Event × Day. This avoids a variable number of date columns in `Varžybos`.

Safety:
- If the workbook is missing, create it with all sheets and headers plus Settings keys with empty values. Print what to fill in, then stop.
- Lock check before doing any network work: try to open the file read-write. If it fails, print "Uždarykite reitingtuvas.xlsx ir paleiskite iš naujo" and stop. This avoids a long run that can't be saved.
- Before saving: copy the workbook to `reitingtuvas.backup.xlsx`.
- Roundtrip test: user-edited Koeficientas, Data and an extra user column survive a run.

## M7. Report sheet (`internal/report`)

One row per Participant × Event, sorted by Participant, then Event Date.

Fixed columns:

`Dalyvis | ID | Varžybų data | Varžybos | Varžybų ID | Sportas | Klasė | Estafetės etapas | Koef. | Dalyvio rezultatas | Vieta | Lyderio rezultatas | Pastabos`

Then one block per Day, up to the maximum Day count in the Report:

`D{n} data | D{n} būsena | D{n} laikas | D{n} vieta | D{n} lyderio laikas`

- Times are written as Excel durations (day fraction) with format `[h]:mm:ss`, so formulas work in phase 2.
- Statuses are translated to Lithuanian for display (`ok` → `ok`, `dsq` → `diskv.`, `not entered` → `nedalyvavo`, …). The mapping lives in `internal/texts`.
- Header row frozen, autofilter on.
- `Pastabos` holds per-row warnings, such as a class page that failed to download.

## M8. Entry point (`cmd/reitingtuvas`)

1. Find the executable's directory: `os.Executable()` + `filepath.EvalSymlinks`. Ignore the working directory (on macOS a double-click runs in `$HOME`).
2. Steps: workbook lock check → load and validate Settings → resolution → collection with progress → upsert sheets → Report → backup → save.
3. Print a summary: Participants collected or skipped, Events, warnings, name of the Report sheet.
4. Always finish with "Spauskite Enter, kad uždarytumėte", including after errors, so the console window doesn't vanish.
5. Dev flags (not needed by users): `-workbook <path>`, `-no-pause`.

## M9. Tests

- Unit: parsers against fixtures; domain sums and edge cases; template ↔ regex; time parsing; workbook roundtrip in a temp dir.
- Smoke (opt-in, local only): `go test -tags smoke ./...` with `REITINGTUVAS_WORKBOOK=/path/to/real.xlsx`. Runs against the real site using the address and templates in that workbook. Never in CI.

## M10. CI and release

- `ci.yml` (push/PR): `go vet`, `go test ./...` on ubuntu, windows and macos.
- `release.yml` (tag `v*`): `CGO_ENABLED=0` builds for `windows/amd64`, `darwin/amd64` and `darwin/arm64`. The Go linker ad-hoc signs arm64 automatically.
  - Zip each build with the Lithuanian user guide.
  - Attach the zips to the GitHub Release.
  - The version is injected via `-ldflags "-X main.version=…"`.

## M11. User guide (Lithuanian, `docs/naudotojo-gidas.md`, shipped in the zip)

1. Download the zip for your system from Releases and unzip it.
2. Put the `reitingtuvas.xlsx` you received from the admin in the same folder.
3. First run warnings:
   - Windows: "Daugiau informacijos → Vis tiek paleisti".
   - macOS: System Settings → Privacy & Security → "Open Anyway".
4. Double-click, wait, then open the workbook and find the new `Ataskaita …` sheet.
5. Fill in Koeficientas in `Varžybos` and fix Data in `Dienos` as needed, then run again.
6. What to do with `Būsena` "keli kandidatai" or "nerastas" in `Dalyviai`.

Screenshots are added after the first working build.

## Order of work

| # | Milestone | Done when |
|---|---|---|
| 1 | M0–M1 skeleton, `.gitignore`, leak guard, CI | CI green on 3 OS with an empty test |
| 2 | M2 templates + M4 parsers + fixtures | all fixture tests pass |
| 3 | M5 domain + collection (with a fake source) | edge-case domain tests pass |
| 4 | M3 HTTP client + cache | back-off and cache policy tested with `httptest` |
| 5 | M6 workbook + M7 report | roundtrip test passes, report opens in Excel/Numbers |
| 6 | M8 entry point | double-click works on Windows and macOS with the real workbook (smoke run) |
| 7 | M10 release + M11 guide | tagged release downloadable; non-tech user follows the guide unaided |
