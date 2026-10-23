package loflv

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	"github.com/dariuskasiulevicius/reitingtuvas/internal/domain"
	"github.com/dariuskasiulevicius/reitingtuvas/internal/source"
)

// Collector orchestrates lof.lv result collection.
type Collector struct {
	client   *source.Client
	settings *LVSettings
	period   *source.Settings
	progress source.ProgressFunc
}

// NewCollector creates a lof.lv Collector.
func NewCollector(client *source.Client, lvSettings *LVSettings, period *source.Settings, progress source.ProgressFunc) *Collector {
	return &Collector{
		client:   client,
		settings: lvSettings,
		period:   period,
		progress: progress,
	}
}

// Collect fetches and matches lof.lv results for participants.
func (c *Collector) Collect(participants []domain.Participant) ([]domain.EventResult, []string) {
	var results []domain.EventResult
	var warnings []string

	// Fetch series page
	seriesPath := "/seriali_rez/?id=" + c.settings.SeriesID
	seriesRes := c.client.Fetch(seriesPath, time.Now(), false)
	if seriesRes.Error != nil {
		warnings = append(warnings, fmt.Sprintf("LV serija: %v", seriesRes.Error))
		return nil, warnings
	}

	events, err := ParseSeriesPage(bytes.NewReader(seriesRes.Body))
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("LV serija: %v", err))
		return nil, warnings
	}

	// Filter by period
	var filtered []SeriesEvent
	for _, ev := range events {
		if !ev.Date.IsZero() && c.period.InPeriod(ev.Date) {
			filtered = append(filtered, ev)
		}
	}

	// Group: standalone vs multi-day sub-events
	standalone, multiDayGroups := groupEvents(filtered)

	idx := newParticipantIndex(participants)

	total := len(standalone) + len(multiDayGroups)
	step := 0

	// Process standalone single-day events
	for _, ev := range standalone {
		step++
		if c.progress != nil {
			c.progress(step, total, ev.Name)
		}

		if ev.IsRelay {
			res, warns := c.collectRelay(ev, idx)
			results = append(results, res...)
			warnings = append(warnings, warns...)
			continue
		}

		res, warns := c.collectSingleDay(ev, idx)
		results = append(results, res...)
		warnings = append(warnings, warns...)
	}

	// Process multi-day groups
	for parentID, group := range multiDayGroups {
		step++
		if c.progress != nil && len(group) > 0 {
			c.progress(step, total, group[0].Name)
		}

		res, warns := c.collectMultiDay(parentID, group, idx)
		results = append(results, res...)
		warnings = append(warnings, warns...)
	}

	return results, warnings
}

func (c *Collector) collectSingleDay(ev SeriesEvent, idx *participantIndex) ([]domain.EventResult, []string) {
	var results []domain.EventResult
	var warnings []string

	body, err := c.fetchResults(ev.ID, 1)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("LV %d: %v", ev.ID, err))
		return nil, warnings
	}

	blocks, err := ParseEventResults(body)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("LV %d parse: %v", ev.ID, err))
		return nil, warnings
	}

	for _, block := range blocks {
		leaderTime := findEventLeaderTime(block.Rows)

		for _, row := range block.Rows {
			if row.Status == "-" {
				continue // skip out-of-competition
			}

			p, ok := idx.Match(row.FirstName, row.LastName)
			if !ok {
				continue
			}

			er := domain.EventResult{
				Participant: p,
				Event: domain.Event{
					ID:       lvEventID(ev.ID),
					Name:     ev.Name,
					Date:     ev.Date,
					DayCount: 1,
				},
				Class: row.Class,
				Place: row.Place,
				Days: []domain.DayResult{
					{
						DayNumber:  1,
						Class:      row.Class,
						Status:     mapLVStatus(row.Status),
						Time:       row.Time,
						Place:      row.Place,
						LeaderTime: leaderTime,
					},
				},
			}
			if row.Time > 0 {
				er.Result = row.Time
			}
			er.LeaderResult = leaderTime
			results = append(results, er)
		}
	}

	return results, warnings
}

func (c *Collector) collectMultiDay(parentID int, subEvents []SeriesEvent, idx *participantIndex) ([]domain.EventResult, []string) {
	var results []domain.EventResult
	var warnings []string

	// Use kopvērtējums (r=99) on the parent event
	body, err := c.fetchResults(parentID, 99)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("LV kopv %d: %v", parentID, err))
		return nil, warnings
	}

	blocks, err := ParseMultiDayResults(body)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("LV kopv %d parse: %v", parentID, err))
		return nil, warnings
	}

	// Find parent event metadata
	parentEvent := findParentEvent(parentID, subEvents)

	for _, block := range blocks {
		for _, row := range block.Rows {
			p, ok := idx.Match(row.FirstName, row.LastName)
			if !ok {
				continue
			}

			er := domain.EventResult{
				Participant: p,
				Event: domain.Event{
					ID:       lvEventID(parentID),
					Name:     parentEvent.Name,
					Date:     parentEvent.Date,
					DayCount: block.DayCount,
				},
				Class:  row.Class,
				Place:  row.Place,
				Result: row.TotalTime,
			}

			// Per-day results
			for d, dayRes := range row.Days {
				dayLeader := findMultiDayLeaderTime(block.Rows, d)
				dr := domain.DayResult{
					DayNumber:  d + 1,
					Class:      row.Class,
					Time:       dayRes.Time,
					Place:      dayRes.Place,
					LeaderTime: dayLeader,
				}
				if dayRes.Time > 0 {
					dr.Status = "ok"
				} else {
					dr.Status = "dns"
				}
				er.Days = append(er.Days, dr)
			}

			// Leader result = sum of day leader times
			var totalLeader time.Duration
			for _, d := range er.Days {
				totalLeader += d.LeaderTime
			}
			er.LeaderResult = totalLeader

			results = append(results, er)
		}
	}

	return results, warnings
}

