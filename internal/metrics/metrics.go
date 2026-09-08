// Package metrics computes software-delivery metrics from deployment events.
//
// It deliberately has no database or framework dependencies: it operates on a
// plain slice of Events. That keeps it fast to unit test and makes the tests
// meaningful without any infrastructure.
package metrics

import "time"

// Deployment outcome statuses.
const (
	StatusSuccess    = "success"
	StatusFailed     = "failed"
	StatusRolledBack = "rolled_back"
)

// Event is the minimal shape the calculations need.
type Event struct {
	Status string
	At     time.Time
}

// Summary is the calculated result for a set of events over a window.
type Summary struct {
	WindowDays        float64 `json:"window_days"`
	TotalDeployments  int     `json:"total_deployments"`
	FailedDeployments int     `json:"failed_deployments"`
	PerDay            float64 `json:"deployments_per_day"`
	PerWeek           float64 `json:"deployments_per_week"`
	ChangeFailureRate float64 `json:"change_failure_rate"`
	Rating            string  `json:"deployment_frequency_rating"`
}

// Calculate derives delivery metrics for the given window.
//
// A deployment counts as a failure if it failed outright or had to be rolled
// back — matching the DORA definition, which counts changes requiring urgent
// remediation rather than every bug found later.
func Calculate(events []Event, window time.Duration) Summary {
	days := window.Hours() / 24
	if days <= 0 {
		days = 1
	}

	s := Summary{WindowDays: days}
	for _, e := range events {
		s.TotalDeployments++
		if e.Status == StatusFailed || e.Status == StatusRolledBack {
			s.FailedDeployments++
		}
	}

	if s.TotalDeployments > 0 {
		s.PerDay = float64(s.TotalDeployments) / days
		s.PerWeek = s.PerDay * 7
		s.ChangeFailureRate = float64(s.FailedDeployments) / float64(s.TotalDeployments)
	}
	s.Rating = rate(s.PerDay)
	return s
}

// rate buckets deployment frequency using the DORA performance bands.
func rate(perDay float64) string {
	switch {
	case perDay >= 1:
		return "elite"
	case perDay >= 1.0/7.0:
		return "high"
	case perDay >= 1.0/30.0:
		return "medium"
	default:
		return "low"
	}
}
