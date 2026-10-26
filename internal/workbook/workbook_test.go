package workbook

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dariuskasiulevicius/reitingtuvas/internal/domain"
	"github.com/dariuskasiulevicius/reitingtuvas/internal/report"
	"github.com/dariuskasiulevicius/reitingtuvas/internal/source"
)

func TestCreateTemplate_AndReadBack(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.xlsx")

	wb, created, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer wb.Close()

	if !created {
		t.Error("expected created=true for new file")
	}

	// Read settings — should have default keys
	settings, err := wb.ReadSettings()
	if err != nil {
		t.Fatal(err)
	}
	if settings["Pauzė tarp užklausų, ms"] != "200" {
		t.Errorf("default delay = %q", settings["Pauzė tarp užklausų, ms"])
	}

	// Participants should be empty
	participants, err := wb.ReadParticipants()
	if err != nil {
		t.Fatal(err)
	}
	if len(participants) != 0 {
		t.Errorf("expected 0 participants, got %d", len(participants))
	}
}

func TestRoundtrip_ParticipantsAndEvents(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.xlsx")

	// Create template
	wb, _, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}

	// Write settings
	f := wb.ExcelFile()
	f.SetCellValue("Nustatymai", "B1", source.FakeBaseURL)
	f.SetCellValue("Nustatymai", "B2", source.FakeSearch)
	f.SetCellValue("Nustatymai", "B3", source.FakePerson)
	f.SetCellValue("Nustatymai", "B4", source.FakeClass)
	f.SetCellValue("Nustatymai", "B5", source.FakeEvent)
	f.SetCellValue("Nustatymai", "B6", source.FakeProfile)
	f.SetCellValue("Nustatymai", "B7", "2026-01-01")
	f.SetCellValue("Nustatymai", "B8", "2026-12-31")

	// Write participants
	f.SetCellValue("Dalyviai", "A2", "Pavardenis")
	f.SetCellValue("Dalyviai", "B2", "Vardenis")

	wb.Save()
	wb.Close()

	// Re-open
	wb, created, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer wb.Close()

	if created {
		t.Error("should not be created on re-open")
	}

	// Read participants
	participants, err := wb.ReadParticipants()
	if err != nil {
		t.Fatal(err)
	}
	if len(participants) != 1 {
		t.Fatalf("got %d participants", len(participants))
	}
	if participants[0].Surname != "Pavardenis" {
		t.Errorf("surname = %q", participants[0].Surname)
	}

	// Write back with SID
	participants[0].SID = "123"
	participants[0].Status = "rastas"
	wb.WriteParticipants(participants)

	// Add event
	wb.AppendEvent(domain.Event{
		ID: "2026058", Name: "Testas", Date: time.Date(2026, 7, 25, 0, 0, 0, 0, time.UTC),
		Sport: "ob", DayCount: 2, Coefficient: 1.00,
	})

	// Add day
	wb.AppendDay(domain.Day{
		EventID: "2026058", Number: 1, Date: time.Date(2026, 7, 25, 0, 0, 0, 0, time.UTC),
	}, "Testas")

	// Read settings and parse
	rawSettings, _ := wb.ReadSettings()
	settings, err := ParseSettings(rawSettings)
	if err != nil {
		t.Fatal(err)
	}
	if settings.BaseURL != source.FakeBaseURL {
		t.Errorf("base url = %q", settings.BaseURL)
	}

	// Read coefficients
	coefs, _ := wb.ReadCoefficients()
	if coefs["2026058"] != 1.00 {
		t.Errorf("coefficient = %f", coefs["2026058"])
	}

	wb.Save()
}

