package loflv

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// EventRow is one competitor row from lof.lv event results.
type EventRow struct {
	Place     int
	FirstName string
	LastName  string
	Club      string
	Time      time.Duration
	RawTime   string
	Status    string // "ok", "dns", "dsq", "mp", "dnf", "-" (out of competition)
	Class     string
}

// ClassBlock groups results for one class within a lof.lv event response.
type ClassBlock struct {
	ClassName string
	Rows      []EventRow
}

var classHeaderRe = regexp.MustCompile(`^([A-Za-z0-9]+)`)

// ParseEventResults parses single-event results HTML (from r=1 response).
// The HTML contains all classes as <h2> + <table> pairs.
func ParseEventResults(html string) ([]ClassBlock, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	var blocks []ClassBlock

	doc.Find("h2").Each(func(_ int, h2 *goquery.Selection) {
		anchor := h2.Find("a")
		if anchor.Length() == 0 {
			return
		}
		className := strings.TrimSpace(anchor.Text())
		if className == "" {
			return
		}

		table := h2.NextFiltered("table")
		if table.Length() == 0 {
			// try next sibling of any type that is a table
			table = h2.NextAll().Filter("table").First()
		}
		if table.Length() == 0 {
			return
		}

		block := ClassBlock{ClassName: className}

		table.Find("tbody tr").Each(func(_ int, tr *goquery.Selection) {
			if tr.HasClass("protFooter") {
				return
			}

			cells := tr.Find("td")
			if cells.Length() < 7 {
				return
			}

			var row EventRow

			// Place: "1." → 1
			placeText := strings.TrimSpace(cells.Eq(0).Text())
			placeText = strings.TrimSuffix(placeText, ".")
			row.Place, _ = strconv.Atoi(placeText)

			// Name: "Vārds Uzvārds" in td.b
			fullName := strings.TrimSpace(cells.Filter(".b").First().Text())
			if fullName == "" {
				fullName = strings.TrimSpace(cells.Eq(2).Text())
			}
			row.FirstName, row.LastName = splitName(fullName)

			// Club
			row.Club = strings.TrimSpace(cells.Eq(3).Text())

			// Time: "00:28:21"
			row.RawTime = strings.TrimSpace(cells.Eq(4).Text())
			row.Time = parseLVTime(row.RawTime)

			// Status
			statusText := strings.TrimSpace(cells.Eq(5).Text())
			row.Status = normalizeLVStatus(statusText, row.Place)

			// Class from td.td_grupa
			grupaCell := cells.Filter(".td_grupa").First()
			if grupaCell.Length() > 0 {
				row.Class = strings.TrimSpace(grupaCell.Text())
			}

			block.Rows = append(block.Rows, row)
		})

		if len(block.Rows) > 0 {
			blocks = append(blocks, block)
		}
	})

	return blocks, nil
}

// splitName splits "FirstName LastName" → (firstName, lastName).
// Last word is the surname. Handles compound first names like "Anna Emīlija".
func splitName(full string) (firstName, lastName string) {
	parts := strings.Fields(full)
	if len(parts) == 0 {
		return "", ""
	}
	if len(parts) == 1 {
		return parts[0], ""
	}
	lastName = parts[len(parts)-1]
	firstName = strings.Join(parts[:len(parts)-1], " ")
	return
}

// parseLVTime parses "00:28:21" or "0:28:21" into duration.
// Returns 0 for non-time strings (dns, dsq, etc.).
func parseLVTime(s string) time.Duration {
	s = strings.TrimSpace(s)
	parts := strings.Split(s, ":")
	if len(parts) != 3 {
		return 0
	}
	h, err1 := strconv.Atoi(parts[0])
	m, err2 := strconv.Atoi(parts[1])
	sec, err3 := strconv.Atoi(parts[2])
	if err1 != nil || err2 != nil || err3 != nil {
		return 0
	}
	return time.Duration(h)*time.Hour + time.Duration(m)*time.Minute + time.Duration(sec)*time.Second
}

func normalizeLVStatus(status string, place int) string {
	status = strings.ToLower(strings.TrimSpace(status))
	switch status {
	case "":
		if place > 0 {
			return "ok"
		}
		return "dns" // no place, no status → likely DNS
	case "-":
		return "-" // out of competition
	case "dns", "dsq", "mp", "dnf":
		return status
	default:
		return status
	}
}
