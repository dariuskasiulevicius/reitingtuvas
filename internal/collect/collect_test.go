package collect

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/dariuskasiulevicius/reitingtuvas/internal/domain"
	"github.com/dariuskasiulevicius/reitingtuvas/internal/parse"
	"github.com/dariuskasiulevicius/reitingtuvas/internal/source"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("../../testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// fakeServer serves fixtures based on path matching the fake templates.
func fakeServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path + "?" + r.URL.RawQuery

		switch {
		case contains(path, "/p/find?q="):
			w.Write(fixture(t, "search_one.html"))
		case contains(path, "/p/find?person=772"):
			w.Write(fixture(t, "person_results.html"))
		case contains(path, "/e/2026172/class/V65?day=1"):
			w.Write(fixture(t, "class_single_day.html"))
		case contains(path, "/e/2026058/class/BM55?day=0"):
			w.Write(fixture(t, "class_multi_day.html"))
		case contains(path, "/e/2026094/class/DV65?day=1"):
			w.Write(fixture(t, "class_3day_day1.html"))
		case contains(path, "/e/2026094/class/DV65?day=2"):
			w.Write(fixture(t, "class_3day_day2.html"))
		case contains(path, "/e/2026094/class/DV65?day=3"):
			w.Write(fixture(t, "class_3day_day3.html"))
		case contains(path, "/e/2026115/class/DV65?day=1"):
			// Single day dsq event — return a minimal single-day page
			w.Write(fixture(t, "class_single_day.html"))
		case contains(path, "/e/2026111/class/X-MAZ?day=1"):
			w.Write(fixture(t, "relay_class.html"))
		default:
			t.Logf("unmatched request: %s", path)
			w.WriteHeader(404)
		}
	}))
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && stringContains(s, sub))
}

func stringContains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestCollector_Resolve(t *testing.T) {
	srv := fakeServer(t)
	defer srv.Close()

	tmpls, _ := source.NewTemplates(
		source.FakeSearch, source.FakePerson,
		source.FakeClass, source.FakeEvent, source.FakeProfile,
	)
	settings := &source.Settings{
		BaseURL:    srv.URL,
		Templates:  tmpls,
		PeriodFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		PeriodTo:   time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		Delay:      0,
	}

	client := source.NewClient(srv.URL, 0, nil, 14, "test")
	coll := NewCollector(client, settings, nil)

	participants := []domain.Participant{
		{Surname: "Pavardenis", Name: "Darius"},               // will be resolved
		{Surname: "Jau", Name: "Žinomas", SID: "772"},         // already has SID
	}

	result := coll.Resolve(participants)

	if result[0].SID != "5890" {
		t.Errorf("resolved SID = %q, want 5890", result[0].SID)
	}
	if result[0].Status != "rastas" {
		t.Errorf("status = %q, want rastas", result[0].Status)
	}
	if result[1].Status != "rastas" {
		t.Errorf("pre-filled status = %q, want rastas", result[1].Status)
	}
}

