package parse

import (
	"testing"
	"time"
)

func TestParseTime(t *testing.T) {
	tests := []struct {
		input string
		want  time.Duration
	}{
		{"45:30", 45*time.Minute + 30*time.Second},
		{"5:02", 5*time.Minute + 2*time.Second},
		{"1:23:45", 1*time.Hour + 23*time.Minute + 45*time.Second},
		{"0:00:01", 1 * time.Second},
		{"12:00:00", 12 * time.Hour},
	}
	for _, tt := range tests {
		got, err := ParseTime(tt.input)
		if err != nil {
			t.Errorf("ParseTime(%q) error: %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseTime(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestParseTime_Errors(t *testing.T) {
	bad := []string{"", "dsq", "dns", "abc", "1:2:3:4", "mp"}
	for _, s := range bad {
		_, err := ParseTime(s)
		if err == nil {
			t.Errorf("ParseTime(%q) should fail", s)
		}
	}
}

func TestParseTimeOrStatus(t *testing.T) {
	tests := []struct {
		input      string
		wantStatus string
		wantTime   time.Duration
	}{
		{"1:23:45", "ok", 1*time.Hour + 23*time.Minute + 45*time.Second},
		{"45:30", "ok", 45*time.Minute + 30*time.Second},
		{"dsq", "dsq", 0},
		{"DNS", "dns", 0},
		{"mp", "mp", 0},
		{"", "not entered", 0},
		{"  dsq  ", "dsq", 0},
	}
	for _, tt := range tests {
		status, d := ParseTimeOrStatus(tt.input)
		if status != tt.wantStatus {
			t.Errorf("ParseTimeOrStatus(%q) status = %q, want %q", tt.input, status, tt.wantStatus)
		}
		if d != tt.wantTime {
			t.Errorf("ParseTimeOrStatus(%q) duration = %v, want %v", tt.input, d, tt.wantTime)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{1*time.Hour + 23*time.Minute + 45*time.Second, "1:23:45"},
		{45*time.Minute + 30*time.Second, "45:30"},
		{5*time.Minute + 2*time.Second, "5:02"},
		{0, "0:00"},
	}
	for _, tt := range tests {
		got := FormatDuration(tt.d)
		if got != tt.want {
			t.Errorf("FormatDuration(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}
