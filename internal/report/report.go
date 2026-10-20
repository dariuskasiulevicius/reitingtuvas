// Package report transforms domain results into a report sheet.
package report

import (
	"fmt"
	"sort"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/dariuskasiulevicius/reitingtuvas/internal/domain"
	"github.com/dariuskasiulevicius/reitingtuvas/internal/texts"
)

const durationFmt = "[h]:mm:ss"

// WriteReport creates a new report sheet in f and returns its name.
func WriteReport(
	f *excelize.File,
	results []domain.EventResult,
	coefficients map[string]float64,
	dayDates map[string]time.Time,
) (string, error) {
	sheetName := fmt.Sprintf("%s %s",
		texts.SheetReportPrefix,
		time.Now().Format("2006-01-02 15.04"),
	)
	// Truncate to Excel's 31-char limit
	if len(sheetName) > 31 {
		sheetName = sheetName[:31]
	}

	idx, err := f.NewSheet(sheetName)
	if err != nil {
		return "", fmt.Errorf("create sheet: %w", err)
	}
	_ = idx

	// Sort: by participant surname+name, then event date
	sort.Slice(results, func(i, j int) bool {
		pi := results[i].Participant.Surname + " " + results[i].Participant.Name
		pj := results[j].Participant.Surname + " " + results[j].Participant.Name
		if pi != pj {
			return pi < pj
		}
		return results[i].Event.Date.Before(results[j].Event.Date)
	})

	// Find max day count
	maxDays := 0
	for _, r := range results {
		if len(r.Days) > maxDays {
			maxDays = len(r.Days)
		}
	}

	// Duration style
	durStyleID, err := f.NewStyle(&excelize.Style{
		CustomNumFmt: strPtr(durationFmt),
	})
	if err != nil {
		durStyleID = 0
	}

	// Write header
	fixedHeaders := []string{
		"Dalyvis", "ID", "Varžybų data", "Varžybos", "Varžybų ID",
		"Sportas", "Klasė", "Estafetės etapas", "Koef.",
		"Dalyvio rezultatas", "Vieta", "Lyderio rezultatas", "Pastabos",
	}
	col := 1
	for _, h := range fixedHeaders {
		f.SetCellValue(sheetName, cell(col, 1), h)
		col++
	}
	dayBlockStart := col
	for d := 1; d <= maxDays; d++ {
		prefix := fmt.Sprintf("D%d ", d)
		for _, h := range []string{"data", "būsena", "laikas", "vieta", "lyderio laikas"} {
			f.SetCellValue(sheetName, cell(col, 1), prefix+h)
			col++
		}
	}

	// Write rows
	for i, r := range results {
		row := i + 2

		// Apply coefficient from workbook (may differ from default)
		coef := r.Event.Coefficient
		if c, ok := coefficients[r.Event.ID]; ok {
			coef = c
		}

		displayName := r.Participant.Surname + " " + r.Participant.Name
		sport := translateSport(r.Event.Sport)
		legStr := ""
		if r.RelayLeg > 0 {
			legStr = fmt.Sprintf("%d", r.RelayLeg)
		}

		col = 1
		f.SetCellValue(sheetName, cell(col, row), displayName); col++
		f.SetCellValue(sheetName, cell(col, row), r.Participant.SID); col++
		f.SetCellValue(sheetName, cell(col, row), r.Event.Date.Format("2006-01-02")); col++
		f.SetCellValue(sheetName, cell(col, row), r.Event.Name); col++
		f.SetCellValue(sheetName, cell(col, row), r.Event.ID); col++
		f.SetCellValue(sheetName, cell(col, row), sport); col++
		f.SetCellValue(sheetName, cell(col, row), r.Class); col++
		f.SetCellValue(sheetName, cell(col, row), legStr); col++
		f.SetCellValue(sheetName, cell(col, row), coef); col++

		// Participant result as duration
		setDuration(f, sheetName, cell(col, row), r.Result, durStyleID); col++

		// Place
		if r.Place > 0 {
			f.SetCellValue(sheetName, cell(col, row), r.Place)
		}
		col++

		// Leader result as duration
		setDuration(f, sheetName, cell(col, row), r.LeaderResult, durStyleID); col++

		// Notes
		f.SetCellValue(sheetName, cell(col, row), r.Notes); col++

		// Day blocks
		for d := 0; d < maxDays; d++ {
			if d < len(r.Days) {
				day := r.Days[d]

				// Day date: check overrides, then estimate
				dayDateKey := fmt.Sprintf("%s:%d", r.Event.ID, day.DayNumber)
				dayDate := domain.EstimateDayDate(r.Event.Date, day.DayNumber)
				if override, ok := dayDates[dayDateKey]; ok {
					dayDate = override
				}
				f.SetCellValue(sheetName, cell(col, row), dayDate.Format("2006-01-02")); col++

				// Status (translated)
				f.SetCellValue(sheetName, cell(col, row), translateStatus(day.Status)); col++

				// Time
				setDuration(f, sheetName, cell(col, row), day.Time, durStyleID); col++

				// Place
				if day.Place > 0 {
					f.SetCellValue(sheetName, cell(col, row), day.Place)
				}
				col++

				// Leader time
				setDuration(f, sheetName, cell(col, row), day.LeaderTime, durStyleID); col++
			} else {
				col += 5 // skip empty day block
			}
		}
	}

	// Freeze header row
	f.SetPanes(sheetName, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	})

	// Autofilter
	lastCol := dayBlockStart - 1 + maxDays*5
	if lastCol < len(fixedHeaders) {
		lastCol = len(fixedHeaders)
	}
	endCell := cell(lastCol, 1)
	f.AutoFilter(sheetName, "A1:"+endCell, nil)

	return sheetName, nil
}

func setDuration(f *excelize.File, sheet, cellRef string, d time.Duration, styleID int) {
	if d > 0 {
		// Excel stores durations as fraction of a day
		dayFraction := d.Seconds() / 86400.0
		f.SetCellValue(sheet, cellRef, dayFraction)
		f.SetCellStyle(sheet, cellRef, cellRef, styleID)
	}
}

func translateStatus(status string) string {
	if t, ok := texts.DayStatusDisplay[status]; ok {
		return t
	}
	return status
}

func translateSport(sport string) string {
	// Strip "sport " prefix from alt text (e.g. "sport ob" → "ob")
	if len(sport) > 6 && sport[:6] == "sport " {
		return sport[6:]
	}
	return sport
}

func cell(col, row int) string {
	name, _ := excelize.CoordinatesToCellName(col, row)
	return name
}

func strPtr(s string) *string { return &s }
