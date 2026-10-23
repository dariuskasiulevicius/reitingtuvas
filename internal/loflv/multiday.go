package loflv

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// MultiDayRow is one competitor in a kopvērtējums.
type MultiDayRow struct {
	Place     int
	LastName  string
	FirstName string
	Club      string
	TotalTime time.Duration
	Days      []DayResult
	Class     string
	Status    string
}

// DayResult is one day's time+place within a kopvērtējums row.
type DayResult struct {
	Time  time.Duration
	Place int
}

// MultiDayBlock groups kopvērtējums results for one class.
type MultiDayBlock struct {
	ClassName string
	DayCount  int
	Rows      []MultiDayRow
}

var dayPlaceRe = regexp.MustCompile(`\((\d+)\)`)

// ParseMultiDayResults parses kopvērtējums HTML (from r=99 response).
func ParseMultiDayResults(html string) ([]MultiDayBlock, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	var blocks []MultiDayBlock

	doc.Find("div.div_gr_rez").Each(func(_ int, div *goquery.Selection) {
		h2 := div.Find("h2")
		if h2.Length() == 0 {
			return
		}
		anchor := h2.Find("a")
		className := strings.TrimSpace(anchor.Text())
		if className == "" {
			className = strings.TrimSpace(h2.Text())
		}

		table := div.Find("table")
		if table.Length() == 0 {
			return
		}

		// Count day columns from header: th.th_d1, th.th_d2, ...
		dayCount := 0
		table.Find("thead th[colspan]").Each(func(_ int, th *goquery.Selection) {
			cls, _ := th.Attr("class")
			if strings.Contains(cls, "th_d") {
				dayCount++
			}
		})
		if dayCount == 0 {
			dayCount = countDayColumns(table)
		}

		block := MultiDayBlock{ClassName: className, DayCount: dayCount}

		table.Find("tbody tr").Each(func(_ int, tr *goquery.Selection) {
			if tr.HasClass("protFooter") {
				return
			}

			var row MultiDayRow

			// Place from td.vi
			viCell := tr.Find("td.vi")
			if viCell.Length() > 0 {
				placeText := strings.TrimSuffix(strings.TrimSpace(viCell.Text()), ".")
				row.Place, _ = strconv.Atoi(placeText)
			}

			// Surname from td.u
			uCell := tr.Find("td.u")
			if uCell.Length() > 0 {
				row.LastName = strings.TrimSpace(uCell.Text())
			}

			// First name from td.v
			vCell := tr.Find("td.v")
			if vCell.Length() > 0 {
				row.FirstName = strings.TrimSpace(vCell.Text())
			}

			// Club
			kCell := tr.Find("td.klubs")
			if kCell.Length() > 0 {
				row.Club = strings.TrimSpace(kCell.Text())
			}

			// Total time from td.r
			rCell := tr.Find("td.r")
			if rCell.Length() > 0 {
				raw := strings.TrimSpace(rCell.Text())
				row.TotalTime = parseLVTime(raw)
			}

			// Per-day results: all day cells share class td_d1_rez / td_d1_vieta,
			// so we parse positionally (index 0 = day 1, etc).
			rezCells := tr.Find("td.td_d1_rez")
			vietaCells := tr.Find("td.td_d1_vieta")
			for d := 0; d < dayCount; d++ {
				var dr DayResult
				if d < rezCells.Length() {
					dr.Time = parseLVTime(strings.TrimSpace(rezCells.Eq(d).Text()))
				}
				if d < vietaCells.Length() {
					m := dayPlaceRe.FindStringSubmatch(vietaCells.Eq(d).Text())
					if m != nil {
						dr.Place, _ = strconv.Atoi(m[1])
					}
				}
				row.Days = append(row.Days, dr)
			}

			// Class from td.gr1
			gr1Cell := tr.Find("td.gr1")
			if gr1Cell.Length() > 0 {
				row.Class = strings.TrimSpace(gr1Cell.Text())
			}

			if row.TotalTime > 0 {
				row.Status = "ok"
			} else if row.Place == 0 {
				row.Status = "dns"
			}

			block.Rows = append(block.Rows, row)
		})

		if len(block.Rows) > 0 {
			blocks = append(blocks, block)
		}
	})

	// Fallback: if no div.div_gr_rez found, try direct h2+table pattern
	if len(blocks) == 0 {
		blocks = parseMultiDayFallback(doc)
	}

	return blocks, nil
}

func parseMultiDayFallback(doc *goquery.Document) []MultiDayBlock {
	var blocks []MultiDayBlock

	doc.Find("h2").Each(func(_ int, h2 *goquery.Selection) {
		anchor := h2.Find("a")
		if anchor.Length() == 0 {
			return
		}
		className := strings.TrimSpace(anchor.Text())
		table := h2.NextAll().Filter("table").First()
		if table.Length() == 0 {
			return
		}

		dayCount := countDayColumns(table)
		block := MultiDayBlock{ClassName: className, DayCount: dayCount}

		table.Find("tbody tr").Each(func(_ int, tr *goquery.Selection) {
			if tr.HasClass("protFooter") {
				return
			}
			cells := tr.Find("td")
			if cells.Length() < 5 {
				return
			}

			var row MultiDayRow
			row.Place, _ = strconv.Atoi(strings.TrimSuffix(strings.TrimSpace(cells.Eq(0).Text()), "."))
			row.LastName = strings.TrimSpace(cells.Eq(1).Text())
			row.FirstName = strings.TrimSpace(cells.Eq(2).Text())
			row.Club = strings.TrimSpace(cells.Eq(3).Text())
			row.TotalTime = parseLVTime(strings.TrimSpace(cells.Eq(4).Text()))

			// Day columns start at index 5, pairs of (time, place)
			idx := 5
			for d := 0; d < dayCount; d++ {
				var dr DayResult
				if idx < cells.Length() {
					dr.Time = parseLVTime(strings.TrimSpace(cells.Eq(idx).Text()))
					idx++
				}
				if idx < cells.Length() {
					m := dayPlaceRe.FindStringSubmatch(cells.Eq(idx).Text())
					if m != nil {
						dr.Place, _ = strconv.Atoi(m[1])
					}
					idx++
				}
				row.Days = append(row.Days, dr)
			}

			if row.TotalTime > 0 {
				row.Status = "ok"
			}

			block.Rows = append(block.Rows, row)
		})

		if len(block.Rows) > 0 {
			blocks = append(blocks, block)
		}
	})

	return blocks
}

func countDayColumns(table *goquery.Selection) int {
	count := 0
	table.Find("thead th").Each(func(_ int, th *goquery.Selection) {
		text := strings.TrimSpace(th.Text())
		if strings.HasPrefix(text, "D") && len(text) <= 3 {
			if _, err := strconv.Atoi(text[1:]); err == nil {
				count++
			}
		}
	})
	if count == 0 {
		// Fallback: count colspan=2 headers that look like day headers
		table.Find("thead th[colspan='2']").Each(func(_ int, th *goquery.Selection) {
			cls, _ := th.Attr("class")
			if strings.Contains(cls, "th_d") {
				count++
			}
		})
	}
	return count
}
