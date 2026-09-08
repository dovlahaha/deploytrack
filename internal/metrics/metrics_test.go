package metrics

import (
	"math"
	"testing"
	"time"
)

func almost(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 0.0001 {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestCalculate_Empty(t *testing.T) {
	s := Calculate(nil, 7*24*time.Hour)
	if s.TotalDeployments != 0 {
		t.Errorf("expected 0 deployments, got %d", s.TotalDeployments)
	}
	almost(t, s.ChangeFailureRate, 0)
	if s.Rating != "low" {
		t.Errorf("empty set should rate low, got %q", s.Rating)
	}
}

func TestCalculate_FrequencyAndFailureRate(t *testing.T) {
	now := time.Now()
	events := []Event{
		{Status: StatusSuccess, At: now},
		{Status: StatusSuccess, At: now},
		{Status: StatusFailed, At: now},
		{Status: StatusRolledBack, At: now},
	}

	s := Calculate(events, 7*24*time.Hour)

	if s.TotalDeployments != 4 {
		t.Errorf("total: got %d, want 4", s.TotalDeployments)
	}
	if s.FailedDeployments != 2 {
		t.Errorf("failed: got %d, want 2 (failed + rolled_back)", s.FailedDeployments)
	}
	almost(t, s.PerWeek, 4)
	almost(t, s.ChangeFailureRate, 0.5)
}

func TestCalculate_Rating(t *testing.T) {
	now := time.Now()
	week := 7 * 24 * time.Hour

	cases := []struct {
		name  string
		count int
		want  string
	}{
		{"daily or better is elite", 7, "elite"},
		{"weekly is high", 1, "high"},
		{"nothing is low", 0, "low"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var events []Event
			for i := 0; i < tc.count; i++ {
				events = append(events, Event{Status: StatusSuccess, At: now})
			}
			if got := Calculate(events, week).Rating; got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestCalculate_ZeroWindowDoesNotDivideByZero(t *testing.T) {
	s := Calculate([]Event{{Status: StatusSuccess, At: time.Now()}}, 0)
	if math.IsInf(s.PerDay, 0) || math.IsNaN(s.PerDay) {
		t.Errorf("per-day must stay finite, got %v", s.PerDay)
	}
}
