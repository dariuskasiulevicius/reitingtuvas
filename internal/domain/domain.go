// Package domain contains the core types from CONTEXT.md.
// Pure data and computations, no I/O.
package domain

import "time"

// Participant is a tracked person on the internal list.
type Participant struct {
	Surname string
	Name    string
	SID     string // person id in the results database; empty until resolved
	Status  string // resolution status: "rastas", "nerastas", "keli kandidatai: ..."
}

// Event is one competition identified by its own id.
type Event struct {
	ID          string
	Name        string
	Date        time.Time // Event Date (start date)
	Sport       string
	IsRelay     bool
	DayCount    int
	Coefficient float64 // default 1.00
}

// Day is one stage of an Event.
type Day struct {
	EventID string
	Number  int       // 1-based
	Name    string    // e.g. "Vidutinė trasa"
	Date    time.Time // estimated Event Date + (Number-1), correctable
}

// DayResult holds one participant's outcome on one Day.
type DayResult struct {
	DayNumber  int
	Class      string
	Status     string        // "ok", "dsq", "dns", "dnf", "mp", "not entered", ...
	Time       time.Duration // zero when Status != "ok"
	Place      int           // zero when unknown
	LeaderTime time.Duration // Day Leader Result; zero when unknown
}

// EventResult is one Participant × Event row for the Report.
type EventResult struct {
	Participant Participant
	Event       Event
	Class       string        // primary class (may differ per Day)
	RelayLeg    int           // 0 when not a relay
	Result      time.Duration // Participant Result: sum of ok Day times
	Place       int           // official total place; 0 when not published
	LeaderResult time.Duration // sum of Day Leader Results
	Days        []DayResult
	Notes       string // warnings
}

// SearchCandidate is one hit from a name search.
type SearchCandidate struct {
	DisplayName string
	SID         string // empty when the person has no profile
	EventCount  int
}

// StoredResult is one row in the Rezultatai or Rankinis įvedimas sheet.
type StoredResult struct {
	SID             string
	EventID         string
	ParticipantName string
	EventName       string
	DayNumber       int
	Class           string
	Time            time.Duration
	LeaderTime      time.Duration
	Status          string // "ok", "dsq", "dns", ...
	Place           int
	Link            string
}
