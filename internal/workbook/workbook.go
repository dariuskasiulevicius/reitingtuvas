// Package workbook handles reading and writing the xlsx workbook.
package workbook

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/dariuskasiulevicius/reitingtuvas/internal/domain"
	"github.com/dariuskasiulevicius/reitingtuvas/internal/source"
	"github.com/dariuskasiulevicius/reitingtuvas/internal/texts"
)

// Workbook wraps the xlsx file with domain-aware read/write operations.
type Workbook struct {
	path string
	f    *excelize.File
}

// Open opens an existing workbook or creates a new template if missing.
// Returns (wb, created, error). If created=true, the caller should inform
// the user to fill in settings and stop.
func Open(path string) (*Workbook, bool, error) {
	wb := &Workbook{path: path}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := wb.createTemplate(); err != nil {
			return nil, false, fmt.Errorf("create template: %w", err)
		}
		return wb, true, nil
	}

	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, false, fmt.Errorf("open workbook: %w", err)
	}
	wb.f = f
	return wb, false, nil
}

// CheckLock tries to open the file for writing. Returns error if locked.
func CheckLock(path string) error {
	f, err := os.OpenFile(path, os.O_RDWR, 0o644)
	if err != nil {
		return fmt.Errorf("%s: %w", texts.MsgCloseWorkbook, err)
	}
	f.Close()
	return nil
}

// Backup copies the workbook to path.backup.xlsx.
func Backup(path string) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	dst := base + ".backup" + ext
	return os.WriteFile(dst, src, 0o644)
}

// ExcelFile returns the underlying excelize file for report writing.
func (wb *Workbook) ExcelFile() *excelize.File {
	return wb.f
}

// Close closes the underlying excelize file.
func (wb *Workbook) Close() error {
	if wb.f != nil {
		return wb.f.Close()
	}
	return nil
}

// Save writes the workbook to disk.
func (wb *Workbook) Save() error {
	return wb.f.SaveAs(wb.path)
}

// --- Settings ---

// ReadSettings reads key-value pairs from the Nustatymai sheet.
func (wb *Workbook) ReadSettings() (map[string]string, error) {
	rows, err := wb.f.GetRows(texts.SheetSettings)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", texts.SheetSettings, err)
	}
	settings := make(map[string]string)
	for _, row := range rows {
		if len(row) >= 2 {
			key := strings.TrimSpace(row[0])
			val := strings.TrimSpace(row[1])
			if key != "" {
				settings[key] = val
			}
		}
	}
	return settings, nil
}

// ParseSettings converts raw settings map into a source.Settings.
func ParseSettings(raw map[string]string) (*source.Settings, error) {
	s := &source.Settings{}
	s.BaseURL = raw[texts.SettingBaseURL]

	tmpls, err := source.NewTemplates(
		raw[texts.SettingSearchPath],
		raw[texts.SettingPersonPath],
		raw[texts.SettingClassPath],
		raw[texts.SettingEventPath],
		raw[texts.SettingProfilePath],
	)
	if err != nil {
		return nil, fmt.Errorf("URL šablonai: %w", err)
	}
	s.Templates = tmpls

	if v := raw[texts.SettingPeriodFrom]; v != "" {
		t, err := parseFlexDate(v)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", texts.SettingPeriodFrom, err)
		}
		s.PeriodFrom = t
	}
	if v := raw[texts.SettingPeriodTo]; v != "" {
		t, err := parseFlexDate(v)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", texts.SettingPeriodTo, err)
		}
		s.PeriodTo = t
	}

	delayMs := 200
	if v := raw[texts.SettingDelay]; v != "" {
		delayMs, _ = strconv.Atoi(v)
	}
	s.Delay = time.Duration(delayMs) * time.Millisecond

	s.CacheFinalDays = 14
	if v := raw[texts.SettingCacheFinalDays]; v != "" {
		s.CacheFinalDays, _ = strconv.Atoi(v)
	}

	s.OpenClassCoef = 0.9
	if v := raw[texts.SettingOpenClassCoef]; v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			s.OpenClassCoef = f
		}
	}

	s.TopN = 8
	if v := raw[texts.SettingTopN]; v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			s.TopN = n
		}
	}

	return s, s.Validate()
}

