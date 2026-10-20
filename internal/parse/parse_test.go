package parse

import (
	"os"
	"testing"

	"github.com/dariuskasiulevicius/reitingtuvas/internal/source"
)

func openFixture(t *testing.T, name string) *os.File {
	t.Helper()
	f, err := os.Open("../../testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Close() })
	return f
}

func profileTmpl(t *testing.T) *source.Template {
	t.Helper()
	tmpl, err := source.NewTemplate(source.FakeProfile)
	if err != nil {
		t.Fatal(err)
	}
	return tmpl
}

// --- Search tests ---

func TestParseSearch_OneResult(t *testing.T) {
	f := openFixture(t, "search_one.html")
	candidates, err := ParseSearch(f, profileTmpl(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 {
		t.Fatalf("got %d candidates, want 1", len(candidates))
	}
	c := candidates[0]
	if c.DisplayName != "Pavardenis Darius" {
		t.Errorf("name = %q", c.DisplayName)
	}
	if c.SID != "5890" {
		t.Errorf("sid = %q, want 5890", c.SID)
	}
	if c.EventCount != 24 {
		t.Errorf("events = %d, want 24", c.EventCount)
	}
}

func TestParseSearch_ManyResults(t *testing.T) {
	f := openFixture(t, "search_many.html")
	candidates, err := ParseSearch(f, profileTmpl(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 4 {
		t.Fatalf("got %d candidates, want 4", len(candidates))
	}
	// Fourth person has no profile (no SID)
	if candidates[3].SID != "" {
		t.Errorf("person without profile should have empty SID, got %q", candidates[3].SID)
	}
	if candidates[3].DisplayName != "Pavardenis Mykolas" {
		t.Errorf("name = %q", candidates[3].DisplayName)
	}
}

func TestParseSearch_NoResults(t *testing.T) {
	f := openFixture(t, "search_none.html")
	candidates, err := ParseSearch(f, profileTmpl(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 0 {
		t.Fatalf("got %d candidates, want 0", len(candidates))
	}
}

// --- Person results tests ---

func TestParsePersonResults(t *testing.T) {
	f := openFixture(t, "person_results.html")
	tmpl := source.FakeTemplates()
	rows, err := ParsePersonResults(f, tmpl.Class, tmpl.Event)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 5 {
		t.Fatalf("got %d rows, want 5", len(rows))
	}

	// Row 0: single-day
	r := rows[0]
	if r.EventID != "2026172" {
		t.Errorf("row0 event id = %q", r.EventID)
	}
	if r.EventName != "Miesto čempionatas" {
		t.Errorf("row0 event = %q", r.EventName)
	}
	if r.Place != 1 {
		t.Errorf("row0 place = %d", r.Place)
	}
	if r.Result != "45:48" {
		t.Errorf("row0 result = %q", r.Result)
	}
	if r.IsRelay {
		t.Error("row0 should not be relay")
	}
	if len(r.DayLabels) != 0 {
		t.Errorf("row0 day labels = %v", r.DayLabels)
	}

	// Row 1: 2-day event
	r = rows[1]
	if r.EventID != "2026058" {
		t.Errorf("row1 event id = %q", r.EventID)
	}
	if r.Place != 6 {
		t.Errorf("row1 place = %d", r.Place)
	}
	if r.Result != "1:48:03" {
		t.Errorf("row1 result = %q", r.Result)
	}
	if len(r.DayLabels) != 2 {
		t.Fatalf("row1 day labels = %v, want 2", r.DayLabels)
	}
	if r.DayLabels[0] != "1 day" || r.DayLabels[1] != "2 day" {
		t.Errorf("row1 labels = %v", r.DayLabels)
	}
	if len(r.DayTimes) != 2 {
		t.Fatalf("row1 day times = %v, want 2", r.DayTimes)
	}

	// Row 2: 3-day with dsq
	r = rows[2]
	if r.EventID != "2026094" {
		t.Errorf("row2 event id = %q", r.EventID)
	}
	if r.Place != 0 {
		t.Errorf("row2 place = %d, want 0 (empty)", r.Place)
	}
	if len(r.DayLabels) != 3 {
		t.Fatalf("row2 labels = %v", r.DayLabels)
	}
	if len(r.DayTimes) < 3 {
		t.Fatalf("row2 times = %v, want 3", r.DayTimes)
	}

	// Row 3: dsq total
	r = rows[3]
	if r.Result != "dsq" {
		t.Errorf("row3 result = %q, want dsq", r.Result)
	}

	// Row 4: relay
	r = rows[4]
	if !r.IsRelay {
		t.Error("row4 should be relay")
	}
	if r.EventID != "2026111" {
		t.Errorf("row4 event id = %q", r.EventID)
	}
}

// --- Class page tests ---

func TestParseClassPage_SingleDay(t *testing.T) {
	f := openFixture(t, "class_single_day.html")
	page, err := ParseClassPage(f, profileTmpl(t))
	if err != nil {
		t.Fatal(err)
	}
	if page.EventName != "Miesto čempionatas" {
		t.Errorf("event = %q", page.EventName)
	}
	if len(page.DayNames) != 0 {
		t.Errorf("day names = %v, want empty", page.DayNames)
	}
	if len(page.Rows) != 4 {
		t.Fatalf("rows = %d, want 4", len(page.Rows))
	}

	// Leader (place 1)
	leader := page.Rows[0]
	if leader.SID != "772" {
		t.Errorf("leader sid = %q", leader.SID)
	}
	if leader.Place != 1 {
		t.Errorf("leader place = %d", leader.Place)
	}
	if leader.Total != "45:48" {
		t.Errorf("leader total = %q", leader.Total)
	}

	// DSQ person (no place)
	dsq := page.Rows[3]
	if dsq.Place != 0 {
		t.Errorf("dsq place = %d, want 0", dsq.Place)
	}
	if dsq.Total != "dsq" {
		t.Errorf("dsq total = %q", dsq.Total)
	}
}

func TestParseClassPage_MultiDay(t *testing.T) {
	f := openFixture(t, "class_multi_day.html")
	page, err := ParseClassPage(f, profileTmpl(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(page.DayNames) != 2 {
		t.Fatalf("day names = %v, want 2", page.DayNames)
	}
	if page.DayNames[0] != "Vidutinė trasa" || page.DayNames[1] != "Sprinto trasa" {
		t.Errorf("day names = %v", page.DayNames)
	}
	if len(page.Rows) != 4 {
		t.Fatalf("rows = %d, want 4", len(page.Rows))
	}

	// Check per-day data for first competitor
	r := page.Rows[0]
	if len(r.PerDay) != 2 {
		t.Fatalf("perday = %v", r.PerDay)
	}

	// Parse day 1 time+place
	timeStr, place := ParseDayTimeAndPlace(r.PerDay[0])
	if timeStr != "1:00:50" || place != 1 {
		t.Errorf("day1 = %q place=%d", timeStr, place)
	}

	// Participant at place 6
	p := page.Rows[3]
	if p.SID != "772" {
		t.Errorf("sid = %q", p.SID)
	}
	if p.Total != "1:48:03" {
		t.Errorf("total = %q", p.Total)
	}
}

func TestParseClassPage_3DayDsq(t *testing.T) {
	f := openFixture(t, "class_3day_dsq.html")
	page, err := ParseClassPage(f, profileTmpl(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(page.DayNames) != 3 {
		t.Fatalf("day names = %v, want 3", page.DayNames)
	}
	if len(page.Rows) != 3 {
		t.Fatalf("rows = %d, want 3", len(page.Rows))
	}

	// Participant with dsq on day 3
	dsqRow := page.Rows[2]
	if dsqRow.SID != "772" {
		t.Errorf("sid = %q", dsqRow.SID)
	}
	if dsqRow.Place != 0 {
		t.Errorf("place = %d, want 0", dsqRow.Place)
	}
	if dsqRow.Total != "" {
		t.Errorf("total = %q, want empty", dsqRow.Total)
	}
	if len(dsqRow.PerDay) != 3 {
		t.Fatalf("perday = %v", dsqRow.PerDay)
	}
	// Day 3 should be dsq
	if dsqRow.PerDay[2] != "dsq" {
		t.Errorf("day3 = %q, want dsq", dsqRow.PerDay[2])
	}
}

// --- Relay tests ---

func TestParseRelayClassPage(t *testing.T) {
	f := openFixture(t, "relay_class.html")
	teams, err := ParseRelayClassPage(f, profileTmpl(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(teams) != 2 {
		t.Fatalf("teams = %d, want 2", len(teams))
	}

	// Team 1
	team1 := teams[0]
	if team1.Place != 1 {
		t.Errorf("team1 place = %d", team1.Place)
	}
	if team1.TeamName != "Pirma komanda, Vilnius" {
		t.Errorf("team1 name = %q", team1.TeamName)
	}
	if team1.TotalTime != "3:15:15" {
		t.Errorf("team1 total = %q", team1.TotalTime)
	}
	if len(team1.Legs) != 5 {
		t.Fatalf("team1 legs = %d, want 5", len(team1.Legs))
	}

	// Leg 5 of team 1 (fastest on leg 5)
	leg5 := team1.Legs[4]
	if leg5.LegNumber != 5 {
		t.Errorf("leg number = %d", leg5.LegNumber)
	}
	if leg5.SID != "19420" {
		t.Errorf("leg5 sid = %q", leg5.SID)
	}
	if leg5.Time != "38:45" {
		t.Errorf("leg5 time = %q", leg5.Time)
	}
	if leg5.Place != 1 {
		t.Errorf("leg5 place = %d", leg5.Place)
	}

	// Team 2, leg 5 = our participant
	team2 := teams[1]
	if len(team2.Legs) < 3 {
		t.Fatalf("team2 legs = %d", len(team2.Legs))
	}
	// Find leg 5 (should be last in team 2's legs)
	var found bool
	for _, leg := range team2.Legs {
		if leg.SID == "772" {
			found = true
			if leg.LegNumber != 5 {
				t.Errorf("our participant leg = %d, want 5", leg.LegNumber)
			}
			if leg.Time != "55:40" {
				t.Errorf("our participant time = %q", leg.Time)
			}
			if leg.Place != 9 {
				t.Errorf("our participant place = %d, want 9", leg.Place)
			}
		}
	}
	if !found {
		t.Error("participant 772 not found in team 2")
	}
}

// --- ParseDayTimeAndPlace ---

func TestParseDayTimeAndPlace(t *testing.T) {
	tests := []struct {
		input     string
		wantTime  string
		wantPlace int
	}{
		{"1:14:36( 7)", "1:14:36", 7},
		{"33:27( 8)", "33:27", 8},
		{"1:00:50( 1)", "1:00:50", 1},
		{"dsq", "dsq", 0},
		{"45:48", "45:48", 0},
		{"", "", 0},
	}
	for _, tt := range tests {
		timeStr, place := ParseDayTimeAndPlace(tt.input)
		if timeStr != tt.wantTime || place != tt.wantPlace {
			t.Errorf("ParseDayTimeAndPlace(%q) = (%q, %d), want (%q, %d)",
				tt.input, timeStr, place, tt.wantTime, tt.wantPlace)
		}
	}
}
