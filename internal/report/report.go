package report

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/dariuskasiulevicius/reitingtuvas/internal/domain"
	"github.com/dariuskasiulevicius/reitingtuvas/internal/texts"
)

// EventColumn represents one column in the report (one event-day).
type EventColumn struct {
	EventID   string
	EventName string
	Date      time.Time
	DayNumber int
	Key       string // "eventID:dayNumber"
}

// WriteReport creates the rating table sheet. Returns its name.
// If overwriteSheet is non-empty, that sheet is replaced.
func WriteReport(
	f *excelize.File,
	rating []domain.RatingEntry,
	columns []EventColumn,
	coefficients map[string]float64,
	openClassCoef float64,
	topN int,
	periodYear int,
	overwriteSheet string,
) (string, error) {
	sheetName := overwriteSheet
	if sheetName == "" {
		sheetName = fmt.Sprintf("%s %s",
			texts.SheetReportPrefix,
			time.Now().Format("2006-01-02 15.04"),
		)
		if len(sheetName) > 31 {
			sheetName = sheetName[:31]
		}
	}

	if overwriteSheet != "" {
		f.DeleteSheet(sheetName)
	}
	if _, err := f.NewSheet(sheetName); err != nil {
		return "", fmt.Errorf("create sheet: %w", err)
	}

	// Styles
	scoreStyle, _ := f.NewStyle(&excelize.Style{CustomNumFmt: strPtr("0.00")})
	boldScoreStyle, _ := f.NewStyle(&excelize.Style{
		CustomNumFmt: strPtr("0.00"),
		Font:         &excelize.Font{Bold: true},
	})
	coefStyle, _ := f.NewStyle(&excelize.Style{CustomNumFmt: strPtr("0.00")})
	boldStyle, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})

	// Column layout:
	// A=Place, B=Name, C..=event days, then StartCount, SeasonScore, Name, Place
	eventColStart := 3 // col C (1-based)
	eventColEnd := eventColStart + len(columns) - 1
	colStarts := eventColEnd + 1
	colSeason := colStarts + 1
	colNameR := colSeason + 1
	colPlaceR := colNameR + 1

	seasonHeader := fmt.Sprintf(texts.ColReportSeasonFmt, topN)

	// --- Row 1: Title ---
	titleCell := cell(1, 1)
	f.SetCellValue(sheetName, titleCell, fmt.Sprintf(texts.ReportTitleFmt, periodYear))
	f.SetCellStyle(sheetName, titleCell, titleCell, boldStyle)

	// --- Row 3: Event names ---
	f.SetCellValue(sheetName, cell(1, 3), texts.ColReportPlace)
	f.SetCellValue(sheetName, cell(2, 3), texts.ColReportEventName)
	for i, col := range columns {
		f.SetCellValue(sheetName, cell(eventColStart+i, 3), col.EventName)
	}
	f.SetCellValue(sheetName, cell(colStarts, 3), texts.ColReportStarts)
	f.SetCellValue(sheetName, cell(colSeason, 3), seasonHeader)
	f.SetCellValue(sheetName, cell(colNameR, 3), texts.ColReportName)
	f.SetCellValue(sheetName, cell(colPlaceR, 3), texts.ColReportPlace)

	// --- Row 4: Dates ---
	f.SetCellValue(sheetName, cell(2, 4), texts.ColReportDate)
	for i, col := range columns {
		f.SetCellValue(sheetName, cell(eventColStart+i, 4), col.Date.Format("01 02"))
	}

	// --- Row 5: Coefficients ---
	f.SetCellValue(sheetName, cell(2, 5), texts.ColReportCoef)
	for i, col := range columns {
		coef := 1.0
		if c, ok := coefficients[col.EventID]; ok {
			coef = c
		}
		c := cell(eventColStart+i, 5)
		f.SetCellValue(sheetName, c, coef)
		f.SetCellStyle(sheetName, c, c, coefStyle)
	}
	f.SetCellValue(sheetName, cell(colStarts, 5), len(columns))
	maxScore := 0.0
	for _, col := range columns {
		coef := 1.0
		if c, ok := coefficients[col.EventID]; ok {
			coef = c
		}
		maxScore += coef
	}
	topMaxScore := 0.0
	if len(columns) > 0 {
		// Sum of top N coefficients
		var coefs []float64
		for _, col := range columns {
			coef := 1.0
			if c, ok := coefficients[col.EventID]; ok {
				coef = c
			}
			coefs = append(coefs, coef)
		}
		sort.Float64s(coefs)
		count := 0
		for i := len(coefs) - 1; i >= 0 && count < topN; i-- {
			topMaxScore += coefs[i]
			count++
		}
	}
	sc := cell(colSeason, 5)
	f.SetCellValue(sheetName, sc, topMaxScore)
	f.SetCellStyle(sheetName, sc, sc, coefStyle)
	f.SetCellValue(sheetName, cell(colNameR, 5), "\"koeficientas\"")

	// --- Row 6: Sequential numbers ---
	f.SetCellValue(sheetName, cell(2, 6), texts.ColReportSeqNum)
	for i := range columns {
		f.SetCellValue(sheetName, cell(eventColStart+i, 6), i+1)
	}

	// --- Data rows (from row 7) ---
	dataRowStart := 7
	for i, entry := range rating {
		row := dataRowStart + i
		displayName := entry.Surname + " " + entry.Name

		f.SetCellValue(sheetName, cell(1, row), entry.Place)
		f.SetCellValue(sheetName, cell(2, row), displayName)

		topKeys := domain.TopNDayKeys(entry.DayScores, topN)
		for j, col := range columns {
			if score, ok := entry.DayScores[col.Key]; ok {
				c := cell(eventColStart+j, row)
				f.SetCellValue(sheetName, c, roundTo2(score))
				if topKeys[col.Key] {
					f.SetCellStyle(sheetName, c, c, boldScoreStyle)
				} else {
					f.SetCellStyle(sheetName, c, c, scoreStyle)
				}
			}
		}

		f.SetCellValue(sheetName, cell(colStarts, row), entry.StartCount)

		sc := cell(colSeason, row)
		f.SetCellValue(sheetName, sc, roundTo2(entry.SeasonScore))
		f.SetCellStyle(sheetName, sc, sc, scoreStyle)

		f.SetCellValue(sheetName, cell(colNameR, row), displayName)
		f.SetCellValue(sheetName, cell(colPlaceR, row), entry.Place)
	}

	// Freeze pane at C1 (name col visible)
	f.SetPanes(sheetName, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      2,
		YSplit:      6,
		TopLeftCell: "C7",
		ActivePane:  "bottomRight",
	})

	return sheetName, nil
}

