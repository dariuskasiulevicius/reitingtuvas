package parse

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/dariuskasiulevicius/reitingtuvas/internal/source"
)

// ParseRelayClassPage parses a relay class results page.
// profileTmpl is used to extract sids from person href attributes.
func ParseRelayClassPage(r io.Reader, profileTmpl *source.Template) ([]RelayTeam, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}

	table := doc.Find("table.w3-table.w3-bordered")
	if table.Length() == 0 {
		return nil, fmt.Errorf("page structure changed: no relay results table found")
	}

	var teams []RelayTeam
	var current *RelayTeam

	table.Find("tr").Each(func(i int, row *goquery.Selection) {
		if i == 0 {
			// Skip header row
			if row.Find("th").Length() > 0 {
				return
			}
		}

		cells := row.Find("td")
		if cells.Length() < 3 {
			return
		}

		firstCell := cells.Eq(0)
		secondCell := cells.Eq(1)

		// Team header row: first cell has <b> with place number,
		// second cell has <b> with team name.
		if firstCell.Find("b").Length() > 0 && secondCell.Find("b").Length() > 0 {
			team := RelayTeam{
				TeamName:  strings.TrimSpace(secondCell.Find("b").Text()),
				TotalTime: cleanResultCell(strings.TrimSpace(cells.Eq(3).Find("b").Text())),
			}
			placeStr := strings.TrimSpace(firstCell.Find("b").Text())
			team.Place, _ = strconv.Atoi(placeStr)

			teams = append(teams, team)
			current = &teams[len(teams)-1]
			return
		}

		if current == nil {
			return
		}

		// Leg row: first cell = leg number (right-aligned), second cell = person
		leg := RelayLeg{}
		legStr := strings.TrimSpace(firstCell.Text())
		leg.LegNumber, _ = strconv.Atoi(legStr)
		if leg.LegNumber == 0 {
			return
		}

		// Person name + SID
		nameLink := secondCell.Find("a")
		if nameLink.Length() > 0 {
			leg.Name = strings.TrimSpace(nameLink.Text())
			href, _ := nameLink.Attr("href")
			vals := profileTmpl.Extract(href)
			if vals != nil {
				leg.SID = vals["sid"]
			}
		} else {
			leg.Name = strings.TrimSpace(secondCell.Text())
		}

		// Time + place from column 3 (format: "38:45 ( 1)")
		if cells.Length() > 3 {
			raw := cleanResultCell(strings.TrimSpace(cells.Eq(3).Text()))
			timeStr, place := ParseDayTimeAndPlace(raw)
			leg.Time = timeStr
			leg.Place = place
		}

		current.Legs = append(current.Legs, leg)
		// Update the pointer in the slice
		teams[len(teams)-1] = *current
	})

	return teams, nil
}
