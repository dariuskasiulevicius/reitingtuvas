package parse

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/dariuskasiulevicius/reitingtuvas/internal/source"
)

// ParseSearch parses a search results page and returns the candidates found.
// profileTmpl is used to extract sids from profile href attributes.
func ParseSearch(r io.Reader, profileTmpl *source.Template) ([]SearchCandidate, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}

	container := doc.Find("div.w3-container")
	if container.Length() == 0 {
		return nil, fmt.Errorf("page structure changed: no w3-container found")
	}

	// Check for "No participants found"
	text := container.First().Text()
	if strings.Contains(text, "No participants found") ||
		strings.Contains(text, "Dalyvių nerasta") {
		return nil, nil
	}

	table := container.Find("table.w3-table")
	if table.Length() == 0 {
		return nil, fmt.Errorf("page structure changed: no results table found")
	}

	var candidates []SearchCandidate
	table.Find("tr").Each(func(i int, row *goquery.Selection) {
		cells := row.Find("td")
		if cells.Length() < 2 {
			return // header row or malformed
		}

		nameCell := cells.First()
		eventsCell := cells.Eq(1)

		var c SearchCandidate
		c.DisplayName = strings.TrimSpace(nameCell.Text())

		// Try to extract SID from profile link
		nameLink := nameCell.Find("a")
		if nameLink.Length() > 0 {
			href, _ := nameLink.Attr("href")
			vals := profileTmpl.Extract(href)
			if vals != nil {
				c.SID = vals["sid"]
			}
		}

		// Event count from the events cell
		evText := strings.TrimSpace(eventsCell.Text())
		c.EventCount, _ = strconv.Atoi(evText)

		candidates = append(candidates, c)
	})

	return candidates, nil
}
