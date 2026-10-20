package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dariuskasiulevicius/reitingtuvas/internal/collect"
	"github.com/dariuskasiulevicius/reitingtuvas/internal/domain"
	"github.com/dariuskasiulevicius/reitingtuvas/internal/report"
	"github.com/dariuskasiulevicius/reitingtuvas/internal/source"
	"github.com/dariuskasiulevicius/reitingtuvas/internal/workbook"
)

// version is set by the linker at build time.
var version = "dev"

const workbookName = "reitingtuvas.xlsx"

func main() {
	// Dev flags
	wbPathFlag := flag.String("workbook", "", "path to workbook (overrides exe dir)")
	noPause := flag.Bool("no-pause", false, "don't wait for Enter at the end")
	flag.Parse()

	dir, err := execDir()
	if err != nil {
		exitWithError(fmt.Sprintf("Nepavyko nustatyti programos vietos: %v", err), *noPause)
	}

	wbPath := filepath.Join(dir, workbookName)
	if *wbPathFlag != "" {
		wbPath = *wbPathFlag
	}

	fmt.Printf("reitingtuvas %s\n", version)

	// Step 1: Lock check
	if _, err := os.Stat(wbPath); err == nil {
		if err := workbook.CheckLock(wbPath); err != nil {
			exitWithError(err.Error(), *noPause)
		}
	}

	// Step 2: Open workbook (or create template)
	wb, created, err := workbook.Open(wbPath)
	if err != nil {
		exitWithError(fmt.Sprintf("Knyga: %v", err), *noPause)
	}
	defer wb.Close()

	if created {
		fmt.Printf("Sukurta nauja knyga: %s\n", wbPath)
		fmt.Println("Užpildykite Nustatymai ir Dalyviai lapus, tada paleiskite iš naujo.")
		waitForEnter(*noPause)
		return
	}

	// Step 3: Read settings
	rawSettings, err := wb.ReadSettings()
	if err != nil {
		exitWithError(fmt.Sprintf("Nustatymai: %v", err), *noPause)
	}

	settings, err := workbook.ParseSettings(rawSettings)
	if err != nil {
		exitWithError(fmt.Sprintf("Nustatymai: %v", err), *noPause)
	}

	fmt.Printf("Laikotarpis: %s — %s\n",
		settings.PeriodFrom.Format("2006-01-02"),
		settings.PeriodTo.Format("2006-01-02"),
	)

	// Step 4: Read participants
	participants, err := wb.ReadParticipants()
	if err != nil {
		exitWithError(fmt.Sprintf("Dalyviai: %v", err), *noPause)
	}
	if len(participants) == 0 {
		exitWithError("Dalyvių sąrašas tuščias.", *noPause)
	}
	fmt.Printf("Dalyviai: %d\n", len(participants))

	// Step 5: Set up cache + client
	cacheDir := filepath.Join(dir, "cache")
	cache, err := source.NewCache(cacheDir)
	if err != nil {
		exitWithError(fmt.Sprintf("Talpykla: %v", err), *noPause)
	}

	client := source.NewClient(settings.BaseURL, settings.Delay, cache, settings.CacheFinalDays, version)

	// Progress callback
	progress := func(current, total int, desc string) {
		fmt.Printf("\r[%d/%d] %s          ", current, total, desc)
	}

	coll := collect.NewCollector(client, settings, progress)

	// Step 6: Resolve participants
	fmt.Println("Ieškoma dalyvių...")
	participants = coll.Resolve(participants)

	resolved := 0
	for _, p := range participants {
		if p.SID != "" {
			resolved++
		}
	}
	fmt.Printf("\nRasta: %d/%d\n", resolved, len(participants))

	// Write back SIDs and statuses
	if err := wb.WriteParticipants(participants); err != nil {
		fmt.Printf("Įspėjimas: nepavyko atnaujinti dalyvių: %v\n", err)
	}

	if resolved == 0 {
		exitWithError("Nė vienas dalyvis nerastas. Patikrinkite vardus ir pavardes.", *noPause)
	}

	// Step 7: Collect results
	fmt.Println("Renkami rezultatai...")
	results, warnings := coll.Collect(participants)
	fmt.Printf("\nSurinkta: %d rezultatų\n", len(results))

	if len(warnings) > 0 {
		fmt.Printf("Įspėjimai: %d\n", len(warnings))
		for _, w := range warnings {
			fmt.Printf("  ! %s\n", w)
		}
	}

	// Step 8: Upsert events and days
	existingEvents, _ := wb.ReadEventIDs()
	seenEvents := make(map[string]bool)
	for _, r := range results {
		if seenEvents[r.Event.ID] {
			continue
		}
		seenEvents[r.Event.ID] = true

		if !existingEvents[r.Event.ID] {
			ev := r.Event
			if ev.Coefficient == 0 {
				ev.Coefficient = 1.00
			}
			wb.AppendEvent(ev)

			// Append days
			for _, d := range r.Days {
				wb.AppendDay(domain_Day(r.Event.ID, d, r.Event), r.Event.Name)
			}
		}
	}

	// Step 9: Read coefficients and day date overrides
	coefficients, _ := wb.ReadCoefficients()
	dayDates, _ := wb.ReadDayDates()

	// Step 10: Write report sheet
	sheetName, err := report.WriteReport(wb.ExcelFile(), results, coefficients, dayDates)
	if err != nil {
		exitWithError(fmt.Sprintf("Ataskaita: %v", err), *noPause)
	}

	// Step 11: Backup and save
	if err := workbook.Backup(wbPath); err != nil {
		fmt.Printf("Įspėjimas: atsarginė kopija nepavyko: %v\n", err)
	}

	if err := wb.Save(); err != nil {
		exitWithError(fmt.Sprintf("Išsaugojimas: %v", err), *noPause)
	}

	fmt.Printf("\nAtaskaita parašyta į lapą: %s\n", sheetName)
	fmt.Printf("Knyga: %s\n", wbPath)
	waitForEnter(*noPause)
}

// execDir returns the directory where the executable lives.
func execDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return "", err
	}
	return filepath.Dir(exe), nil
}

func exitWithError(msg string, noPause bool) {
	fmt.Fprintf(os.Stderr, "Klaida: %s\n", msg)
	waitForEnter(noPause)
	os.Exit(1)
}

func waitForEnter(noPause bool) {
	if noPause {
		return
	}
	fmt.Println("Spauskite Enter, kad uždarytumėte.")
	buf := make([]byte, 1)
	os.Stdin.Read(buf) //nolint:errcheck
}

// domain_Day converts an EventResult DayResult into a domain.Day for storage.
func domain_Day(eventID string, dr domain.DayResult, ev domain.Event) domain.Day {
	return domain.Day{
		EventID: eventID,
		Number:  dr.DayNumber,
		Date:    domain.EstimateDayDate(ev.Date, dr.DayNumber),
	}
}
