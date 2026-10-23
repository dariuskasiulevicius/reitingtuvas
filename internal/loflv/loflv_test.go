package loflv

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/dariuskasiulevicius/reitingtuvas/internal/domain"
)

// --- Settings ---

func TestParseLVSettings_BothEmpty(t *testing.T) {
	raw := map[string]string{}
	s, err := ParseLVSettings(raw)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if s != nil {
		t.Fatal("expected nil settings")
	}
}

func TestParseLVSettings_BothSet(t *testing.T) {
	raw := map[string]string{
		"LV šaltinio adresas": "https://example.com",
		"LV serijos ID":      "42",
	}
	s, err := ParseLVSettings(raw)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if s == nil || s.BaseURL != "https://example.com" || s.SeriesID != "42" {
		t.Fatalf("bad settings: %+v", s)
	}
	if !s.Enabled() {
		t.Fatal("expected enabled")
	}
}

func TestParseLVSettings_OnlyOne(t *testing.T) {
	raw := map[string]string{
		"LV šaltinio adresas": "https://example.com",
	}
	_, err := ParseLVSettings(raw)
	if err == nil {
		t.Fatal("expected error when only one key set")
	}
}

// --- API ---

func TestUnwrapResponse_HTML(t *testing.T) {
	body := `{"html":"<h1>Test</h1>","msg":""}`
	html, err := unwrapResponse([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if html != "<h1>Test</h1>" {
		t.Fatalf("got %q", html)
	}
}

func TestUnwrapResponse_Msg(t *testing.T) {
	body := `{"html":"","msg":"<p>fallback</p>"}`
	html, err := unwrapResponse([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if html != "<p>fallback</p>" {
		t.Fatalf("got %q", html)
	}
}

func TestUnwrapResponse_Empty(t *testing.T) {
	body := `{"html":"","msg":""}`
	_, err := unwrapResponse([]byte(body))
	if err == nil {
		t.Fatal("expected error on empty")
	}
}

func TestFormData(t *testing.T) {
	fd := formData(123, 1)
	if fd.Get("p") != "ev=123" {
		t.Fatalf("p=%q", fd.Get("p"))
	}
	if fd.Get("r") != "1" {
		t.Fatalf("r=%q", fd.Get("r"))
	}
}

// --- Series ---

func TestParseSeriesPage(t *testing.T) {
	f, err := os.Open("testdata/lv_series_page.html")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	events, err := ParseSeriesPage(f)
	if err != nil {
		t.Fatal(err)
	}

	if len(events) != 5 {
		t.Fatalf("events=%d, want 5", len(events))
	}

	// Sprint Race
	if events[0].ID != 101 || events[0].Name != "Sprint Race" {
		t.Errorf("event[0]=%+v", events[0])
	}
	if !events[0].IsStandalone() {
		t.Error("Sprint Race should be standalone")
	}

	// Relay
	if !events[2].IsRelay {
		t.Error("event[2] should be relay")
	}

	// Multi-day sub-events
	if events[3].ParentID != 400 || events[3].IsStandalone() {
		t.Errorf("event[3] multi-day: %+v", events[3])
	}
	if events[4].ParentID != 400 {
		t.Errorf("event[4] parentID=%d", events[4].ParentID)
	}
}

// --- Event ---

func TestParseEventResults(t *testing.T) {
	data, err := os.ReadFile("testdata/lv_event_single.json")
	if err != nil {
		t.Fatal(err)
	}

	html, err := unwrapResponse(data)
	if err != nil {
		t.Fatal(err)
	}

	blocks, err := ParseEventResults(html)
	if err != nil {
		t.Fatal(err)
	}

	if len(blocks) != 2 {
		t.Fatalf("blocks=%d, want 2", len(blocks))
	}

	s21 := blocks[0]
	if s21.ClassName != "S21" {
		t.Fatalf("class=%q", s21.ClassName)
	}
	if len(s21.Rows) != 3 {
		t.Fatalf("rows=%d, want 3", len(s21.Rows))
	}

	// 1st place
	r := s21.Rows[0]
	if r.Place != 1 || r.FirstName != "Janis" || r.LastName != "Berzins" {
		t.Errorf("row0=%+v", r)
	}
	if r.Time != 28*time.Minute+21*time.Second {
		t.Errorf("time=%v", r.Time)
	}
	if r.Status != "ok" {
		t.Errorf("status=%q", r.Status)
	}

	// Out of competition
	oc := s21.Rows[2]
	if oc.Status != "-" {
		t.Errorf("out-of-comp status=%q", oc.Status)
	}

	// V21
	if blocks[1].ClassName != "V21" || len(blocks[1].Rows) != 1 {
		t.Errorf("V21=%+v", blocks[1])
	}
}

func TestParseLVTime(t *testing.T) {
	cases := []struct {
		in  string
		out time.Duration
	}{
		{"00:28:21", 28*time.Minute + 21*time.Second},
		{"1:05:30", 1*time.Hour + 5*time.Minute + 30*time.Second},
		{"00:00:00", 0},
		{"dns", 0},
		{"", 0},
	}
	for _, c := range cases {
		if got := parseLVTime(c.in); got != c.out {
			t.Errorf("parseLVTime(%q)=%v, want %v", c.in, got, c.out)
		}
	}
}

func TestSplitName(t *testing.T) {
	cases := []struct {
		full, first, last string
	}{
		{"Janis Berzins", "Janis", "Berzins"},
		{"Anna Emilija Berzins", "Anna Emilija", "Berzins"},
		{"Solo", "Solo", ""},
		{"", "", ""},
	}
	for _, c := range cases {
		f, l := splitName(c.full)
		if f != c.first || l != c.last {
			t.Errorf("splitName(%q)=(%q,%q), want (%q,%q)", c.full, f, l, c.first, c.last)
		}
	}
}

// --- Multi-day ---

func TestParseMultiDayResults(t *testing.T) {
	data, err := os.ReadFile("testdata/lv_multiday_overall.json")
	if err != nil {
		t.Fatal(err)
	}

	html, err := unwrapResponse(data)
	if err != nil {
		t.Fatal(err)
	}

	blocks, err := ParseMultiDayResults(html)
	if err != nil {
		t.Fatal(err)
	}

	if len(blocks) != 1 {
		t.Fatalf("blocks=%d, want 1", len(blocks))
	}

	b := blocks[0]
	if b.ClassName != "M21" {
		t.Fatalf("class=%q", b.ClassName)
	}
	if b.DayCount != 2 {
		t.Fatalf("dayCount=%d, want 2", b.DayCount)
	}
	if len(b.Rows) != 3 {
		t.Fatalf("rows=%d, want 3", len(b.Rows))
	}

	r := b.Rows[0]
	if r.Place != 1 || r.LastName != "Berzins" || r.FirstName != "Janis" {
		t.Errorf("row0=%+v", r)
	}
	if r.TotalTime != 1*time.Hour+5*time.Minute+30*time.Second {
		t.Errorf("total=%v", r.TotalTime)
	}
	if len(r.Days) != 2 {
		t.Fatalf("days=%d", len(r.Days))
	}
	if r.Days[0].Time != 30*time.Minute+15*time.Second || r.Days[0].Place != 1 {
		t.Errorf("day1=%+v", r.Days[0])
	}
	if r.Days[1].Time != 35*time.Minute+15*time.Second || r.Days[1].Place != 2 {
		t.Errorf("day2=%+v", r.Days[1])
	}
}

// --- Relay ---

func TestParseRelayResults(t *testing.T) {
	data, err := os.ReadFile("testdata/lv_relay.json")
	if err != nil {
		t.Fatal(err)
	}

	html, err := unwrapResponse(data)
	if err != nil {
		t.Fatal(err)
	}

	blocks, err := ParseRelayResults(html)
	if err != nil {
		t.Fatal(err)
	}

	if len(blocks) != 1 {
		t.Fatalf("blocks=%d, want 1", len(blocks))
	}

	b := blocks[0]
	if b.ClassName != "Open" {
		t.Fatalf("class=%q", b.ClassName)
	}
	if len(b.Teams) != 2 {
		t.Fatalf("teams=%d, want 2", len(b.Teams))
	}

	t1 := b.Teams[0]
	if t1.Place != 1 || t1.TeamName != "OK Azimuts A" {
		t.Errorf("team1=%+v", t1)
	}
	if t1.TotalTime != 1*time.Hour+25*time.Minute+30*time.Second {
		t.Errorf("totalTime=%v", t1.TotalTime)
	}
	if len(t1.Legs) != 3 {
		t.Fatalf("legs=%d, want 3", len(t1.Legs))
	}

	leg1 := t1.Legs[0]
	if leg1.FirstName != "Janis" || leg1.LastName != "Berzins" {
		t.Errorf("leg1=%+v", leg1)
	}
	if leg1.Time != 28*time.Minute+10*time.Second {
		t.Errorf("leg1.time=%v", leg1.Time)
	}
}

func TestParseRelayTime(t *testing.T) {
	cases := []struct {
		in  string
		out time.Duration
	}{
		{"0:28:10,00", 28*time.Minute + 10*time.Second},
		{"1:25:30,00", 1*time.Hour + 25*time.Minute + 30*time.Second},
		{"0:00:00,00", 0},
	}
	for _, c := range cases {
		if got := parseRelayTime(c.in); got != c.out {
			t.Errorf("parseRelayTime(%q)=%v, want %v", c.in, got, c.out)
		}
	}
}

// --- Match ---

func TestParticipantIndex(t *testing.T) {
	participants := []domain.Participant{
		{Name: "Janis", Surname: "Berzins", SID: "1001"},
		{Name: "Maris", Surname: "Ozols", SID: "1002"},
		{Name: "Karlis", Surname: "Liepa", SID: ""}, // no SID → skipped
	}

	idx := newParticipantIndex(participants)

	// exact match
	p, ok := idx.Match("Janis", "Berzins")
	if !ok || p.SID != "1001" {
		t.Errorf("match Janis Berzins: ok=%v p=%+v", ok, p)
	}

	// case insensitive
	p, ok = idx.Match("JANIS", "BERZINS")
	if !ok || p.SID != "1001" {
		t.Errorf("match case insensitive: ok=%v", ok)
	}

	// no SID → not indexed
	_, ok = idx.Match("Karlis", "Liepa")
	if ok {
		t.Error("Karlis should not be found (no SID)")
	}

	// not found
	_, ok = idx.Match("Unknown", "Person")
	if ok {
		t.Error("unknown should not match")
	}
}

// --- Helpers ---

func TestLvEventID(t *testing.T) {
	if got := lvEventID(42); got != "lv:42" {
		t.Errorf("got %q", got)
	}
}

func TestMapLVStatus(t *testing.T) {
	cases := []struct{ in, out string }{
		{"ok", "ok"},
		{"dns", "dns"},
		{"dsq", "dsq"},
		{"", "ok"},
		{"mp", "mp"},
		{"weird", "weird"},
	}
	for _, c := range cases {
		if got := mapLVStatus(c.in); got != c.out {
			t.Errorf("mapLVStatus(%q)=%q, want %q", c.in, got, c.out)
		}
	}
}

func TestGroupEvents(t *testing.T) {
	events := []SeriesEvent{
		{ID: 1, ParentID: 0},    // standalone
		{ID: 2, ParentID: 2},    // standalone (self-ref)
		{ID: 3, ParentID: 100},  // multi-day child
		{ID: 4, ParentID: 100},  // multi-day child
	}
	standalone, multi := groupEvents(events)
	if len(standalone) != 2 {
		t.Errorf("standalone=%d, want 2", len(standalone))
	}
	if len(multi) != 1 || len(multi[100]) != 2 {
		t.Errorf("multi=%+v", multi)
	}
}

func TestFindEventLeaderTime(t *testing.T) {
	rows := []EventRow{
		{Place: 2, Time: 30 * time.Minute, Status: "ok"},
		{Place: 1, Time: 28 * time.Minute, Status: "ok"},
		{Place: 3, Time: 35 * time.Minute, Status: "ok"},
	}
	if got := findEventLeaderTime(rows); got != 28*time.Minute {
		t.Errorf("got %v", got)
	}
}

// Use json.Unmarshal to verify rawSeriesEvent nullable fields
func TestRawSeriesEventNullable(t *testing.T) {
	input := `{"id":"1","sacen":"Test","datums":"2025-01-01","stafete":null,"cik_dienas":"1","sacid":null}`
	var r rawSeriesEvent
	if err := json.Unmarshal([]byte(input), &r); err != nil {
		t.Fatal(err)
	}
	if r.Relay != nil {
		t.Error("relay should be nil")
	}
	if r.ParentID != nil {
		t.Error("parentID should be nil")
	}
}