// BuildEventColumns creates the sorted column list from stored results + events.
func BuildEventColumns(
	results []domain.StoredResult,
	events map[string]domain.Event,
) []EventColumn {
	seen := make(map[string]bool)
	var cols []EventColumn

	for _, r := range results {
		key := r.EventID + ":" + strconv.Itoa(r.DayNumber)
		if seen[key] {
			continue
		}
		seen[key] = true

		ev := events[r.EventID]
		cols = append(cols, EventColumn{
			EventID:   r.EventID,
			EventName: ev.Name,
			Date:      domain.EstimateDayDate(ev.Date, r.DayNumber),
			DayNumber: r.DayNumber,
			Key:       key,
		})
	}

	sort.Slice(cols, func(i, j int) bool {
		if !cols[i].Date.Equal(cols[j].Date) {
			return cols[i].Date.Before(cols[j].Date)
		}
		if cols[i].EventID != cols[j].EventID {
			return cols[i].EventID < cols[j].EventID
		}
		return cols[i].DayNumber < cols[j].DayNumber
	})

	return cols
}

func cell(col, row int) string {
	name, _ := excelize.CoordinatesToCellName(col, row)
	return name
}

func strPtr(s string) *string { return &s }

func roundTo2(f float64) float64 {
	return math.Round(f*100) / 100
}
