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
One competition as identified by its own id in the results database; may span several **Days**.
_Avoid_: Competition, race, varžybos

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

### Rating and reporting

**Period**:
Inclusive date range (from–to) of **Events** included in a report; an Event is in or out as a whole, never split by Day.
_Avoid_: Season (when meaning the date range)

**Season Rating**:
Internal ranking of the **Participants**, computed from their results across **Events** in a **Period**.
_Avoid_: Ranking, standings

**Event Coefficient**:
Weight assigned to an **Event** that scales its contribution to the **Season Rating**; 1.00 unless changed manually.
_Avoid_: Multiplier, weight factor

**Report**:
Snapshot of all collected results of the **Participants** for one **Period**, produced by a single run; every run produces a new Report, older ones are kept.
_Avoid_: Export, output

## Relationships

- A **Participant** maps to at most one person id; a Participant without one cannot be collected
- An **Event** has one or more **Days**; two competitions on consecutive dates with different ids are two separate **Events**, never Days of one Event
- An **Event** has exactly one **Event Coefficient**; a **Season Rating** is computed from many Events
- A participant has one **Day Result** per **Day** of the **Event**; a Day not `ok` (including `not entered`) has no time
- A **Leader Result** is the sum of one **Day Leader Result** per **Day**, always over all Days of the Event regardless of which Days the participant ran; it is not the overall Event winner's total
- Each **Day Leader Result** comes from the **Class** the participant ran that Day; if the Class changed between Days, each Day uses its own Class
- **Place** is never derived from **Day Places**; it is the published total only
- A **Report** covers one **Period** and lists one row per **Participant** × **Event**

## Example dialogue

> **Dev:** "Vardenis Pavardenis ran Pavyzdžio taurė, two **Days**, dsq on Day 2. What's their **Participant Result**?"
> **Domain expert:** "Only Day 1 time — Day 2 has **Day Status** dsq and no time. **Place** stays empty because the site publishes no total for them."
> **Dev:** "And the **Leader Result**?"
> **Domain expert:** "Day 1 leader plus Day 2 leader of their **Class**, even if they're different people. The rating formula later gives their Day 2 zero — we never write a 0 time."
> **Dev:** "Bandomasis čempionatas on Saturday and Pavyzdžio sprintas on Sunday — one Event with two Days?"
> **Domain expert:** "No, different ids, so two **Events**, each with its own **Event Coefficient**."

## Flagged ambiguities

- Rating formula not decided yet — data collection must keep enough raw data (times, places, leader times, statuses) to support any formula later.
- "Participant gets 0 for a dsq/dns day" means zero contribution in the rating calculation, not a 0 time — a 0 time would look faster than the leader. Stored data keeps the **Day Status** and no time.
- "Season" was used both for the rating and the date range — resolved: **Season Rating** is the ranking, **Period** is the date range.