func TestCollector_Collect(t *testing.T) {
	srv := fakeServer(t)
	defer srv.Close()

	tmpls, _ := source.NewTemplates(
		source.FakeSearch, source.FakePerson,
		source.FakeClass, source.FakeEvent, source.FakeProfile,
	)
	settings := &source.Settings{
		BaseURL:    srv.URL,
		Templates:  tmpls,
		PeriodFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		PeriodTo:   time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		Delay:      0,
	}

	client := source.NewClient(srv.URL, 0, nil, 14, "test")
	coll := NewCollector(client, settings, nil)

	participants := []domain.Participant{
		{Surname: "Pavardenis", Name: "Rimvydas", SID: "772", Status: "rastas"},
	}

	results, warnings := coll.Collect(participants)

	if len(warnings) > 0 {
		t.Logf("warnings: %v", warnings)
	}

	if len(results) != 5 {
		t.Fatalf("got %d results, want 5", len(results))
	}

	// Check single-day event (2026172)
	var singleDay *domain.EventResult
	for i := range results {
		if results[i].Event.ID == "2026172" {
			singleDay = &results[i]
			break
		}
	}
	if singleDay == nil {
		t.Fatal("single-day event 2026172 not found")
	}
	if singleDay.Place != 1 {
		t.Errorf("single-day place = %d, want 1", singleDay.Place)
	}
	if len(singleDay.Days) != 1 {
		t.Fatalf("single-day has %d days", len(singleDay.Days))
	}
	if singleDay.Days[0].Status != "ok" {
		t.Errorf("day status = %q", singleDay.Days[0].Status)
	}
	if singleDay.Days[0].LeaderTime == 0 {
		t.Error("day leader time should be set")
	}

	// Check multi-day event (2026058)
	var multiDay *domain.EventResult
	for i := range results {
		if results[i].Event.ID == "2026058" {
			multiDay = &results[i]
			break
		}
	}
	if multiDay == nil {
		t.Fatal("multi-day event 2026058 not found")
	}
	if len(multiDay.Days) != 2 {
		t.Fatalf("multi-day has %d days, want 2", len(multiDay.Days))
	}
	if multiDay.LeaderResult == 0 {
		t.Error("leader result should be non-zero for multi-day")
	}

	// Check 3-day per-day event (2026094)
	var perDay *domain.EventResult
	for i := range results {
		if results[i].Event.ID == "2026094" {
			perDay = &results[i]
			break
		}
	}
	if perDay == nil {
		t.Fatal("3-day event 2026094 not found")
	}
	if len(perDay.Days) != 3 {
		t.Fatalf("3-day has %d days", len(perDay.Days))
	}
	// Day 1: 30:55, place 3, leader 28:00
	if perDay.Days[0].Status != "ok" {
		t.Errorf("day 1 status = %q, want ok", perDay.Days[0].Status)
	}
	if perDay.Days[0].Time != 30*time.Minute+55*time.Second {
		t.Errorf("day 1 time = %v", perDay.Days[0].Time)
	}
	if perDay.Days[0].Place != 3 {
		t.Errorf("day 1 place = %d, want 3", perDay.Days[0].Place)
	}
	if perDay.Days[0].LeaderTime != 28*time.Minute {
		t.Errorf("day 1 leader = %v, want 28m", perDay.Days[0].LeaderTime)
	}
	// Day 2: 55:16, place 7, leader 50:00
	if perDay.Days[1].Status != "ok" {
		t.Errorf("day 2 status = %q, want ok", perDay.Days[1].Status)
	}
	if perDay.Days[1].Time != 55*time.Minute+16*time.Second {
		t.Errorf("day 2 time = %v", perDay.Days[1].Time)
	}
	if perDay.Days[1].Place != 7 {
		t.Errorf("day 2 place = %d, want 7", perDay.Days[1].Place)
	}
	if perDay.Days[1].LeaderTime != 50*time.Minute {
		t.Errorf("day 2 leader = %v, want 50m", perDay.Days[1].LeaderTime)
	}
	// Day 3: dsq, leader 45:00
	if perDay.Days[2].Status != "dsq" {
		t.Errorf("day 3 status = %q, want dsq", perDay.Days[2].Status)
	}
	if perDay.Days[2].Time != 0 {
		t.Error("dsq day should have zero time")
	}
	if perDay.Days[2].LeaderTime != 45*time.Minute {
		t.Errorf("day 3 leader = %v, want 45m", perDay.Days[2].LeaderTime)
	}

	// Check relay event (2026111)
	var relay *domain.EventResult
	for i := range results {
		if results[i].Event.ID == "2026111" {
			relay = &results[i]
			break
		}
	}
	if relay == nil {
		t.Fatal("relay event 2026111 not found")
	}
	if !relay.Event.IsRelay {
		t.Error("should be marked as relay")
	}
	if relay.RelayLeg != 5 {
		t.Errorf("relay leg = %d, want 5", relay.RelayLeg)
	}
	if relay.Days[0].LeaderTime == 0 {
		t.Error("relay leader time should be set")
	}
}

func TestFindLeaderTime_Fallback(t *testing.T) {
	// When no row has Place == 1 (e.g. place parsing failed),
	// findLeaderTime should fall back to shortest time.
	rows := []parse.ClassCompetitor{
		{Place: 0, Total: "1:09:55"}, // actual leader, but place not parsed
		{Place: 0, Total: "1:12:40"},
		{Place: 0, Total: "1:17:30"},
		{Place: 0, Total: "dsq"},
	}

	got := findLeaderTime(rows)
	want := 1*time.Hour + 9*time.Minute + 55*time.Second
	if got != want {
		t.Errorf("findLeaderTime fallback = %v, want %v", got, want)
	}
}

func TestFindLeaderTime_ExactPlace(t *testing.T) {
	rows := []parse.ClassCompetitor{
		{Place: 1, Total: "1:09:55"},
		{Place: 2, Total: "1:12:40"},
	}

	got := findLeaderTime(rows)
	want := 1*time.Hour + 9*time.Minute + 55*time.Second
	if got != want {
		t.Errorf("findLeaderTime exact = %v, want %v", got, want)
	}
}
