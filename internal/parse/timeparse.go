package parse

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseTime parses time strings in h:mm:ss or m:ss format into a Duration.
// Returns an error if the format is unrecognized.
//
// Examples: "1:23:45" → 1h23m45s, "45:30" → 45m30s, "5:02" → 5m2s.
func ParseTime(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty time string")
	}

	parts := strings.Split(s, ":")
	switch len(parts) {
	case 2:
		// m:ss
		m, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0, fmt.Errorf("parse minutes %q: %w", parts[0], err)
		}
		sec, err := strconv.Atoi(parts[1])
		if err != nil {
			return 0, fmt.Errorf("parse seconds %q: %w", parts[1], err)
		}
		return time.Duration(m)*time.Minute + time.Duration(sec)*time.Second, nil
	case 3:
		// h:mm:ss
		h, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0, fmt.Errorf("parse hours %q: %w", parts[0], err)
		}
		m, err := strconv.Atoi(parts[1])
		if err != nil {
			return 0, fmt.Errorf("parse minutes %q: %w", parts[1], err)
		}
		sec, err := strconv.Atoi(parts[2])
		if err != nil {
			return 0, fmt.Errorf("parse seconds %q: %w", parts[2], err)
		}
		return time.Duration(h)*time.Hour + time.Duration(m)*time.Minute + time.Duration(sec)*time.Second, nil
	default:
		return 0, fmt.Errorf("unrecognized time format %q (expected h:mm:ss or m:ss)", s)
	}
}

// ParseTimeOrStatus tries to parse a result cell value. If it's a valid time,
// returns status "ok" and the duration. Otherwise returns the lowercased text
// as the status and zero duration.
func ParseTimeOrStatus(s string) (status string, d time.Duration) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "not entered", 0
	}
	d, err := ParseTime(s)
	if err == nil {
		return "ok", d
	}
	return strings.ToLower(s), 0
}

// FormatDuration formats a duration as h:mm:ss, matching the site's display.
func FormatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}
