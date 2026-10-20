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

	results := []domain.EventResult{
		{
			Participant: domain.Participant{Surname: "Pirmas", Name: "Vardas", SID: "100"},
			Event: domain.Event{
				ID: "2026172", Name: "Testas", Date: time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC),
				Sport: "sport ob", DayCount: 1, Coefficient: 1.00,
			},
			Class:        "V65",
			Result:       45*time.Minute + 48*time.Second,
			Place:        1,
			LeaderResult: 45*time.Minute + 48*time.Second,
			Days: []domain.DayResult{
				{DayNumber: 1, Class: "V65", Status: "ok",
					Time: 45*time.Minute + 48*time.Second, Place: 1,
					LeaderTime: 45*time.Minute + 48*time.Second},
			},
		},
		{
			Participant: domain.Participant{Surname: "Antras", Name: "Vardas", SID: "200"},
			Event: domain.Event{
				ID: "2026058", Name: "Dviejų dienų", Date: time.Date(2026, 7, 25, 0, 0, 0, 0, time.UTC),
				Sport: "sport ob", DayCount: 2, Coefficient: 1.50,
			},
			Class:        "BM55",
			Result:       1*time.Hour + 48*time.Minute + 3*time.Second,
			Place:        6,
			LeaderResult: 1*time.Hour + 28*time.Minute + 5*time.Second,
			Days: []domain.DayResult{
				{DayNumber: 1, Class: "BM55", Status: "ok",
					Time: 1*time.Hour + 14*time.Minute + 36*time.Second, Place: 7,
					LeaderTime: 1*time.Hour + 50*time.Second},
				{DayNumber: 2, Class: "BM55", Status: "ok",
					Time: 33*time.Minute + 27*time.Second, Place: 8,
					LeaderTime: 27*time.Minute + 15*time.Second},
			},
		},
	}

	coefficients := map[string]float64{"2026058": 1.50}
	dayDates := map[string]time.Time{}

	sheetName, err := report.WriteReport(wb.ExcelFile(), results, coefficients, dayDates)
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