func (c *Collector) collectRelay(ev SeriesEvent, idx *participantIndex) ([]domain.EventResult, []string) {
	var results []domain.EventResult
	var warnings []string

	body, err := c.fetchResults(ev.ID, 1)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("LV relay %d: %v", ev.ID, err))
		return nil, warnings
	}

	blocks, err := ParseRelayResults(body)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("LV relay %d parse: %v", ev.ID, err))
		return nil, warnings
	}

	for _, block := range blocks {
		// Find leader leg times per leg number
		legLeaders := make(map[int]time.Duration)
		for _, team := range block.Teams {
			if team.Place == 1 {
				for _, leg := range team.Legs {
					legLeaders[leg.LegNumber] = leg.Time
				}
				break
			}
		}

		for _, team := range block.Teams {
			for _, leg := range team.Legs {
				p, ok := idx.Match(leg.FirstName, leg.LastName)
				if !ok {
					continue
				}

				er := domain.EventResult{
					Participant: p,
					Event: domain.Event{
						ID:       lvEventID(ev.ID),
						Name:     ev.Name,
						Date:     ev.Date,
						IsRelay:  true,
						DayCount: 1,
					},
					Class:    block.ClassName,
					RelayLeg: leg.LegNumber,
					Place:    team.Place,
					Result:   leg.Time,
					Days: []domain.DayResult{
						{
							DayNumber:  1,
							Class:      block.ClassName,
							Status:     "ok",
							Time:       leg.Time,
							Place:      team.Place,
							LeaderTime: legLeaders[leg.LegNumber],
						},
					},
					LeaderResult: legLeaders[leg.LegNumber],
				}
				results = append(results, er)
			}
		}
	}

	return results, warnings
}

func (c *Collector) fetchResults(eventID int, view int) (string, error) {
	path := "/seriali_rez/source.php"
	fd := formData(eventID, view)

	// Use event date for caching (zero = recent → cache by finalDays)
	res := c.client.Post(path, fd, time.Now(), false)
	if res.Error != nil {
		return "", res.Error
	}

	return unwrapResponse(res.Body)
}

func groupEvents(events []SeriesEvent) (standalone []SeriesEvent, multiDay map[int][]SeriesEvent) {
	multiDay = make(map[int][]SeriesEvent)

	for _, ev := range events {
		if ev.IsStandalone() {
			standalone = append(standalone, ev)
		} else {
			multiDay[ev.ParentID] = append(multiDay[ev.ParentID], ev)
		}
	}

	return
}

func findParentEvent(parentID int, subEvents []SeriesEvent) SeriesEvent {
	// The parent event might be in the subEvents list itself
	var earliest SeriesEvent
	for _, ev := range subEvents {
		if ev.ID == parentID {
			return ev
		}
		if earliest.ID == 0 || ev.Date.Before(earliest.Date) {
			earliest = ev
		}
	}
	// Use earliest sub-event as fallback
	if earliest.ID != 0 {
		earliest.ID = parentID
	}
	return earliest
}

func findEventLeaderTime(rows []EventRow) time.Duration {
	// First try place==1
	for _, r := range rows {
		if r.Place == 1 && r.Time > 0 {
			return r.Time
		}
	}
	// Fallback: shortest valid time
	var min time.Duration
	for _, r := range rows {
		if r.Time > 0 && r.Status == "ok" {
			if min == 0 || r.Time < min {
				min = r.Time
			}
		}
	}
	return min
}

func findMultiDayLeaderTime(rows []MultiDayRow, dayIndex int) time.Duration {
	// Find place==1 for this day
	for _, r := range rows {
		if dayIndex < len(r.Days) && r.Days[dayIndex].Place == 1 && r.Days[dayIndex].Time > 0 {
			return r.Days[dayIndex].Time
		}
	}
	// Fallback: shortest
	var min time.Duration
	for _, r := range rows {
		if dayIndex < len(r.Days) && r.Days[dayIndex].Time > 0 {
			if min == 0 || r.Days[dayIndex].Time < min {
				min = r.Days[dayIndex].Time
			}
		}
	}
	return min
}

func lvEventID(id int) string {
	return "lv:" + strconv.Itoa(id)
}

func mapLVStatus(s string) string {
	switch s {
	case "ok", "dns", "dsq", "mp", "dnf":
		return s
	case "":
		return "ok"
	default:
		return s
	}
}
