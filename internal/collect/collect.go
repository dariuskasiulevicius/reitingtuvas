// Package collect orchestrates participant resolution and result collection.
package collect

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/dariuskasiulevicius/reitingtuvas/internal/domain"
	"github.com/dariuskasiulevicius/reitingtuvas/internal/parse"
	"github.com/dariuskasiulevicius/reitingtuvas/internal/source"
)

// Collector orchestrates the full collection flow.
type Collector struct {
	client   *source.Client
	settings *source.Settings
	progress source.ProgressFunc
}

// NewCollector creates a Collector.
func NewCollector(client *source.Client, settings *source.Settings, progress source.ProgressFunc) *Collector {
	return &Collector{client: client, settings: settings, progress: progress}
}

// Resolve searches for each participant without a SID and fills it in.
// Returns updated participants with Status set.
func (c *Collector) Resolve(participants []domain.Participant) []domain.Participant {
	for i := range participants {
		p := &participants[i]
		if p.SID != "" {
			p.Status = "rastas"
			continue
		}

		query := p.Surname + " " + p.Name
		path, err := c.settings.Templates.Search.Expand(map[string]string{"query": query})
		if err != nil {
			p.Status = fmt.Sprintf("klaida: %v", err)
			continue
		}

		res := c.client.Fetch(path, time.Time{}, true)
		if res.Error != nil {
			p.Status = fmt.Sprintf("klaida: %v", res.Error)
			continue
		}

		candidates, err := parse.ParseSearch(bytes.NewReader(res.Body), c.settings.Templates.Profile)
		if err != nil {
			p.Status = fmt.Sprintf("klaida: %v", err)
			continue
		}

		// Filter to candidates with a SID
		var withSID []parse.SearchCandidate
		for _, cand := range candidates {
			if cand.SID != "" {
				withSID = append(withSID, cand)
			}
		}

		switch len(withSID) {
		case 0:
			p.Status = "nerastas"
		case 1:
			p.SID = withSID[0].SID
			p.Status = "rastas"
		default:
			var parts []string
			for _, cand := range withSID {
				parts = append(parts, fmt.Sprintf("%s (id:%s, %d varž.)",
					cand.DisplayName, cand.SID, cand.EventCount))
			}
			p.Status = "keli kandidatai: " + strings.Join(parts, ", ")
		}
	}
	return participants
}

// personRow holds a parsed person result row with its source participant.
type personRow struct {
	participant domain.Participant
	row         parse.PersonResultRow
}

// classData holds the parsed results of a class page fetch.
type classData struct {
	page  *parse.ClassPage
	relay []parse.RelayTeam
}

// classKey uniquely identifies a class page to fetch.
type classKey struct {
	eventID string
	class   string
	dayView string // "0" for total, "1","2",... for single day
}

// Collect fetches results for all resolved participants in the period,
// then fetches class pages for leader times, and builds EventResults.
func (c *Collector) Collect(participants []domain.Participant) ([]domain.EventResult, []string) {
	var warnings []string

	// Step 1: fetch person results for each resolved participant
	var allRows []personRow
	for _, p := range participants {
		if p.SID == "" {
			continue
		}

		path, err := c.settings.Templates.Person.Expand(map[string]string{"sid": p.SID})
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("%s %s: %v", p.Surname, p.Name, err))
			continue
		}

		res := c.client.Fetch(path, time.Time{}, true)
		if res.Error != nil {
			warnings = append(warnings, fmt.Sprintf("%s %s: %v", p.Surname, p.Name, res.Error))
			continue
		}

		rows, err := parse.ParsePersonResults(
			bytes.NewReader(res.Body),
			c.settings.Templates.Class,
			c.settings.Templates.Event,
		)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("%s %s: %v", p.Surname, p.Name, err))
			continue
		}

		for _, row := range rows {
			if c.settings.InPeriod(row.Date) {
				allRows = append(allRows, personRow{participant: p, row: row})
			}
		}
	}

	// Step 2: plan class page fetches (deduplicate)
	type fetchPlan struct {
		key       classKey
		eventDate time.Time
		isRelay   bool
	}
	planned := make(map[classKey]fetchPlan)
	for _, pr := range allRows {
		// Extract day view from class href
		vals := c.settings.Templates.Class.Extract(pr.row.ClassHref)
		dayView := "1"
		if vals != nil {
			dayView = vals["day"]
		}

		// Multi-day without total view: plan one fetch per day
		if len(pr.row.DayLabels) > 1 && dayView != "0" && !pr.row.IsRelay {
			for d := 1; d <= len(pr.row.DayLabels); d++ {
				dv := strconv.Itoa(d)
				key := classKey{eventID: pr.row.EventID, class: pr.row.Class, dayView: dv}
				if _, ok := planned[key]; !ok {
					planned[key] = fetchPlan{key: key, eventDate: pr.row.Date, isRelay: false}
				}
			}
			continue
		}

		key := classKey{
			eventID: pr.row.EventID,
			class:   pr.row.Class,
			dayView: dayView,
		}
		if _, ok := planned[key]; !ok {
			planned[key] = fetchPlan{key: key, eventDate: pr.row.Date, isRelay: pr.row.IsRelay}
		}
	}

	// Step 3: fetch class pages
	classResults := make(map[classKey]classData)
	total := len(planned)
	idx := 0

	for _, plan := range planned {
		idx++
		path, err := c.settings.Templates.Class.Expand(map[string]string{
			"event": plan.key.eventID,
			"class": plan.key.class,
			"day":   plan.key.dayView,
		})
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("event %s class %s: %v", plan.key.eventID, plan.key.class, err))
			continue
		}

		if c.progress != nil {
			c.progress(idx, total, fmt.Sprintf("%s %s", plan.key.eventID, plan.key.class))
		}

		res := c.client.Fetch(path, plan.eventDate, false)
		if res.Error != nil {
			warnings = append(warnings, fmt.Sprintf("event %s class %s: %v", plan.key.eventID, plan.key.class, res.Error))
			continue
		}

		if plan.isRelay {
			teams, err := parse.ParseRelayClassPage(
				bytes.NewReader(res.Body),
				c.settings.Templates.Profile,
			)
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("event %s class %s relay: %v", plan.key.eventID, plan.key.class, err))
				continue
			}
			classResults[plan.key] = classData{relay: teams}
		} else {
			page, err := parse.ParseClassPage(
				bytes.NewReader(res.Body),
				c.settings.Templates.Profile,
			)
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("event %s class %s: %v", plan.key.eventID, plan.key.class, err))
				continue
			}
			classResults[plan.key] = classData{page: page}
		}
	}

	// Step 4: build EventResults
	var results []domain.EventResult
	for _, pr := range allRows {
		vals := c.settings.Templates.Class.Extract(pr.row.ClassHref)
		dayView := "1"
		if vals != nil {
			dayView = vals["day"]
		}

		er := c.buildEventResult(pr, dayView, classResults, &warnings)
		results = append(results, er)
	}

	return results, warnings
}

