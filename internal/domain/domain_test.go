package domain

import (
	"testing"
	"time"
)

func TestEventResult_SumsOkDayTimes(t *testing.T) {
	er := EventResult{
		Days: []DayResult{
			{DayNumber: 1, Status: "ok", Time: 45 * time.Minute},
			{DayNumber: 2, Status: "dsq"},
			{DayNumber: 3, Status: "ok", Time: 30 * time.Minute},
		},
	}

	var got time.Duration
	for _, d := range er.Days {
		if d.Status == "ok" {
			got += d.Time
		}
	}

	want := 75 * time.Minute
	if got != want {
		t.Errorf("participant result = %v, want %v", got, want)
	}
}

func TestDayResult_DsqHasNoTime(t *testing.T) {
	d := DayResult{Status: "dsq"}
	if d.Time != 0 {
		t.Errorf("dsq day has non-zero time: %v", d.Time)
	}
}

func TestEventResult_LeaderResultSumsAllDays(t *testing.T) {
	days := []DayResult{
		{DayNumber: 1, Status: "ok", LeaderTime: 40 * time.Minute},
		{DayNumber: 2, Status: "not entered", LeaderTime: 35 * time.Minute},
		{DayNumber: 3, Status: "ok", LeaderTime: 30 * time.Minute},
	}

	var got time.Duration
	for _, d := range days {
		got += d.LeaderTime
	}

	want := 105 * time.Minute
	if got != want {
		t.Errorf("leader result = %v, want %v", got, want)
	}
}
