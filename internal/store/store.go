// Package store owns persistence for deployment records.
package store

import (
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/dovlahaha/deploytrack/internal/metrics"
)

// Deployment is one recorded deployment of a service.
type Deployment struct {
	ID          uint      `gorm:"primaryKey"                 json:"id"`
	ServiceName string    `gorm:"index;not null"             json:"service_name"`
	Version     string    `gorm:"not null"                   json:"version"`
	Environment string    `gorm:"index;not null"             json:"environment"`
	Status      string    `gorm:"not null"                   json:"status"`
	DeployedAt  time.Time `gorm:"index;not null"             json:"deployed_at"`
}

// Store wraps the database handle.
type Store struct{ db *gorm.DB }

// Open connects to Postgres and applies the schema.
func Open(dsn string) (*Store, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	if err := db.AutoMigrate(&Deployment{}); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return &Store{db: db}, nil
}

// Create records a deployment, defaulting the timestamp to now.
func (s *Store) Create(d *Deployment) error {
	if d.DeployedAt.IsZero() {
		d.DeployedAt = time.Now().UTC()
	}
	return s.db.Create(d).Error
}

// ListFilter narrows a query. Zero values mean "no filter".
type ListFilter struct {
	ServiceName string
	Environment string
	Since       time.Time
}

// List returns matching deployments, newest first.
func (s *Store) List(f ListFilter) ([]Deployment, error) {
	q := s.db.Model(&Deployment{})
	if f.ServiceName != "" {
		q = q.Where("service_name = ?", f.ServiceName)
	}
	if f.Environment != "" {
		q = q.Where("environment = ?", f.Environment)
	}
	if !f.Since.IsZero() {
		q = q.Where("deployed_at >= ?", f.Since)
	}

	var out []Deployment
	if err := q.Order("deployed_at desc").Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

// Events adapts deployments into the shape the metrics package expects.
// Keeping the conversion here means metrics stays free of storage concerns.
func Events(ds []Deployment) []metrics.Event {
	out := make([]metrics.Event, 0, len(ds))
	for _, d := range ds {
		out = append(out, metrics.Event{Status: d.Status, At: d.DeployedAt})
	}
	return out
}

// Reset clears the table. Used by tests only.
func (s *Store) Reset() error {
	return s.db.Exec("TRUNCATE TABLE deployments RESTART IDENTITY").Error
}