// --- Participants ---

// ReadParticipants reads the Dalyviai sheet.
func (wb *Workbook) ReadParticipants() ([]domain.Participant, error) {
	rows, err := wb.f.GetRows(texts.SheetParticipants)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", texts.SheetParticipants, err)
	}

	headers := findHeaders(rows)
	colSurname := headers[texts.ColSurname]
	colName := headers[texts.ColName]
	colSID := headers[texts.ColSID]

	var participants []domain.Participant
	for i, row := range rows {
		if i == 0 {
			continue // header
		}
		p := domain.Participant{
			Surname: cellAt(row, colSurname),
			Name:    cellAt(row, colName),
			SID:     cellAt(row, colSID),
		}
		if p.Surname == "" && p.Name == "" {
			continue
		}
		participants = append(participants, p)
	}
	return participants, nil
}

// WriteParticipants updates the SID and Status columns in the Dalyviai sheet.
func (wb *Workbook) WriteParticipants(participants []domain.Participant) error {
	sheet := texts.SheetParticipants
	rows, err := wb.f.GetRows(sheet)
	if err != nil {
		return err
	}
	headers := findHeaders(rows)
	colSID := headers[texts.ColSID]
	colStatus := headers[texts.ColStatus]

	for i, p := range participants {
		row := i + 2 // 1-based, skip header
		if colSID >= 0 {
			cell, _ := excelize.CoordinatesToCellName(colSID+1, row)
			wb.f.SetCellValue(sheet, cell, p.SID)
		}
		if colStatus >= 0 {
			cell, _ := excelize.CoordinatesToCellName(colStatus+1, row)
			wb.f.SetCellValue(sheet, cell, p.Status)
		}
	}
	return nil
}

// --- Events ---

// ReadEvents reads event IDs already in the Varžybos sheet.
func (wb *Workbook) ReadEventIDs() (map[string]bool, error) {
	rows, err := wb.f.GetRows(texts.SheetEvents)
	if err != nil {
		return nil, err
	}
	ids := make(map[string]bool)
	for i, row := range rows {
		if i == 0 {
			continue
		}
		if len(row) > 0 && row[0] != "" {
			ids[row[0]] = true
		}
	}
	return ids, nil
}

// AppendEvent adds a new event row if its ID is not already present.
func (wb *Workbook) AppendEvent(ev domain.Event) error {
	sheet := texts.SheetEvents
	rows, err := wb.f.GetRows(sheet)
	if err != nil {
		return err
	}
	nextRow := len(rows) + 1

	setCell := func(col int, val interface{}) {
		cell, _ := excelize.CoordinatesToCellName(col, nextRow)
		wb.f.SetCellValue(sheet, cell, val)
	}

	relay := ""
	if ev.IsRelay {
		relay = "taip"
	}

	setCell(1, ev.ID)
	setCell(2, ev.Name)
	setCell(3, ev.Date.Format("2006-01-02"))
	setCell(4, ev.Sport)
	setCell(5, relay)
	setCell(6, ev.DayCount)
	setCell(7, ev.Coefficient)
	return nil
}

// ReadEvents reads all events from the Varžybos sheet.
func (wb *Workbook) ReadEvents() (map[string]domain.Event, error) {
	rows, err := wb.f.GetRows(texts.SheetEvents)
	if err != nil {
		return nil, err
	}
	events := make(map[string]domain.Event)
	for i, row := range rows {
		if i == 0 {
			continue
		}
		if len(row) < 3 || strings.TrimSpace(row[0]) == "" {
			continue
		}
		ev := domain.Event{ID: strings.TrimSpace(row[0])}
		if len(row) > 1 {
			ev.Name = strings.TrimSpace(row[1])
		}
		if len(row) > 2 {
			ev.Date, _ = parseFlexDate(row[2])
		}
		if len(row) > 3 {
			ev.Sport = strings.TrimSpace(row[3])
		}
		if len(row) > 4 {
			ev.IsRelay = strings.TrimSpace(row[4]) == "taip"
		}
		if len(row) > 5 {
			ev.DayCount, _ = strconv.Atoi(strings.TrimSpace(row[5]))
		}
		if len(row) > 6 {
			c, err := strconv.ParseFloat(strings.TrimSpace(row[6]), 64)
			if err == nil {
				ev.Coefficient = c
			}
		}
		events[ev.ID] = ev
	}
	return events, nil
}

