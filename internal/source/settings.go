package source

import (
	"fmt"
	"time"
)

// Settings holds the configuration read from the workbook's Nustatymai sheet.
type Settings struct {
	BaseURL        string        // e.g. "https://results.example.org"
	Templates      *Templates    // parsed URL templates
	PeriodFrom     time.Time     // inclusive
	PeriodTo       time.Time     // inclusive
	Delay          time.Duration // pause between requests
	CacheFinalDays int           // events older than this many days are cached forever
}

// Validate checks that all required settings are present and sensible.
func (s *Settings) Validate() error {
	if s.BaseURL == "" {
		return fmt.Errorf("base URL is empty")
	}
	if s.Templates == nil {
		return fmt.Errorf("templates not configured")
	}
	if s.PeriodFrom.IsZero() || s.PeriodTo.IsZero() {
		return fmt.Errorf("period dates not set")
	}
	if s.PeriodTo.Before(s.PeriodFrom) {
		return fmt.Errorf("period end %s is before start %s", s.PeriodTo.Format("2006-01-02"), s.PeriodFrom.Format("2006-01-02"))
	}
	if s.Delay < 0 {
		return fmt.Errorf("delay cannot be negative")
	}
	return nil
}

// InPeriod reports whether the given event date falls within the period (inclusive).
func (s *Settings) InPeriod(eventDate time.Time) bool {
	d := eventDate.Truncate(24 * time.Hour)
	from := s.PeriodFrom.Truncate(24 * time.Hour)
	to := s.PeriodTo.Truncate(24 * time.Hour)
	return !d.Before(from) && !d.After(to)
}
