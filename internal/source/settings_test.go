package source

import (
	"testing"
	"time"
)

func TestSettings_Validate(t *testing.T) {
	good := Settings{
		BaseURL:        FakeBaseURL,
		Templates:      FakeTemplates(),
		PeriodFrom:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		PeriodTo:       time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		Delay:          200 * time.Millisecond,
		CacheFinalDays: 14,
	}
	if err := good.Validate(); err != nil {
		t.Fatalf("valid settings: %v", err)
	}

	tests := []struct {
		name   string
		modify func(*Settings)
	}{
		{"no base url", func(s *Settings) { s.BaseURL = "" }},
		{"no templates", func(s *Settings) { s.Templates = nil }},
		{"no period from", func(s *Settings) { s.PeriodFrom = time.Time{} }},
		{"no period to", func(s *Settings) { s.PeriodTo = time.Time{} }},
		{"period reversed", func(s *Settings) {
			s.PeriodFrom, s.PeriodTo = s.PeriodTo, s.PeriodFrom
		}},
		{"negative delay", func(s *Settings) { s.Delay = -1 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := good // copy
			tt.modify(&s)
			if err := s.Validate(); err == nil {
				t.Error("expected validation error")
			}
		})
	}
}

func TestSettings_InPeriod(t *testing.T) {
	s := Settings{
		PeriodFrom: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		PeriodTo:   time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC),
	}
	tests := []struct {
		date time.Time
		want bool
	}{
		{time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC), false},
		{time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), true},
		{time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC), true},
		{time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC), true},
		{time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), false},
	}
	for _, tt := range tests {
		got := s.InPeriod(tt.date)
		if got != tt.want {
			t.Errorf("InPeriod(%s) = %v, want %v", tt.date.Format("2006-01-02"), got, tt.want)
		}
	}
}