// ReadCoefficients reads event ID → coefficient from the Varžybos sheet.
func (wb *Workbook) ReadCoefficients() (map[string]float64, error) {
	rows, err := wb.f.GetRows(texts.SheetEvents)
	if err != nil {
		return nil, err
	}
	coefs := make(map[string]float64)
	for i, row := range rows {
		if i == 0 {
			continue
		}
		if len(row) < 7 {
			continue
		}
		id := strings.TrimSpace(row[0])
		coefStr := strings.TrimSpace(row[6])
		if id != "" && coefStr != "" {
			coef, err := strconv.ParseFloat(coefStr, 64)
			if err == nil {
				coefs[id] = coef
			}
		}
	}
	return coefs, nil
}

// --- Days ---

// AppendDay adds a new day row to the Dienos sheet.
func (wb *Workbook) AppendDay(d domain.Day, eventName string) error {
	sheet := texts.SheetDays
	rows, err := wb.f.GetRows(sheet)
	if err != nil {
		return err
	}
	nextRow := len(rows) + 1

	setCell := func(col int, val interface{}) {
		cell, _ := excelize.CoordinatesToCellName(col, nextRow)
		wb.f.SetCellValue(sheet, cell, val)
	}

	setCell(1, d.EventID)
	setCell(2, eventName)
	setCell(3, d.Number)
	setCell(4, d.Name)
	setCell(5, d.Date.Format("2006-01-02"))
	return nil
}

// ReadDayDates reads event-day → date overrides from the Dienos sheet.
func (wb *Workbook) ReadDayDates() (map[string]time.Time, error) {
	rows, err := wb.f.GetRows(texts.SheetDays)
	if err != nil {
		return nil, err
	}
	dates := make(map[string]time.Time)
	for i, row := range rows {
		if i == 0 {
			continue
		}
		if len(row) < 5 {
			continue
		}
		eventID := strings.TrimSpace(row[0])
		dayNumStr := strings.TrimSpace(row[2])
		dateStr := strings.TrimSpace(row[4])
		if eventID != "" && dateStr != "" {
			key := eventID + ":" + dayNumStr
			if t, err := time.Parse("2006-01-02", dateStr); err == nil {
				dates[key] = t
			}
		}
	}
	return dates, nil
}

// --- Results (Rezultatai) ---

// ReadStoredResults reads rows from a results-format sheet.
func (wb *Workbook) readResultSheet(sheet string) ([]domain.StoredResult, error) {
	rows, err := wb.f.GetRows(sheet)
	if err != nil {
		return nil, err
	}
	headers := findHeaders(rows)

	colSID := headers[texts.ColResultSID]
	colEventID := headers[texts.ColResultEventID]
	colDay := headers[texts.ColResultDay]
	colClass := headers[texts.ColResultClass]
	colTime := headers[texts.ColResultTime]
	colLeader := headers[texts.ColResultLeaderTime]
	colStatus := headers[texts.ColResultStatus]
	colPlace := headers[texts.ColResultPlace]

	var results []domain.StoredResult
	for i, row := range rows {
		if i == 0 {
			continue
		}
		sid := cellAt(row, colSID)
		eventID := cellAt(row, colEventID)
		if sid == "" || eventID == "" {
			continue
		}

		r := domain.StoredResult{
			SID:     sid,
			EventID: eventID,
			Class:   cellAt(row, colClass),
			Status:  cellAt(row, colStatus),
		}
		if v := cellAt(row, colDay); v != "" {
			r.DayNumber, _ = strconv.Atoi(v)
		}
		if r.DayNumber == 0 {
			r.DayNumber = 1
		}
		if v := cellAt(row, colTime); v != "" {
			r.Time = parseDurationCell(v)
		}
		if v := cellAt(row, colLeader); v != "" {
			r.LeaderTime = parseDurationCell(v)
		}
		if v := cellAt(row, colPlace); v != "" {
			r.Place, _ = strconv.Atoi(v)
		}
		if r.Status == "" && r.Time > 0 {
			r.Status = "ok"
		}
		results = append(results, r)
	}
	return results, nil
}

