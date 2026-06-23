package repo

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"harness-workspace/services/harness/internal/domain"
)

// planDecisionRow maps the `council_decisions` table. decision holds the validated CouncilDecision
// as JSON. Append-only: one row per recorded decision.
type planDecisionRow struct {
	ID        string    `gorm:"column:id;primaryKey"`
	PlanRunID string    `gorm:"column:plan_run_id"`
	Decision  string    `gorm:"column:decision"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (planDecisionRow) TableName() string { return "council_decisions" }

// planDecisionWriter is a GORM-backed domain.PlanDecisionWriter that inserts a decision row.
type planDecisionWriter struct {
	db *gorm.DB
}

// NewPlanDecisionWriter builds a domain.PlanDecisionWriter over db.
func NewPlanDecisionWriter(db *gorm.DB) domain.PlanDecisionWriter { return planDecisionWriter{db: db} }

// Save inserts the decision row (append-only).
func (w planDecisionWriter) Save(ctx context.Context, d domain.PlanDecision) error {
	snap := d.Snapshot()
	decision, err := toJSON(snap.Decision)
	if err != nil {
		return fmt.Errorf("repo.planDecisionWriter.Save: marshal decision: %w", err)
	}
	row := planDecisionRow{
		ID: string(snap.ID), PlanRunID: string(snap.PlanRunID), Decision: decision, CreatedAt: snap.CreatedAt,
	}
	if err := w.db.WithContext(ctx).Create(&row).Error; err != nil {
		return fmt.Errorf("repo.planDecisionWriter.Save: %w", err)
	}
	return nil
}

// staticcheck: ensure the repo type satisfies the domain port.
var _ domain.PlanDecisionWriter = planDecisionWriter{}
