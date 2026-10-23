package loflv

import (
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// RelayTeam is one team in lof.lv relay results.
type RelayTeam struct {
	Place     int
	TeamName  string
	TotalTime time.Duration
	Legs      []RelayLeg
	Class     string
}

// RelayLeg is one leg in a relay team.
type RelayLeg struct {
	LegNumber int
	FirstName string
	LastName  string
	Time      time.Duration
}

// RelayBlock groups relay results for one class.
type RelayBlock struct {
	ClassName string
	Teams     []RelayTeam
}

// ParseRelayResults parses relay HTML from a lof.lv event response.
func ParseRelayResults(html string) ([]RelayBlock, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	var blocks []RelayBlock

	doc.Find("h2").Each(func(_ int, h2 *goquery.Selection) {
		anchor := h2.Find("a")
		if anchor.Length() == 0 {
			return
		}
		className := strings.TrimSpace(anchor.Text())
		if className == "" {
			return
		}

		block := RelayBlock{ClassName: className}

		// Find the table after this h2
		table := h2.NextAll().Filter("table").First()
		if table.Length() == 0 {
			return
		}

		var currentTeam *RelayTeam

		table.Find("tbody tr, tr").Each(func(_ int, tr *goquery.Selection) {
			if tr.HasClass("kom") {
				// Team row
				team := RelayTeam{Class: className}

				cells := tr.Find("td")
				if cells.Length() >= 3 {
					team.Place, _ = strconv.Atoi(strings.TrimSuffix(strings.TrimSpace(cells.Eq(0).Text()), "."))

					klubsCell := cells.Filter(".klubs").First()
					if klubsCell.Length() > 0 {
						team.TeamName = strings.TrimSpace(klubsCell.Text())
					}

					v1Cell := cells.Filter(".v1").First()
					if v1Cell.Length() > 0 {
						team.TotalTime = parseRelayTime(strings.TrimSpace(v1Cell.Text()))
					}
				}

				block.Teams = append(block.Teams, team)
				currentTeam = &block.Teams[len(block.Teams)-1]

			} else if tr.HasClass("dal") && currentTeam != nil {
				// Legs row — nested table
				tr.Find("table.t_dal tr").Each(func(_ int, legTr *goquery.Selection) {
					if legTr.Find("th").Length() > 0 {
						return // skip header
					}

					var leg RelayLeg

					// Leg number from first visible td
					legTr.Find("td").Each(func(i int, td *goquery.Selection) {
						if td.HasClass("v") {
							fullName := strings.TrimSpace(td.Text())
							leg.FirstName, leg.LastName = splitName(fullName)
						} else if td.HasClass("m") {
							text := strings.TrimSpace(td.Text())
							if t := parseRelayTime(text); t > 0 {
								leg.Time = t
							}
						} else if !td.HasClass("d-none") {
							// Might be leg number
							text := strings.TrimSuffix(strings.TrimSpace(td.Text()), ".")
							if n, err := strconv.Atoi(text); err == nil && n > 0 {
								leg.LegNumber = n
							}
						}
					})

					if leg.FirstName != "" || leg.LastName != "" {
						if leg.LegNumber == 0 {
							leg.LegNumber = len(currentTeam.Legs) + 1
						}
						currentTeam.Legs = append(currentTeam.Legs, leg)
					}
				})
			}
		})

		if len(block.Teams) > 0 {
			blocks = append(blocks, block)
		}
	})

	return blocks, nil
}

// parseRelayTime handles "H:MM:SS,00" or "0:46:50,00" format.
func parseRelayTime(s string) time.Duration {
	// Strip centiseconds
	if idx := strings.Index(s, ","); idx >= 0 {
		s = s[:idx]
	}
	return parseLVTime(s)
}