// ReadResults reads the Rezultatai sheet.
func (wb *Workbook) ReadResults() ([]domain.StoredResult, error) {
	return wb.readResultSheet(texts.SheetResults)
}

// ReadManualEntries reads the Rankinis įvedimas sheet.
func (wb *Workbook) ReadManualEntries() ([]domain.StoredResult, error) {
	return wb.readResultSheet(texts.SheetManualEntry)
}

// ClearAndWriteResults clears the Rezultatai sheet and writes new rows.
func (wb *Workbook) ClearAndWriteResults(results []domain.StoredResult) error {
	sheet := texts.SheetResults

	// Delete and recreate sheet to clear all data
	wb.f.DeleteSheet(sheet)
	wb.f.NewSheet(sheet)
	for i, h := range texts.ResultColumns {
		wb.f.SetCellValue(sheet, cellName(i+1, 1), h)
	}

	for i, r := range results {
		row := i + 2
		wb.f.SetCellValue(sheet, cellName(1, row), r.SID)
		wb.f.SetCellValue(sheet, cellName(2, row), r.EventID)
		wb.f.SetCellValue(sheet, cellName(3, row), r.DayNumber)
		wb.f.SetCellValue(sheet, cellName(4, row), r.Class)
		if r.Time > 0 {
			wb.f.SetCellValue(sheet, cellName(5, row), formatDuration(r.Time))
		}
		if r.LeaderTime > 0 {
			wb.f.SetCellValue(sheet, cellName(6, row), formatDuration(r.LeaderTime))
		}
		wb.f.SetCellValue(sheet, cellName(7, row), r.Status)
		if r.Place > 0 {
			wb.f.SetCellValue(sheet, cellName(8, row), r.Place)
		}
	}
	return nil
}

// HasResults returns true if the Rezultatai sheet has data rows.
func (wb *Workbook) HasResults() bool {
	rows, err := wb.f.GetRows(texts.SheetResults)
	if err != nil {
		return false
	}
	return len(rows) > 1
}

// parseDurationCell parses "h:mm:ss" or Excel day-fraction into time.Duration.
func parseDurationCell(s string) time.Duration {
	// Try h:mm:ss format
	if strings.Contains(s, ":") {
		parts := strings.Split(s, ":")
		switch len(parts) {
		case 3:
			h, _ := strconv.Atoi(parts[0])
			m, _ := strconv.Atoi(parts[1])
			sec, _ := strconv.Atoi(parts[2])
			return time.Duration(h)*time.Hour + time.Duration(m)*time.Minute + time.Duration(sec)*time.Second
		case 2:
			m, _ := strconv.Atoi(parts[0])
			sec, _ := strconv.Atoi(parts[1])
			return time.Duration(m)*time.Minute + time.Duration(sec)*time.Second
		}
	}
	// Try decimal (Excel day fraction)
	if f, err := strconv.ParseFloat(s, 64); err == nil && f > 0 && f < 1 {
		return time.Duration(f * 86400 * float64(time.Second))
	}
	return 0
}

// formatDuration formats a duration as h:mm:ss.
func formatDuration(d time.Duration) string {
	total := int(d.Seconds())
	h := total / 3600
	m := (total % 3600) / 60
	s := total % 60
	return fmt.Sprintf("%d:%02d:%02d", h, m, s)
}

// EventResultsToStored converts crawled EventResults into StoredResult rows.
func EventResultsToStored(results []domain.EventResult) []domain.StoredResult {
	var stored []domain.StoredResult
	for _, er := range results {
		for _, day := range er.Days {
			if day.Status == "not entered" {
				continue
			}
			stored = append(stored, domain.StoredResult{
				SID:        er.Participant.SID,
				EventID:    er.Event.ID,
				DayNumber:  day.DayNumber,
				Class:      day.Class,
				Time:       day.Time,
				LeaderTime: day.LeaderTime,
				Status:     day.Status,
				Place:      day.Place,
			})
		}
	}
	return stored
}

// LastReportSheet returns the name of the most recent report sheet, or "".
func (wb *Workbook) LastReportSheet() string {
	var last string
	for _, name := range wb.f.GetSheetList() {
		if strings.HasPrefix(name, texts.SheetReportPrefix) {
			last = name
		}
	}
	return last
}

