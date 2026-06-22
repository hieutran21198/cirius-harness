package repo

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"harness-workspace/services/harness/internal/domain"
)

// eventRow maps the `events` table (append-only audit log).
type eventRow struct {
	ID         string    `gorm:"column:id;primaryKey"`
	OccurredAt time.Time `gorm:"column:occurred_at"`
	Kind       string    `gorm:"column:kind"`
	Actor      string    `gorm:"column:actor"`
	Status     string    `gorm:"column:status"`
	Message    string    `gorm:"column:message"`
	Detail     string    `gorm:"column:detail"`
}

func (eventRow) TableName() string { return "events" }

// eventWriter is a GORM-backed domain.EventWriter bound to a db handle.
type eventWriter struct {
	db *gorm.DB
}

// NewEventWriter builds a domain.EventWriter over db.
func NewEventWriter(db *gorm.DB) domain.EventWriter { return eventWriter{db: db} }

// Append inserts one audit event.
func (w eventWriter) Append(ctx context.Context, e domain.Event) error {
	s := e.Snapshot()
	row := eventRow{
		ID:         string(s.ID),
		OccurredAt: s.OccurredAt,
		Kind:       s.Kind,
		Actor:      s.Actor,
		Status:     s.Status,
		Message:    s.Message,
		Detail:     s.Detail,
	}
	if err := w.db.WithContext(ctx).Create(&row).Error; err != nil {
		return fmt.Errorf("repo.eventWriter.Append: %w", err)
	}
	return nil
}

// staticcheck: ensure eventWriter satisfies the domain port.
var _ domain.EventWriter = eventWriter{}