func TestReportSheet_WritesAndOpens(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.xlsx")

	wb, _, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer wb.Close()

	stored := []domain.StoredResult{
		{SID: "100", EventID: "2026172", DayNumber: 1, Class: "V65", Status: "ok",
			Time: 45*time.Minute + 48*time.Second, LeaderTime: 45*time.Minute + 48*time.Second, Place: 1},
		{SID: "200", EventID: "2026058", DayNumber: 1, Class: "BM55", Status: "ok",
			Time: 1*time.Hour + 14*time.Minute + 36*time.Second, LeaderTime: 1*time.Hour + 50*time.Second, Place: 7},
		{SID: "200", EventID: "2026058", DayNumber: 2, Class: "BM55", Status: "ok",
			Time: 33*time.Minute + 27*time.Second, LeaderTime: 27*time.Minute + 15*time.Second, Place: 8},
	}

	participants := []domain.Participant{
		{Surname: "Pirmas", Name: "Vardas", SID: "100"},
		{Surname: "Antras", Name: "Vardas", SID: "200"},
	}

	events := map[string]domain.Event{
		"2026172": {ID: "2026172", Name: "Testas", Date: time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)},
		"2026058": {ID: "2026058", Name: "Dviejų dienų", Date: time.Date(2026, 7, 25, 0, 0, 0, 0, time.UTC)},
	}

	coefficients := map[string]float64{"2026058": 1.50}
	openClassCoef := 0.9
	topN := 8

	rating := domain.ComputeRating(stored, participants, coefficients, openClassCoef, topN)
	columns := report.BuildEventColumns(stored, events)

	sheetName, err := report.WriteReport(wb.ExcelFile(), rating, columns, coefficients, openClassCoef, topN, 2026, "")
	if err != nil {
		t.Fatal(err)
	}

	if sheetName == "" {
		t.Error("empty sheet name")
	}

	// Save and verify file exists
	wb.Save()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() < 1000 {
		t.Errorf("file too small: %d bytes", info.Size())
	}
}

func TestParseFlexDate(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"2026-01-01", "2026-01-01"},
		{"46023", "2026-01-01"},   // Excel serial number
		{"46387", "2026-12-31"},   // Excel serial number
		{"01-01-26", "2026-01-01"},
		{"1/1/2026", "2026-01-01"},
		{"01/01/2026", "2026-01-01"},
	}
	for _, tt := range tests {
		got, err := parseFlexDate(tt.input)
		if err != nil {
			t.Errorf("parseFlexDate(%q) error: %v", tt.input, err)
			continue
		}
		if got.Format("2006-01-02") != tt.want {
			t.Errorf("parseFlexDate(%q) = %s, want %s", tt.input, got.Format("2006-01-02"), tt.want)
		}
	}
}

func TestParseFlexDate_Errors(t *testing.T) {
	bad := []string{"", "hello", "2026"}
	for _, s := range bad {
		_, err := parseFlexDate(s)
		if err == nil {
			t.Errorf("parseFlexDate(%q) should fail", s)
		}
	}
}

func TestReadManualEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.xlsx")

	wb, _, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer wb.Close()

	sheet := "Rankinis įvedimas"
	f := wb.ExcelFile()

	// Row 2: normal single-day result
	f.SetCellValue(sheet, "A2", "Kalnų taurė")
	f.SetCellValue(sheet, "B2", "2026-09-15")
	f.SetCellValue(sheet, "C2", "Pavardenis Vardenis")
	f.SetCellValue(sheet, "D2", "1:23:45")
	f.SetCellValue(sheet, "E2", "1:15:30")

	// Row 3: same event, day 2
	f.SetCellValue(sheet, "A3", "Kalnų taurė")
	f.SetCellValue(sheet, "B3", "2026-09-16")
	f.SetCellValue(sheet, "C3", "Pavardenis Vardenis")
	f.SetCellValue(sheet, "D3", "1:30:00")
	f.SetCellValue(sheet, "E3", "1:20:00")

	// Row 4: different event, open class
	f.SetCellValue(sheet, "A4", "Miško bėgimas")
	f.SetCellValue(sheet, "B4", "2026-10-01")
	f.SetCellValue(sheet, "C4", "Vardenis Pavardenis") // reversed order
	f.SetCellValue(sheet, "D4", "0:45:00")
	f.SetCellValue(sheet, "E4", "0:40:00")
	f.SetCellValue(sheet, "F4", "taip") // open class

	// Row 5: dsq entry
	f.SetCellValue(sheet, "A5", "Kalnų taurė")
	f.SetCellValue(sheet, "B5", "2026-09-15")
	f.SetCellValue(sheet, "C5", "Kitenis Antanas")
	f.SetCellValue(sheet, "D5", "1:10:00")
	f.SetCellValue(sheet, "E5", "1:15:30")
	f.SetCellValue(sheet, "G5", "taip") // dsq

	// Row 6: unknown participant
	f.SetCellValue(sheet, "A6", "Kalnų taurė")
	f.SetCellValue(sheet, "B6", "2026-09-15")
	f.SetCellValue(sheet, "C6", "Nežinomas Asmuo")
	f.SetCellValue(sheet, "D6", "2:00:00")
	f.SetCellValue(sheet, "E6", "1:15:30")

	participants := []domain.Participant{
		{Surname: "Pavardenis", Name: "Vardenis", SID: "101"},
		{Surname: "Kitenis", Name: "Antanas", SID: "102"},
	}

	results, events, days, warnings := wb.ReadManualEntries(participants)

	// Should have 4 results (row 6 skipped — unknown participant)
	if len(results) != 4 {
		t.Fatalf("results=%d, want 4", len(results))
	}

	// Check name resolution — both orderings work
	if results[0].SID != "101" {
		t.Errorf("row2 SID=%s, want 101", results[0].SID)
	}
	if results[2].SID != "101" { // reversed name
		t.Errorf("row4 SID=%s, want 101 (reversed name)", results[2].SID)
	}

	// Check event ID generation
	wantEventID := "manual:2026-09-15:Kalnų taurė"
	if results[0].EventID != wantEventID {
		t.Errorf("eventID=%s, want %s", results[0].EventID, wantEventID)
	}

	// Check day number derivation (2 days for Kalnų taurė)
	if results[0].DayNumber != 1 {
		t.Errorf("row2 day=%d, want 1", results[0].DayNumber)
	}
	if results[1].DayNumber != 2 {
		t.Errorf("row3 day=%d, want 2", results[1].DayNumber)
	}

	// Check time parsing
	if results[0].Time != 1*time.Hour+23*time.Minute+45*time.Second {
		t.Errorf("row2 time=%v", results[0].Time)
	}
	if results[0].LeaderTime != 1*time.Hour+15*time.Minute+30*time.Second {
		t.Errorf("row2 leader=%v", results[0].LeaderTime)
	}

	// Check open class
	if results[2].Class != "open" {
		t.Errorf("row4 class=%q, want open", results[2].Class)
	}

	// Check dsq
	if results[3].Status != "dsq" {
		t.Errorf("row5 status=%q, want dsq", results[3].Status)
	}
	if results[0].Status != "ok" {
		t.Errorf("row2 status=%q, want ok", results[0].Status)
	}

	// Check events generated
	if len(events) != 2 {
		t.Fatalf("events=%d, want 2", len(events))
	}
	// Kalnų taurė should have DayCount=2
	for _, ev := range events {
		if ev.Name == "Kalnų taurė" && ev.DayCount != 2 {
			t.Errorf("Kalnų taurė DayCount=%d, want 2", ev.DayCount)
		}
		if ev.Coefficient != 1.00 {
			t.Errorf("event %s coef=%f, want 1.00", ev.Name, ev.Coefficient)
		}
	}

	// Check days generated
	if len(days) != 3 { // 2 for Kalnų taurė + 1 for Miško bėgimas
		t.Errorf("days=%d, want 3", len(days))
	}

	// Check warning for unknown participant
	if len(warnings) != 1 {
		t.Fatalf("warnings=%d, want 1", len(warnings))
	}
	if warnings[0] != `Rankinis: dalyvis "Nežinomas Asmuo" nerastas` {
		t.Errorf("warning=%q", warnings[0])
	}
}

func TestReadManualEntries_Empty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.xlsx")

	wb, _, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer wb.Close()

	results, events, days, warnings := wb.ReadManualEntries(nil)
	if len(results) != 0 || len(events) != 0 || len(days) != 0 || len(warnings) != 0 {
		t.Errorf("empty sheet should return all nils/empty")
	}
}

func TestBackup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.xlsx")

	wb, _, _ := Open(path)
	wb.Save()
	wb.Close()

	err := Backup(path)
	if err != nil {
		t.Fatal(err)
	}

	backupPath := filepath.Join(dir, "test.backup.xlsx")
	if _, err := os.Stat(backupPath); err != nil {
		t.Errorf("backup not found: %v", err)
	}
}
