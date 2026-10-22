package parse

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/dariuskasiulevicius/reitingtuvas/internal/source"
)

var timeWithPlaceRegex = regexp.MustCompile(`^([\d:]+)\s*(?:\(\s*(\d+)\s*\))?$`)

// dayColumn tracks where each day's data lives in the row.
type dayColumn struct {
	name      string
	colIdx    int  // index of the time column in td cells
	hasPoints bool // true when colspan=2 (points + time)
}

// ParseClassPage parses a class results page (single-day or multi-day total view).
// profileTmpl is used to extract sids from person href attributes.
func ParseClassPage(r io.Reader, profileTmpl *source.Template) (*ClassPage, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}

	page := &ClassPage{}

	// Event name from nodisplay h4
	doc.Find("h4.nodisplay").Each(func(i int, s *goquery.Selection) {
		if i == 0 {
			page.EventName = strings.TrimSpace(s.Text())
		}
	})

	table := doc.Find("table.w3-table.w3-bordered")
	if table.Length() == 0 {
		return nil, fmt.Errorf("page structure changed: no class results table found")
	}

	// Map header <th> elements to data column positions.
	// Some headers have colspan="2" (points-scored events) — each such header
	// occupies 2 <td> cells in data rows: points + time.
	headerRow := table.Find("tr").First()
	headers := headerRow.Find("th")

	// Build a column map: track the td-index each header maps to.
	type headerInfo struct {
		text    string
		colspan int
		tdStart int // starting td index for this header
	}
	var hdrMap []headerInfo
	tdIdx := 0
	headers.Each(func(i int, th *goquery.Selection) {
		cs := 1
		if csAttr, ok := th.Attr("colspan"); ok {
			cs, _ = strconv.Atoi(csAttr)
			if cs < 1 {
				cs = 1
			}
		}
		hdrMap = append(hdrMap, headerInfo{
			text:    strings.TrimSpace(th.Text()),
			colspan: cs,
			tdStart: tdIdx,
		})
		tdIdx += cs
	})

	// Fixed headers: Place(0), Name(1), Country(2), Club(3).
	// Everything from index 4+ is result columns.
	// Single-day without colspan: index 4 = Time, 5 = min/km
	// Multi-day without colspan: index 4 = total, 5+ = day names
	// Multi-day with colspan=2: index 4 = total (2 cols), 5+ = days (2 cols each)

	var dayCols []dayColumn
	isSingleDay := true

	if len(hdrMap) > 5 {
		// Check headers beyond fixed 4 + total
		for i := 5; i < len(hdrMap); i++ {
			name := hdrMap[i].text
			if name == "" || name == "min/km" {
				continue
			}
			isSingleDay = false
			dc := dayColumn{
				name:      name,
				hasPoints: hdrMap[i].colspan >= 2,
			}
			if dc.hasPoints {
				// Time is in the second sub-column
				dc.colIdx = hdrMap[i].tdStart + 1
			} else {
				dc.colIdx = hdrMap[i].tdStart
			}
			dayCols = append(dayCols, dc)
		}
	}

	// Check if "total" header (index 4) also has colspan
	totalHasPoints := false
	totalColIdx := 4
	if len(hdrMap) > 4 {
		if hdrMap[4].colspan >= 2 {
			totalHasPoints = true
		}
		totalColIdx = hdrMap[4].tdStart
	}
	// When totalHasPoints, the time is in the 2nd sub-column (tdStart+1)
	totalTimeIdx := totalColIdx
	if totalHasPoints {
		totalTimeIdx = totalColIdx + 1
	}

	if !isSingleDay {
		for _, dc := range dayCols {
			page.DayNames = append(page.DayNames, dc.name)
		}
	}

	// Parse competitor rows
	table.Find("tr").Each(func(i int, row *goquery.Selection) {
		if i == 0 {
			return // skip header
		}
		cells := row.Find("td")
		if cells.Length() < 4 {
			return
		}

		var c ClassCompetitor

		// Place
		placeText := strings.TrimSpace(cells.Eq(0).Text())
		c.Place, _ = strconv.Atoi(placeText)

		// Name + SID
		nameCell := cells.Eq(1)
		c.Name = strings.TrimSpace(nameCell.Text())
		nameLink := nameCell.Find("a")
		if nameLink.Length() > 0 {
			href, _ := nameLink.Attr("href")
			vals := profileTmpl.Extract(href)
			if vals != nil {
				c.SID = vals["sid"]
			}
		}

		if isSingleDay {
			if totalTimeIdx < cells.Length() {
				c.Total = cleanResultCell(strings.TrimSpace(cells.Eq(totalTimeIdx).Text()))
			}
		} else {
			if totalTimeIdx < cells.Length() {
				c.Total = cleanResultCell(strings.TrimSpace(cells.Eq(totalTimeIdx).Text()))
			}

			// Per-day results — read the time column for each day
			for _, dc := range dayCols {
				if dc.colIdx < cells.Length() {
					raw := cleanResultCell(strings.TrimSpace(cells.Eq(dc.colIdx).Text()))
					c.PerDay = append(c.PerDay, raw)
				} else {
					c.PerDay = append(c.PerDay, "")
				}
			}
		}

		page.Rows = append(page.Rows, c)
	})

	return page, nil
}

// ParseDayTimeAndPlace extracts time and place from a per-day cell value
// like "1:14:36( 7)" or "dsq".
func ParseDayTimeAndPlace(s string) (timeStr string, place int) {
	s = strings.TrimSpace(s)
	m := timeWithPlaceRegex.FindStringSubmatch(s)
	if m != nil {
		timeStr = m[1]
		if m[2] != "" {
			place, _ = strconv.Atoi(m[2])
		}
		return
	}
	// No match — treat entire string as time/status
	return s, 0
}

// cleanResultCell removes nbsp and excess whitespace from a result cell.
func cleanResultCell(s string) string {
	s = strings.ReplaceAll(s, "\u00a0", " ")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = strings.TrimSpace(s)
	return s
}
