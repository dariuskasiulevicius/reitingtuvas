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

	// Detect day names from header row.
	// Single-day: Place | Name | Country | Club | Time | min/km
	// Multi-day:  Place | Name | Country | Club | total | Day1name | Day2name | ...
	headerRow := table.Find("tr").First()
	headers := headerRow.Find("th")
	headerCount := headers.Length()

	// Check if multi-day: headers beyond the 5th column (index 4+) that
	// aren't "min/km" are day names.
	if headerCount > 5 {
		for j := 5; j < headerCount; j++ {
			name := strings.TrimSpace(headers.Eq(j).Text())
			if name != "" && name != "min/km" {
				page.DayNames = append(page.DayNames, name)
			}
		}
	}

	isMultiDay := len(page.DayNames) > 0

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

		if isMultiDay {
			// Column 4 = total
			c.Total = cleanResultCell(strings.TrimSpace(cells.Eq(4).Text()))

			// Columns 5+ = per-day results
			for j := 0; j < len(page.DayNames); j++ {
				colIdx := 5 + j
				if colIdx < cells.Length() {
					raw := cleanResultCell(strings.TrimSpace(cells.Eq(colIdx).Text()))
					c.PerDay = append(c.PerDay, raw)
				}
			}
		} else {
			// Single-day: column 4 = time
			if cells.Length() > 4 {
				c.Total = cleanResultCell(strings.TrimSpace(cells.Eq(4).Text()))
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
