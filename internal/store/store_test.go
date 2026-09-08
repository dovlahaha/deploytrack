package store

import (
	"os"
	"testing"
	"time"
)

// openTestStore connects to the database named by TEST_DATABASE_URL.
//
// The test skips rather than fails when that variable is absent, so a plain
// `go test ./...` works on a laptop with no database running. CI sets the
// variable, so the same test is a real integration test there.
func openTestStore(t *testing.T) *Store {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping integration test")
	}

	s, err := Open(dsn)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	if err := s.Reset(); err != nil {
		t.Fatalf("reset: %v", err)
	}
	return s
}

func TestCreateAndList(t *testing.T) {
	s := openTestStore(t)

	d := &Deployment{
		ServiceName: "checkout",
		Version:     "abc123",
		Environment: "production",
		Status:      "success",
	}
	if err := s.Create(d); err != nil {
		t.Fatalf("create: %v", err)
	}
	if d.ID == 0 {
		t.Error("expected an ID to be assigned")
	}
	if d.DeployedAt.IsZero() {
		t.Error("expected deployed_at to default to now")
	}

	got, err := s.List(ListFilter{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 deployment, got %d", len(got))
	}
	if got[0].ServiceName != "checkout" {
		t.Errorf("service_name: got %q, want %q", got[0].ServiceName, "checkout")
	}
}

func TestListFiltersByServiceAndTime(t *testing.T) {
	s := openTestStore(t)

	old := time.Now().UTC().Add(-30 * 24 * time.Hour)
	seed := []*Deployment{
		{ServiceName: "checkout", Version: "a", Environment: "production", Status: "success"},
		{ServiceName: "checkout", Version: "b", Environment: "staging", Status: "failed"},
		{ServiceName: "billing", Version: "c", Environment: "production", Status: "success"},
		{ServiceName: "checkout", Version: "d", Environment: "production", Status: "success", DeployedAt: old},
	}
	for _, d := range seed {
		if err := s.Create(d); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	byService, err := s.List(ListFilter{ServiceName: "checkout"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(byService) != 3 {
		t.Errorf("service filter: got %d, want 3", len(byService))
	}

	recent, err := s.List(ListFilter{
		ServiceName: "checkout",
		Since:       time.Now().UTC().Add(-7 * 24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(recent) != 2 {
		t.Errorf("time filter: got %d, want 2 (the 30-day-old one is excluded)", len(recent))
	}
}

func TestEventsConversion(t *testing.T) {
	now := time.Now()
	ds := []Deployment{{Status: "success", DeployedAt: now}, {Status: "failed", DeployedAt: now}}

	events := Events(ds)
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}
	if events[1].Status != "failed" {
		t.Errorf("status: got %q, want %q", events[1].Status, "failed")
	}
}