func (c *Collector) buildEventResult(pr personRow, dayView string, allClass map[classKey]classData, warnings *[]string) domain.EventResult {
	row := pr.row

	er := domain.EventResult{
		Participant: pr.participant,
		Event: domain.Event{
			ID:      row.EventID,
			Name:    row.EventName,
			Date:    row.Date,
			Sport:   row.Sport,
			IsRelay: row.IsRelay,
		},
		Class: row.Class,
		Place: row.Place,
	}

	key := classKey{eventID: row.EventID, class: row.Class, dayView: dayView}
	cd := allClass[key]

	if row.IsRelay {
		er = c.buildRelayResult(er, pr, cd)
	} else if len(row.DayLabels) > 1 && dayView != "0" && !row.IsRelay {
		er = c.buildPerDayResult(er, pr, allClass)
	} else if cd.page != nil {
		er = c.buildClassResult(er, pr, cd.page)
	} else {
		er = c.buildFromPersonOnly(er, pr)
		er.Notes = appendNote(er.Notes, "klasės puslapis nepasiekiamas")
	}

	er.Result = domain.ComputeParticipantResult(er.Days)
	er.LeaderResult = domain.ComputeLeaderResult(er.Days)
	er.Event.DayCount = maxInt(len(er.Days), 1)

	return er
}

func (c *Collector) buildClassResult(er domain.EventResult, pr personRow, page *parse.ClassPage) domain.EventResult {
	if len(page.DayNames) == 0 {
		// Single-day event
		dr := domain.DayResult{DayNumber: 1, Class: pr.row.Class}

		// Find participant in class page
		for _, comp := range page.Rows {
			if comp.SID == pr.participant.SID {
				status, t := parse.ParseTimeOrStatus(comp.Total)
				dr.Status = status
				dr.Time = t
				dr.Place = comp.Place
				break
			}
		}
		if dr.Status == "" {
			dr.Status = "not entered"
		}

		// Leader = place 1
		dr.LeaderTime = findLeaderTime(page.Rows)

		er.Days = []domain.DayResult{dr}
	} else {
		// Multi-day event
		for dayIdx, dayName := range page.DayNames {
			dr := domain.DayResult{
				DayNumber: dayIdx + 1,
				Class:     pr.row.Class,
			}
			_ = dayName

			// Find participant's per-day result
			for _, comp := range page.Rows {
				if comp.SID != pr.participant.SID {
					continue
				}
				if dayIdx < len(comp.PerDay) {
					timeStr, place := parse.ParseDayTimeAndPlace(comp.PerDay[dayIdx])
					status, t := parse.ParseTimeOrStatus(timeStr)
					dr.Status = status
					dr.Time = t
					dr.Place = place
				}
				break
			}
			if dr.Status == "" {
				dr.Status = "not entered"
			}

			// Day leader = place 1 for this day column
			dr.LeaderTime = findDayLeaderTime(page.Rows, dayIdx)

			er.Days = append(er.Days, dr)
		}
	}
	return er
}

