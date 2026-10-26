# Reitingtuvas

Collects orienteering results of a known group of participants from a public results database and produces an internal season rating report.

## Language

### People

**Participant**:
A tracked person on the internal list, identified by their person id (sid) in the results database; name is for display only.
_Avoid_: Athlete, runner, member, competitor

**Resolution**:
Matching a **Participant**'s surname + name to exactly one person id in the results database; ambiguous or missing matches are settled manually.

### Events

**Event**:
One competition as identified by its own id in a results source; may span several **Days**. Events from different sources are distinguished by a **Source** marker.
_Avoid_: Competition, race, varžybos

**Source**:
The origin of an **Event**'s data: the primary results database, lof.lv, or manual entry. Stored per Event for traceability; does not affect scoring.
_Avoid_: Provider, origin

**Event Date**:
Start date of an **Event** as published; decides whether the whole Event falls inside the **Period**.

**Day**:
One stage of an **Event**, identified by its number and name, with its own course and own results.
_Avoid_: Stage, etapas, diena

**Day Date**:
Date of a **Day**, for report display only; estimated as **Event Date** + (Day number − 1) because the source does not publish it, and may be corrected manually.

**Class**:
Age/skill group within an **Event** in which a participant competes (e.g. M21, W35); results and places are ranked per Class.
_Avoid_: Group, category

**Sport**:
Discipline of an **Event** as labelled by the results database (foot orienteering, MTB orienteering, running, ...); all Sports are collected.
_Avoid_: Discipline, type

### Results

**Day Result**:
A participant's time on one **Day**, together with its **Day Status** and **Day Place**.
_Avoid_: Split, leg time

**Day Status**:
Outcome of a participant's **Day**: `ok` when a valid time exists, the official non-finish mark (dsq, dns, dnf, mp, ...), or `not entered` when the participant has no entry for that Day at all.
_Avoid_: Result code

**Day Place**:
Participant's place within their **Class** on one **Day**.

**Participant Result**:
Sum of a participant's valid **Day Result** times over all **Days** of an **Event**.
_Avoid_: Total time, final time

**Place**:
Official total place of a participant in their **Class** for the whole **Event**, as published; empty when not published (e.g. any Day not `ok`).
_Avoid_: Rank, position

**Day Leader Result**:
Time of whoever placed 1st in the **Participant**'s **Class** on one **Day**; only the time is kept, not who it was.
_Avoid_: Winner time, best time

**Leader Result**:
Sum of **Day Leader Results** over all **Days** of an **Event** for the participant's **Class**; different Days may have different leaders.
_Avoid_: Winner total, overall winner time

### Relays

**Relay**:
An **Event** (or **Day**) where a team runs consecutive **Legs**; collected, but whether it counts toward the **Season Rating** is not decided yet.

**Leg**:
One participant's part of a **Relay**, identified by its leg number.

For a **Relay**, the individual terms map to the **Leg**: **Participant Result** = own Leg time, **Place** = Leg place, **Day Leader Result** = fastest time on the same Leg number in the **Class**. Team time and team place are kept only as extra information.

### Manual entry

**Manual Event**:
An **Event** entered by hand in the workbook (not scraped). The user provides event name, date, participant name, time, and leader time. The tool resolves participant names against the Dalyviai list (matching both "Surname Name" and "Name Surname" orderings, case-insensitive), generates a deterministic event ID (`manual:YYYY-MM-DD:EventName`), and derives day numbers from sorted unique dates within the same event name. Included in **Day Score** and **Season Score** calculation like any other Event. Events are auto-registered in the Varžybos sheet with coefficient 1.00.
_Avoid_: Custom event, external event

### Rating and reporting

**Period**:
Inclusive date range (from–to) of **Events** included in a report; an Event is in or out as a whole, never split by Day.
_Avoid_: Season (when meaning the date range)

**Day Score**:
Per-Day performance number for a **Participant**: `Day Leader Result / Day Result time × Event Coefficient`. Zero when **Day Status** is not `ok`.
_Avoid_: Day coefficient, day rating, day points

**Season Score**:
Sum of a **Participant**'s top N **Day Scores** (default N=8; fewer if the Participant has fewer); determines the **Season Rating** order.
_Avoid_: Total score, total coefficient

**Season Rating**:
Internal ranking of the **Participants** by **Season Score**, highest first.
_Avoid_: Ranking, standings

**Event Coefficient**:
Weight assigned to an **Event** that scales its contribution to the **Season Rating**; 1.00 unless changed manually. Replaced by **Open Class Coefficient** when the participant's **Class** contains "open" (case-insensitive).
_Avoid_: Multiplier, weight factor

**Open Class Coefficient**:
Fixed coefficient applied instead of (not multiplied by) the **Event Coefficient** when the participant's **Class** contains "open" (case-insensitive anywhere in the name); configurable in Settings (`Atviros klasės koeficientas`), default 0.9.
_Avoid_: OPEN penalty, OPEN multiplier

**Start Count**:
Number of event-days where a **Participant** has any entry (including dsq/dnf — score 0 still counts as a start); dns entries are excluded entirely. Displayed in the report.
_Avoid_: Participation count

**Report**:
Season Rating table: one row per **Participant**, one column per event-**Day**, cell value = **Day Score**. Sorted by **Season Score** descending, ties get equal **Season Rating** place. Includes start count and top-N sum. Created as a new sheet or overwrites the previous one (user chooses). Title row contains period year.
_Avoid_: Export, output

