package domain

import (
	"sort"
	"strconv"
	"strings"
	"time"
)

// ComputeParticipantResult sums ok Day times.
func ComputeParticipantResult(days []DayResult) time.Duration {
	var total time.Duration
	for _, d := range days {
		if d.Status == "ok" {
			total += d.Time
		}
	}
	return total
}

// ComputeLeaderResult sums all Day Leader Results regardless of participant status.
func ComputeLeaderResult(days []DayResult) time.Duration {
	var total time.Duration
	for _, d := range days {
		total += d.LeaderTime
	}
	return total
}

// EstimateDayDate returns eventDate + (dayNumber-1) days.
func EstimateDayDate(eventDate time.Time, dayNumber int) time.Time {
	return eventDate.AddDate(0, 0, dayNumber-1)
}

// IsOpenClass returns true if the class name contains "open" (case-insensitive).
func IsOpenClass(class string) bool {
	return strings.Contains(strings.ToLower(class), "open")
}

// ComputeDayScore returns leader/time × coefficient. Zero when status != "ok".
func ComputeDayScore(r StoredResult, eventCoef, openClassCoef float64) float64 {
	if r.Status != "ok" || r.Time <= 0 || r.LeaderTime <= 0 {
		return 0
	}
	coef := eventCoef
	if IsOpenClass(r.Class) {
		coef = openClassCoef
	}
	return float64(r.LeaderTime) / float64(r.Time) * coef
}

// ComputeSeasonScore returns the sum of the top N scores.
func ComputeSeasonScore(scores []float64, topN int) float64 {
	sorted := make([]float64, len(scores))
	copy(sorted, scores)
	sort.Float64s(sorted)

	sum := 0.0
	count := 0
	for i := len(sorted) - 1; i >= 0 && count < topN; i-- {
		sum += sorted[i]
		count++
	}
	return sum
}

// TopNDayKeys returns the set of day keys whose scores are in the top N.
// Ties at the boundary are all included.
func TopNDayKeys(scores map[string]float64, topN int) map[string]bool {
	if len(scores) == 0 || topN <= 0 {
		return nil
	}

	vals := make([]float64, 0, len(scores))
	for _, v := range scores {
		vals = append(vals, v)
	}
	sort.Float64s(vals)

	// threshold = the Nth value from the top
	idx := len(vals) - topN
	if idx < 0 {
		idx = 0
	}
	threshold := vals[idx]

	result := make(map[string]bool)
	for k, v := range scores {
		if v >= threshold && v > 0 {
			result[k] = true
		}
	}
	return result
}

// RatingEntry is one row in the Season Rating.
type RatingEntry struct {
	SID         string
	Surname     string
	Name        string
	DayScores   map[string]float64 // key = "eventID:dayNumber"
	StartCount  int
	SeasonScore float64
	Place       string // "1", "2", "3-4", ...
}

// ComputeRating builds the full Season Rating from stored results.
func ComputeRating(
	results []StoredResult,
	participants []Participant,
	coefficients map[string]float64,
	openClassCoef float64,
	topN int,
) []RatingEntry {
	participantMap := make(map[string]Participant)
	for _, p := range participants {
		if p.SID != "" {
			participantMap[p.SID] = p
		}
	}

	type sidData struct {
		scores    map[string]float64
		allScores []float64
		starts    int
	}
	bySID := make(map[string]*sidData)

	for _, r := range results {
		sd, ok := bySID[r.SID]
		if !ok {
			sd = &sidData{scores: make(map[string]float64)}
			bySID[r.SID] = sd
		}

		eventCoef := 1.0
		if c, ok := coefficients[r.EventID]; ok {
			eventCoef = c
		}
		score := ComputeDayScore(r, eventCoef, openClassCoef)
		key := r.EventID + ":" + itoa(r.DayNumber)
		sd.scores[key] = score
		sd.allScores = append(sd.allScores, score)
		sd.starts++
	}

	var entries []RatingEntry
	for sid, sd := range bySID {
		p := participantMap[sid]
		entries = append(entries, RatingEntry{
			SID:         sid,
			Surname:     p.Surname,
			Name:        p.Name,
			DayScores:   sd.scores,
			StartCount:  sd.starts,
			SeasonScore: ComputeSeasonScore(sd.allScores, topN),
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].SeasonScore != entries[j].SeasonScore {
			return entries[i].SeasonScore > entries[j].SeasonScore
		}
		return entries[i].Surname+entries[i].Name < entries[j].Surname+entries[j].Name
	})

	AssignPlaces(entries)
	return entries
}

// AssignPlaces sets Place strings, handling ties.
func AssignPlaces(entries []RatingEntry) {
	i := 0
	for i < len(entries) {
		j := i + 1
		for j < len(entries) && entries[j].SeasonScore == entries[i].SeasonScore {
			j++
		}
		var place string
		if j-i == 1 {
			place = itoa(i + 1)
		} else {
			place = itoa(i+1) + "-" + itoa(j)
		}
		for k := i; k < j; k++ {
			entries[k].Place = place
		}
		i = j
	}
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
