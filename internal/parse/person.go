package parse

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/dariuskasiulevicius/reitingtuvas/internal/source"
)

var legRegex = regexp.MustCompile(`(\d+)\s*leg`)

// LegRegex returns the compiled regex for matching "N leg" patterns.
func LegRegex() *regexp.Regexp { return legRegex }

// ParsePersonResults parses a person's results page and returns the event rows.
// classTmpl is used to extract event/class/day from class link hrefs.
// eventTmpl is used to extract event ids from event link hrefs.
func ParsePersonResults(r io.Reader, classTmpl, eventTmpl *source.Template) ([]PersonResultRow, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}

	table := doc.Find("table.w3-table.w3-striped.w3-bordered")
	if table.Length() == 0 {
		return nil, fmt.Errorf("page structure changed: no results table found")
	}

	var rows []PersonResultRow
	table.Find("tr").Each(func(i int, row *goquery.Selection) {
		cells := row.Find("td")
		if cells.Length() < 6 {
			return
		}

		var r PersonResultRow

		// Date
		dateStr := strings.TrimSpace(cells.Eq(0).Text())
		r.Date, _ = time.Parse("2006-01-02", dateStr)

		// Event name + ID
		eventCell := cells.Eq(1)
		eventLink := eventCell.Find("a").First()
		r.EventName = strings.TrimSpace(eventLink.Text())
		if href, ok := eventLink.Attr("href"); ok {
			vals := eventTmpl.Extract(href)
			if vals != nil {
				r.EventID = vals["event"]
			}
		}

		// Sport + Class
		sportCell := cells.Eq(2)
		sportImg := sportCell.Find("img")
		if sportImg.Length() > 0 {
			r.Sport, _ = sportImg.Attr("alt")
		}
		classLink := sportCell.Find("a")
		if classLink.Length() > 0 {
			r.Class = strings.TrimSpace(classLink.Text())
			r.ClassHref, _ = classLink.Attr("href")
		}

		// Place
		placeStr := strings.TrimSpace(cells.Eq(4).Text())
		r.Place, _ = strconv.Atoi(placeStr)

		// Result
		r.Result = strings.TrimSpace(cells.Eq(5).Text())

		// Day/Leg labels (column 6)
		if cells.Length() > 6 {
			dayCell := cells.Eq(6)
			dayHTML, _ := dayCell.Html()
			labels := splitByBr(dayHTML)
			for _, l := range labels {
				l = strings.TrimSpace(l)
				if l != "" {
					r.DayLabels = append(r.DayLabels, l)
				}
			}

			// Check if relay
			for _, l := range r.DayLabels {
				if legRegex.MatchString(l) {
					r.IsRelay = true
					break
				}
			}
		}

		// Per-day times (column 7)
		if cells.Length() > 7 {
			timeCell := cells.Eq(7)
			timeHTML, _ := timeCell.Html()
			times := splitByBr(timeHTML)
			for _, t := range times {
				t = cleanTimeCell(t)
				if t != "" {
					r.DayTimes = append(r.DayTimes, t)
				}
			}
		}

		rows = append(rows, r)
	})

	return rows, nil
}

// splitByBr splits HTML content by <br> or <br/> tags.
func splitByBr(html string) []string {
	html = strings.ReplaceAll(html, "<br/>", "<br>")
	html = strings.ReplaceAll(html, "<br />", "<br>")
	return strings.Split(html, "<br>")
}

// cleanTimeCell removes HTML tags, nbsp, split time links, and trims whitespace
// from a per-day time cell. Returns e.g. "1:14:36( 7)" or "dsq".
func cleanTimeCell(s string) string {
	// Remove <a> tags (split time links)
	re := regexp.MustCompile(`<[^>]+>`)
	s = re.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = strings.ReplaceAll(s, "\u00a0", " ")
	s = strings.TrimSpace(s)
	return s
}