## Relationships

- A **Participant** maps to at most one person id; a Participant without one cannot be collected
- An **Event** has one or more **Days**; two competitions on consecutive dates with different ids are two separate **Events**, never Days of one Event
- An **Event** has exactly one **Event Coefficient**; a **Season Rating** is computed from many Events
- A participant has one **Day Result** per **Day** of the **Event**; a Day not `ok` (including `not entered`) has no time
- A **Leader Result** is the sum of one **Day Leader Result** per **Day**, always over all Days of the Event regardless of which Days the participant ran; it is not the overall Event winner's total
- Each **Day Leader Result** comes from the **Class** the participant ran that Day; if the Class changed between Days, each Day uses its own Class
- **Place** is never derived from **Day Places**; it is the published total only
- A **Report** covers one **Period** and lists one row per **Participant**, one column per event-**Day**

## Example dialogue

> **Dev:** "Vardenis Pavardenis ran Pavyzdžio taurė, two **Days**, dsq on Day 2. What's their **Participant Result**?"
> **Domain expert:** "Only Day 1 time — Day 2 has **Day Status** dsq and no time. **Place** stays empty because the site publishes no total for them."
> **Dev:** "And the **Leader Result**?"
> **Domain expert:** "Day 1 leader plus Day 2 leader of their **Class**, even if they're different people. The rating formula later gives their Day 2 zero — we never write a 0 time."
> **Dev:** "Bandomasis čempionatas on Saturday and Pavyzdžio sprintas on Sunday — one Event with two Days?"
> **Domain expert:** "No, different ids, so two **Events**, each with its own **Event Coefficient**."

## Workbook sheets

| Sheet | Purpose | Written by |
|---|---|---|
| `Nustatymai` | Settings key-value pairs | User |
| `Dalyviai` | Participant list (surname, name, ID, status) | User + tool (SID, status) |
| `Varžybos` | Event metadata (ID, name, date, sport, relay, day count, coefficient) | Tool appends; user edits coefficient |
| `Dienos` | Day metadata (event ID, event name, day #, name, date) | Tool appends; user may correct date |
| `Rezultatai` | Crawled raw results: one row per participant × event × day | Tool (cleared & rebuilt each crawl) |
| `Rankinis įvedimas` | Manual results: Varžybos, Data, Pavardė Vardas, Laikas, Lyderio laikas, Open grupė?, Diskvalifikuotas?, Klaida | User + tool (Klaida) |

`Rezultatai` columns: Dalyvio ID, Varžybų ID, Dalyvis, Varžybos, Diena, Grupė, Laikas, Lyderio laikas, Būsena, Vieta, Nuoroda.
`Rankinis įvedimas` columns: Varžybos (event name), Data (date), Pavardė Vardas (full name), Laikas, Lyderio laikas, Open grupė? (optional boolean), Diskvalifikuotas? (optional boolean), Klaida (tool-written: "nerastas" if name not found, cleared on success). The tool resolves names to SIDs, generates event IDs, derives day numbers, and auto-registers events in Varžybos.

Data never flows between `Rezultatai` and `Rankinis įvedimas`. Both are read by the report generator.

## Data flow

1. User double-clicks exe → tool finds workbook next to itself
2. Prompt: "Parsisiųsti duomenis iš šaltinių? (t/n)"
3. If **t**: crawl sources → clear & rebuild `Rezultatai` → upsert `Varžybos`/`Dienos`
4. If **n**: skip crawl, `Rezultatai` stays as-is
5. Report reads `Rezultatai` + `Rankinis įvedimas` + `Varžybos` (coefficients) + `Nustatymai` (open class coef, top N)
6. Compute Day Scores from raw times, sort by Season Score
7. Prompt: "Perrašyti paskutinę ataskaitą ar sukurti naują? (p/n)"
8. Write report sheet → backup → save

## Report layout

Header rows (3–6):
- Row 3: `Vieta` | `Pavardė Vardas` | event names (1 col per day, multi-day repeats name) | `Startų skaičius` | `Geriausių N suma` | `Pavardė Vardas` | `Vieta`
- Row 4: | `Data` | dates as `MM DD` | | | |
- Row 5: | `Koeficientas` | coefficients (0.00) | event count | max possible score | `"koeficientas"` |
- Row 6: | `Eilės nr.` | 1, 2, 3... | | | |

Data rows: sorted by Season Score desc. Place supports ties ("7-8"). Day Score values: `0.00` format, empty = didn't participate, `0` = dsq/dns. Freeze pane at C1.

Events sorted by date, days 1,2,3 within same event.

## Resolved ambiguities

- ~~Rating formula not decided yet~~ → **Decided**: Day Score = Leader Time / Participant Time × Coefficient. Season Score = top N (default 8) Day Scores. Season Rating = sorted by Season Score desc.
- "Participant gets 0 for a dsq/dns day" means zero contribution in the rating calculation, not a 0 time — a 0 time would look faster than the leader. Stored data keeps the **Day Status** and no time.
- "Season" was used both for the rating and the date range — resolved: **Season Rating** is the ranking, **Period** is the date range.
- ~~Open class detection~~ → any class name containing "open" (case-insensitive), not just prefix.
