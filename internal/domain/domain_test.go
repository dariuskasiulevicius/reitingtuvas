package domain

import (
	"testing"
	"time"
)

func TestComputeParticipantResult_SumsOkOnly(t *testing.T) {
	days := []DayResult{
		{DayNumber: 1, Status: "ok", Time: 45 * time.Minute},
		{DayNumber: 2, Status: "dsq"},
		{DayNumber: 3, Status: "ok", Time: 30 * time.Minute},
	}
	got := ComputeParticipantResult(days)
	want := 75 * time.Minute
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestComputeParticipantResult_AllDsq(t *testing.T) {
	days := []DayResult{
		{DayNumber: 1, Status: "dsq"},
		{DayNumber: 2, Status: "dns"},
	}
	got := ComputeParticipantResult(days)
	if got != 0 {
		t.Errorf("got %v, want 0", got)
	}
}

func TestComputeLeaderResult_SumsAllDays(t *testing.T) {
	days := []DayResult{
		{DayNumber: 1, Status: "ok", LeaderTime: 40 * time.Minute},
		{DayNumber: 2, Status: "not entered", LeaderTime: 35 * time.Minute},
		{DayNumber: 3, Status: "ok", LeaderTime: 30 * time.Minute},
	}
	got := ComputeLeaderResult(days)
	want := 105 * time.Minute
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestComputeLeaderResult_IncludesDsqDays(t *testing.T) {
	// Leader result includes leader time for days where participant got dsq
	days := []DayResult{
		{DayNumber: 1, Status: "ok", LeaderTime: 28 * time.Minute},
		{DayNumber: 2, Status: "dsq", LeaderTime: 50 * time.Minute},
	}
	got := ComputeLeaderResult(days)
	want := 78 * time.Minute
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestDayResult_DsqHasNoTime(t *testing.T) {
	d := DayResult{Status: "dsq"}
	if d.Time != 0 {
		t.Errorf("dsq day has non-zero time: %v", d.Time)
	}
}

func TestTopNDayKeys(t *testing.T) {
	scores := map[string]float64{
		"a:1": 0.90,
		"b:1": 0.80,
		"c:1": 0.70,
		"d:1": 0.60,
		"e:1": 0.50,
	}
	got := TopNDayKeys(scores, 3)
	if len(got) != 3 {
		t.Fatalf("got %d keys, want 3", len(got))
	}
	for _, k := range []string{"a:1", "b:1", "c:1"} {
		if !got[k] {
			t.Errorf("missing key %s", k)
		}
	}
}

func TestTopNDayKeys_Ties(t *testing.T) {
	scores := map[string]float64{
		"a:1": 0.90,
		"b:1": 0.80,
		"c:1": 0.80, // tie at boundary
		"d:1": 0.60,
	}
	got := TopNDayKeys(scores, 2)
	// threshold=0.80 → all with >=0.80 included (a, b, c)
	if len(got) != 3 {
		t.Fatalf("got %d keys, want 3 (ties)", len(got))
	}
}

func TestTopNDayKeys_FewerThanN(t *testing.T) {
	scores := map[string]float64{
		"a:1": 0.90,
		"b:1": 0.80,
	}
	got := TopNDayKeys(scores, 8)
	if len(got) != 2 {
		t.Fatalf("got %d keys, want 2", len(got))
	}
}

func TestTopNDayKeys_ZeroScoresExcluded(t *testing.T) {
	scores := map[string]float64{
		"a:1": 0.90,
		"b:1": 0.0,
		"c:1": 0.0,
	}
	got := TopNDayKeys(scores, 3)
	if len(got) != 1 {
		t.Fatalf("got %d keys, want 1 (zero excluded)", len(got))
	}
}

func TestEstimateDayDate(t *testing.T) {
	base := time.Date(2026, 7, 25, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		dayNum int
		want   string
	}{
		{1, "2026-07-25"},
		{2, "2026-07-26"},
		{3, "2026-07-27"},
	}
	for _, tt := range tests {
		got := EstimateDayDate(base, tt.dayNum)
		if got.Format("2006-01-02") != tt.want {
			t.Errorf("day %d: got %s, want %s", tt.dayNum, got.Format("2006-01-02"), tt.want)
		}
	}
}
