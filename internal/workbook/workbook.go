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
	}
	defaults := map[string]string{
		texts.SettingDelay:          "200",
		texts.SettingCacheFinalDays: "14",
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