func (c *Collector) buildPerDayResult(er domain.EventResult, pr personRow, allClass map[classKey]classData) domain.EventResult {
	for i := range pr.row.DayLabels {
		dayNum := i + 1
		dr := domain.DayResult{DayNumber: dayNum, Class: pr.row.Class}

		// Participant time/place from person page DayTimes
		if i < len(pr.row.DayTimes) {
			timeStr, place := parse.ParseDayTimeAndPlace(pr.row.DayTimes[i])
			status, t := parse.ParseTimeOrStatus(timeStr)
			dr.Status = status
			dr.Time = t
			dr.Place = place
		}
		if dr.Status == "" {
			dr.Status = "not entered"
		}

		// Leader time from per-day class page
		dayKey := classKey{
			eventID: pr.row.EventID,
			class:   pr.row.Class,
			dayView: strconv.Itoa(dayNum),
		}
		if cd, ok := allClass[dayKey]; ok && cd.page != nil {
			dr.LeaderTime = findLeaderTime(cd.page.Rows)
		}

		er.Days = append(er.Days, dr)
	}
	return er
}

func (c *Collector) buildRelayResult(er domain.EventResult, pr personRow, cd classData) domain.EventResult {
	// Find participant's leg number from person page
	legNum := 0
	for _, label := range pr.row.DayLabels {
		if m := parse.LegRegex().FindStringSubmatch(label); m != nil {
			fmt.Sscanf(m[1], "%d", &legNum)
		}
	}
	er.RelayLeg = legNum

	dr := domain.DayResult{DayNumber: 1, Class: pr.row.Class}

	// Find participant in relay teams
	if cd.relay != nil {
		for _, team := range cd.relay {
			for _, leg := range team.Legs {
				if leg.SID == pr.participant.SID {
					status, t := parse.ParseTimeOrStatus(leg.Time)
					dr.Status = status
					dr.Time = t
					dr.Place = leg.Place
					break
				}
			}
		}

		// Leader = fastest time on same leg number
		if legNum > 0 {
			dr.LeaderTime = findRelayLegLeader(cd.relay, legNum)
		}
	}
	if dr.Status == "" {
		dr.Status = "not entered"
	}

	er.Days = []domain.DayResult{dr}
	return er
}

func (c *Collector) buildFromPersonOnly(er domain.EventResult, pr personRow) domain.EventResult {
	// Fallback when class page is unavailable
	if len(pr.row.DayTimes) == 0 {
		// Single day
		dr := domain.DayResult{DayNumber: 1, Class: pr.row.Class}
		status, t := parse.ParseTimeOrStatus(pr.row.Result)
		dr.Status = status
		dr.Time = t
		dr.Place = pr.row.Place
		er.Days = []domain.DayResult{dr}
	} else {
		for i, rawTime := range pr.row.DayTimes {
			dr := domain.DayResult{DayNumber: i + 1, Class: pr.row.Class}
			timeStr, place := parse.ParseDayTimeAndPlace(rawTime)
			status, t := parse.ParseTimeOrStatus(timeStr)
			dr.Status = status
			dr.Time = t
			dr.Place = place
			er.Days = append(er.Days, dr)
		}
	}
	return er
}

// findLeaderTime returns the time of the place-1 competitor (single-day).
// Falls back to the shortest valid time when no row has Place == 1.
func findLeaderTime(rows []parse.ClassCompetitor) time.Duration {
	var best time.Duration
	for _, r := range rows {
		if r.Place == 1 {
			_, t := parse.ParseTimeOrStatus(r.Total)
			return t
		}
		_, t := parse.ParseTimeOrStatus(r.Total)
		if t > 0 && (best == 0 || t < best) {
			best = t
		}
	}
	return best
}

// findDayLeaderTime returns the leader time for a specific day column.
func findDayLeaderTime(rows []parse.ClassCompetitor, dayIdx int) time.Duration {
	var best time.Duration
	for _, r := range rows {
		if dayIdx >= len(r.PerDay) {
			continue
		}
		timeStr, place := parse.ParseDayTimeAndPlace(r.PerDay[dayIdx])
		if place == 1 {
			_, t := parse.ParseTimeOrStatus(timeStr)
			return t
		}
		// Fallback: if no explicit place 1, take the shortest time
		_, t := parse.ParseTimeOrStatus(timeStr)
		if t > 0 && (best == 0 || t < best) {
			best = t
		}
	}
	return best
}

// findRelayLegLeader returns the fastest time on a given leg number across all teams.
func findRelayLegLeader(teams []parse.RelayTeam, legNum int) time.Duration {
	var best time.Duration
	for _, team := range teams {
		for _, leg := range team.Legs {
			if leg.LegNumber == legNum {
				_, t := parse.ParseTimeOrStatus(leg.Time)
				if t > 0 && (best == 0 || t < best) {
					best = t
				}
			}
		}
	}
	return best
}

func appendNote(existing, note string) string {
	if existing == "" {
		return note
	}
	return existing + "; " + note
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
