package loflv

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"time"
)

// SeriesEvent is one event from the visas_kartas JS array.
type SeriesEvent struct {
	ID       int
	Name     string
	Date     time.Time
	IsRelay  bool
	DayCount int
	ParentID int // 0 = standalone or parent itself
}

type rawSeriesEvent struct {
	ID        string  `json:"id"`
	Name      string  `json:"sacen"`
	Date      string  `json:"datums"`
	Relay     *string `json:"stafete"`
	DayCount  string  `json:"cik_dienas"`
	ParentID  *string `json:"sacid"`
}

var visasKartasRe = regexp.MustCompile(`visas_kartas\s*=\s*(\[.*?\])\s*;`)

// ParseSeriesPage extracts the visas_kartas JavaScript array from series HTML.
func ParseSeriesPage(r io.Reader) ([]SeriesEvent, error) {
	body, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read series page: %w", err)
	}

	m := visasKartasRe.FindSubmatch(body)
	if m == nil {
		return nil, fmt.Errorf("series page structure changed: visas_kartas not found")
	}

	var raw []rawSeriesEvent
	if err := json.Unmarshal(m[1], &raw); err != nil {
		return nil, fmt.Errorf("decode visas_kartas: %w", err)
	}

	events := make([]SeriesEvent, 0, len(raw))
	for _, r := range raw {
		id, _ := strconv.Atoi(r.ID)
		if id == 0 {
			continue
		}

		date, _ := time.Parse("2006-01-02", r.Date)
		days, _ := strconv.Atoi(r.DayCount)
		if days < 1 {
			days = 1
		}

		var parentID int
		if r.ParentID != nil && *r.ParentID != "" {
			parentID, _ = strconv.Atoi(*r.ParentID)
		}

		events = append(events, SeriesEvent{
			ID:       id,
			Name:     r.Name,
			Date:     date,
			IsRelay:  r.Relay != nil && *r.Relay == "1",
			DayCount: days,
			ParentID: parentID,
		})
	}

	return events, nil
}

// IsStandalone returns true if this event is not a sub-event of a multi-day.
func (e SeriesEvent) IsStandalone() bool {
	return e.ParentID == 0 || e.ParentID == e.ID
}
