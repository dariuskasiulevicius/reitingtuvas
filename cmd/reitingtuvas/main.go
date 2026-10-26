package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dariuskasiulevicius/reitingtuvas/internal/collect"
	"github.com/dariuskasiulevicius/reitingtuvas/internal/domain"
	"github.com/dariuskasiulevicius/reitingtuvas/internal/loflv"
	"github.com/dariuskasiulevicius/reitingtuvas/internal/report"
	"github.com/dariuskasiulevicius/reitingtuvas/internal/source"
	"github.com/dariuskasiulevicius/reitingtuvas/internal/texts"
	"github.com/dariuskasiulevicius/reitingtuvas/internal/workbook"
)

var version = "dev"

const workbookName = "reitingtuvas.xlsx"

func main() {
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

	// Lock check
	if _, err := os.Stat(wbPath); err == nil {
		if err := workbook.CheckLock(wbPath); err != nil {
			exitWithError(err.Error(), *noPause)
		}
	}

	// Open workbook (or create template)
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

	// Read settings
	rawSettings, err := wb.ReadSettings()
	if err != nil {
		exitWithError(fmt.Sprintf("Nustatymai: %v", err), *noPause)
	}

	settings, err := workbook.ParseSettings(rawSettings)
	if err != nil {
		exitWithError(fmt.Sprintf("Nustatymai: %v", err), *noPause)
	}

	lvSettings, err := loflv.ParseLVSettings(rawSettings)
	if err != nil {
		exitWithError(fmt.Sprintf("LV nustatymai: %v", err), *noPause)
	}

	fmt.Printf("Laikotarpis: %s — %s\n",
		settings.PeriodFrom.Format("2006-01-02"),
		settings.PeriodTo.Format("2006-01-02"),
	)

	// Read participants
	participants, err := wb.ReadParticipants()
	if err != nil {
		exitWithError(fmt.Sprintf("Dalyviai: %v", err), *noPause)
	}
	if len(participants) == 0 {
		exitWithError("Dalyvių sąrašas tuščias.", *noPause)
	}
	fmt.Printf("Dalyviai: %d\n", len(participants))

	// Prompt: download?
	fmt.Print(texts.MsgPromptDownload)
	download := askYN()

	if download {
		cacheDir := filepath.Join(dir, "cache")
		cache, err := source.NewCache(cacheDir)
		if err != nil {
			exitWithError(fmt.Sprintf("Talpykla: %v", err), *noPause)
		}

		client := source.NewClient(settings.BaseURL, settings.Delay, cache, settings.CacheFinalDays, version)

		progress := func(current, total int, desc string) {
			fmt.Printf("\r[%d/%d] %s          ", current, total, desc)
		}

		coll := collect.NewCollector(client, settings, progress)

		// Resolve participants
		fmt.Println("Ieškoma dalyvių...")
		participants = coll.Resolve(participants)

		resolved := 0
		for _, p := range participants {
			if p.SID != "" {
				resolved++
			}
		}
		fmt.Printf("\nRasta: %d/%d\n", resolved, len(participants))

		if err := wb.WriteParticipants(participants); err != nil {
			fmt.Printf("Įspėjimas: nepavyko atnaujinti dalyvių: %v\n", err)
		}

		if resolved == 0 {
			exitWithError("Nė vienas dalyvis nerastas. Patikrinkite vardus ir pavardes.", *noPause)
		}

		// Collect results
		fmt.Println("Renkami rezultatai...")
		crawled, warnings := coll.Collect(participants)
		stats := client.Stats
		fmt.Printf("\nSurinkta: %d rezultatų\n", len(crawled))
		fmt.Printf("Užklausos: %d, iš talpyklos: %d, klaidos: %d\n",
			stats.Requests, stats.CacheHits, stats.Errors)

		if len(warnings) > 0 {
			fmt.Printf("Įspėjimai: %d\n", len(warnings))
			for _, w := range warnings {
				fmt.Printf("  ! %s\n", w)
			}
		}

		// Upsert events + days
		existingEvents, _ := wb.ReadEventIDs()
		seenEvents := make(map[string]bool)
		for _, r := range crawled {
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

				for _, d := range r.Days {
					wb.AppendDay(domainDay(r.Event.ID, d, r.Event), r.Event.Name)
				}
			}
		}

		// Clear + write Rezultatai
		stored := workbook.EventResultsToStored(crawled, settings.BaseURL, settings.Templates.Class)
		if err := wb.ClearAndWriteResults(stored); err != nil {
			exitWithError(fmt.Sprintf("Rezultatai: %v", err), *noPause)
		}
		fmt.Printf("Rezultatai įrašyti: %d eilučių\n", len(stored))

		// LV collection
		if lvSettings.Enabled() {
			fmt.Println("\nRenkami LV rezultatai...")
			lvClient := source.NewClient(lvSettings.BaseURL, settings.Delay, cache, settings.CacheFinalDays, version)

			lvProgress := func(current, total int, desc string) {
				fmt.Printf("\r[%d/%d] LV: %s          ", current, total, desc)
			}

			lvColl := loflv.NewCollector(lvClient, lvSettings, settings, lvProgress)
			lvResults, lvWarnings := lvColl.Collect(participants)
			lvStats := lvClient.Stats
			fmt.Printf("\nLV surinkta: %d rezultatų\n", len(lvResults))
			fmt.Printf("LV užklausos: %d, iš talpyklos: %d, klaidos: %d\n",
				lvStats.Requests, lvStats.CacheHits, lvStats.Errors)

			if len(lvWarnings) > 0 {
				fmt.Printf("LV įspėjimai: %d\n", len(lvWarnings))
				for _, w := range lvWarnings {
					fmt.Printf("  ! %s\n", w)
				}
			}

			// Upsert LV events + days
			for _, r := range lvResults {
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

					for _, d := range r.Days {
						wb.AppendDay(domainDay(r.Event.ID, d, r.Event), r.Event.Name)
					}
				}
			}

			// Write LV results to separate sheet
			lvStored := workbook.EventResultsToStored(lvResults, "", nil)
			if err := wb.ClearAndWriteResultsLV(lvStored); err != nil {
				exitWithError(fmt.Sprintf("Rezultatai-LV: %v", err), *noPause)
			}
			fmt.Printf("LV rezultatai įrašyti: %d eilučių\n", len(lvStored))
		}
	}

	// Read all results (Rezultatai + Rezultatai-LV + Rankinis)
	rezultatai, _ := wb.ReadResults()
	rezultataiLV, _ := wb.ReadResultsLV()
	rankinis, manualEvents, manualDays, manualWarnings := wb.ReadManualEntries(participants)

	if len(manualWarnings) > 0 {
		fmt.Printf("Rankinio įvedimo įspėjimai: %d\n", len(manualWarnings))
		for _, w := range manualWarnings {
			fmt.Printf("  ! %s\n", w)
		}
	}

	// Auto-register manual events/days
	if len(manualEvents) > 0 {
		existingEvt, _ := wb.ReadEventIDs()
		manualEventNames := make(map[string]string)
		for _, ev := range manualEvents {
			manualEventNames[ev.ID] = ev.Name
			if !existingEvt[ev.ID] {
				wb.AppendEvent(ev)
			}
		}
		for _, d := range manualDays {
			wb.AppendDay(d, manualEventNames[d.EventID])
		}
	}

	allResults := append(rezultatai, rezultataiLV...)
	allResults = append(allResults, rankinis...)

	if len(allResults) == 0 {
		exitWithError(texts.MsgNoResults, *noPause)
	}

	fmt.Printf("Rezultatai ataskaitai: %d (LT: %d, LV: %d, rankiniai: %d)\n",
		len(allResults), len(rezultatai), len(rezultataiLV), len(rankinis))

	// Read coefficients + events
	coefficients, _ := wb.ReadCoefficients()
	events, _ := wb.ReadEvents()

	// Compute rating
	rating := domain.ComputeRating(allResults, participants, coefficients, settings.OpenClassCoef, settings.TopN)
	columns := report.BuildEventColumns(allResults, events)

	periodYear := settings.PeriodFrom.Year()

	// Prompt: overwrite or new?
	var overwriteSheet string
	if last := wb.LastReportSheet(); last != "" {
		fmt.Print(texts.MsgPromptOverwrite)
		if askOverwrite() {
			overwriteSheet = last
		}
	}

	sheetName, err := report.WriteReport(
		wb.ExcelFile(), rating, columns, coefficients,
		settings.OpenClassCoef, settings.TopN, periodYear, overwriteSheet,
	)
	if err != nil {
		exitWithError(fmt.Sprintf("Ataskaita: %v", err), *noPause)
	}

	// Backup + save
	if err := workbook.Backup(wbPath); err != nil {
		fmt.Printf("Įspėjimas: atsarginė kopija nepavyko: %v\n", err)
	}

	if err := wb.Save(); err != nil {
		exitWithError(fmt.Sprintf("Išsaugojimas: %v", err), *noPause)
	}

	fmt.Printf("\nAtaskaita parašyta į lapą: %s\n", sheetName)
	fmt.Printf("Dalyvių: %d, Varžybų dienų: %d\n", len(rating), len(columns))
	fmt.Printf("Knyga: %s\n", wbPath)
	waitForEnter(*noPause)
}

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
	fmt.Println(texts.MsgPressEnter)
	buf := make([]byte, 1)
	os.Stdin.Read(buf) //nolint:errcheck
}

func askYN() bool {
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return strings.TrimSpace(strings.ToLower(scanner.Text())) == "t"
	}
	return false
}

func askOverwrite() bool {
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return strings.TrimSpace(strings.ToLower(scanner.Text())) == "p"
	}
	return false
}

func domainDay(eventID string, dr domain.DayResult, ev domain.Event) domain.Day {
	return domain.Day{
		EventID: eventID,
		Number:  dr.DayNumber,
		Date:    domain.EstimateDayDate(ev.Date, dr.DayNumber),
	}
}