// --- Template creation ---

func (wb *Workbook) createTemplate() error {
	f := excelize.NewFile()
	wb.f = f

	// Settings sheet
	f.SetSheetName("Sheet1", texts.SheetSettings)
	settingsKeys := []string{
		texts.SettingBaseURL,
		texts.SettingSearchPath,
		texts.SettingPersonPath,
		texts.SettingClassPath,
		texts.SettingEventPath,
		texts.SettingProfilePath,
		texts.SettingPeriodFrom,
		texts.SettingPeriodTo,
		texts.SettingDelay,
		texts.SettingCacheFinalDays,
		texts.SettingOpenClassCoef,
		texts.SettingTopN,
	}
	defaults := map[string]string{
		texts.SettingDelay:          "200",
		texts.SettingCacheFinalDays: "14",
		texts.SettingOpenClassCoef:  "0.9",
		texts.SettingTopN:           "8",
	}
	for i, key := range settingsKeys {
		row := i + 1
		f.SetCellValue(texts.SheetSettings, cellName(1, row), key)
		if def, ok := defaults[key]; ok {
			f.SetCellValue(texts.SheetSettings, cellName(2, row), def)
		}
	}

	// Participants sheet
	f.NewSheet(texts.SheetParticipants)
	for i, h := range []string{texts.ColSurname, texts.ColName, texts.ColSID, texts.ColStatus} {
		f.SetCellValue(texts.SheetParticipants, cellName(i+1, 1), h)
	}

	// Events sheet
	f.NewSheet(texts.SheetEvents)
	for i, h := range []string{"ID", "Pavadinimas", "Data", "Sportas", "Estafetė", "Dienų sk.", "Koeficientas"} {
		f.SetCellValue(texts.SheetEvents, cellName(i+1, 1), h)
	}

	// Days sheet
	f.NewSheet(texts.SheetDays)
	for i, h := range []string{"Varžybų ID", "Varžybos", "Diena", "Dienos pavadinimas", "Data"} {
		f.SetCellValue(texts.SheetDays, cellName(i+1, 1), h)
	}

	// Results sheet (crawled data)
	f.NewSheet(texts.SheetResults)
	for i, h := range texts.ResultColumns {
		f.SetCellValue(texts.SheetResults, cellName(i+1, 1), h)
	}

	// Manual entry sheet (same columns)
	f.NewSheet(texts.SheetManualEntry)
	for i, h := range texts.ResultColumns {
		f.SetCellValue(texts.SheetManualEntry, cellName(i+1, 1), h)
	}

	return f.SaveAs(wb.path)
}

// --- Helpers ---

// parseFlexDate parses dates in multiple formats that Excel may produce:
// "2006-01-02", "01-02-06", "01/02/2006", "1/2/2006", or an Excel serial number.
func parseFlexDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)

	// Try Excel serial number first (pure digits or float)
	// Excel serial 36526 = 2000-01-01, 73050 ≈ 2099-12-31
	if serial, err := strconv.ParseFloat(s, 64); err == nil && serial >= 36526 && serial <= 73050 {
		// Excel epoch: 1900-01-01 = serial 1 (with the 1900 leap year bug)
		days := int(serial) - 1
		if serial >= 60 {
			days-- // Excel incorrectly treats 1900 as a leap year
		}
		epoch := time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)
		return epoch.AddDate(0, 0, days), nil
	}

	// Try common date formats
	formats := []string{
		"2006-01-02",
		"01-02-06",
		"1/2/2006",
		"01/02/2006",
		"2006/01/02",
		"02.01.2006",
		"2006.01.02",
	}
	for _, fmt := range formats {
		if t, err := time.Parse(fmt, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("neatpažintas datos formatas %q (naudokite YYYY-MM-DD)", s)
}

func findHeaders(rows [][]string) map[string]int {
	headers := make(map[string]int)
	if len(rows) == 0 {
		return headers
	}
	for i, cell := range rows[0] {
		headers[strings.TrimSpace(cell)] = i
	}
	return headers
}

func cellAt(row []string, col int) string {
	if col < 0 || col >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[col])
}

func cellName(col, row int) string {
	name, _ := excelize.CoordinatesToCellName(col, row)
	return name
}
