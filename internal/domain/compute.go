package domain

import "time"

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
