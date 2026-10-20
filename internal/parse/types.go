// Package parse converts HTML pages into raw data rows.
// Parsers fail loudly on unexpected structure rather than guessing.
package parse

import "time"

// SearchCandidate is one person found by name search.
type SearchCandidate struct {
	DisplayName string
	SID         string // empty when the person has no profile
	EventCount  int
}

// PersonResultRow is one event row from the person results page.
type PersonResultRow struct {
	Date      time.Time
	EventID   string
	EventName string
	Sport     string // icon alt text, e.g. "sport ob"
	Class     string
	ClassHref string // href of the class link (contains day selector)
	IsRelay   bool
	Place     int    // total place; 0 when empty
	Result    string // total result text (time or status)
	DayLabels []string
	DayTimes  []string // per day: "1:14:36( 7)" or "dsq" etc.
}

// ClassCompetitor is one row in a class results page.
type ClassCompetitor struct {
	SID   string
	Name  string
	Place int
	Total string // time or status
	// PerDay is populated only for multi-day (total) view.
	// Key: day column index (0-based), Value: "time( place)" or status.
	PerDay []string
}

// ClassPage is the parsed content of a class results page.
type ClassPage struct {
	EventName string
	DayNames  []string          // empty for single-day
	Rows      []ClassCompetitor
}

// RelayTeam is one team in a relay class results page.
type RelayTeam struct {
	TeamName  string
	Place     int
	TotalTime string
	Legs      []RelayLeg
}

// RelayLeg is one leg within a relay team.
type RelayLeg struct {
	LegNumber int
	SID       string
	Name      string
	Time      string // leg time or status
	Place     int
}
