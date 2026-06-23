package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"harness-workspace/services/harness/internal/domain"
)

// taskReportRow maps the `task_reports` table, keyed by (plan_run_id, task_ref). envelope holds the
// validated TaskReportEnvelope as JSON; raw is the worker's full output, kept for audit.
type taskReportRow struct {
	ID        string    `gorm:"column:id;primaryKey"`
	PlanRunID string    `gorm:"column:plan_run_id"`
	TaskRef   string    `gorm:"column:task_ref"`
	Agent     string    `gorm:"column:agent"`
	Status    string    `gorm:"column:status"`
	Envelope  string    `gorm:"column:envelope"`
	Raw       string    `gorm:"column:raw"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (taskReportRow) TableName() string { return "task_reports" }

// taskReportWriter is a GORM-backed domain.TaskReportWriter that UPSERTs a report on its
// (plan_run_id, task_ref): a retried task overwrites its earlier report in place.
type taskReportWriter struct {
	db *gorm.DB
}

// NewTaskReportWriter builds a domain.TaskReportWriter over db.
func NewTaskReportWriter(db *gorm.DB) domain.TaskReportWriter { return taskReportWriter{db: db} }

// Save upserts the report row on (plan_run_id, task_ref).
func (w taskReportWriter) Save(ctx context.Context, r domain.TaskReport) error {
	snap := r.Snapshot()
	envelope, err := toJSON(snap.Envelope)
	if err != nil {
		return fmt.Errorf("repo.taskReportWriter.Save: marshal envelope: %w", err)
	}
	row := taskReportRow{
		ID: string(snap.ID), PlanRunID: string(snap.PlanRunID), TaskRef: snap.TaskRef,
		Agent: snap.Agent, Status: snap.Envelope.Status, Envelope: envelope, Raw: snap.Raw,
		CreatedAt: snap.CreatedAt, UpdatedAt: snap.UpdatedAt,
	}
	if err := w.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "plan_run_id"}, {Name: "task_ref"}},
		DoUpdates: clause.AssignmentColumns([]string{"agent", "status", "envelope", "raw", "updated_at"}),
	}).Create(&row).Error; err != nil {
		return fmt.Errorf("repo.taskReportWriter.Save: %w", err)
	}
	return nil
}

// taskReportReader is a GORM-backed domain.TaskReportReader.
type taskReportReader struct {
	db *gorm.DB
}

// NewTaskReportReader builds a domain.TaskReportReader over db.
func NewTaskReportReader(db *gorm.DB) domain.TaskReportReader { return taskReportReader{db: db} }

// FindByPlanRun loads every report for a run, ordered by task ref.
func (r taskReportReader) FindByPlanRun(ctx context.Context, planRunID domain.PlanRunID) ([]domain.TaskReport, error) {
	var rows []taskReportRow
	if err := r.db.WithContext(ctx).
		Where("plan_run_id = ?", string(planRunID)).
		Order("task_ref").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("repo.taskReportReader.FindByPlanRun: %w", err)
	}
	reports := make([]domain.TaskReport, 0, len(rows))
	for _, row := range rows {
		var env domain.TaskReportEnvelope
		if err := json.Unmarshal([]byte(row.Envelope), &env); err != nil {
			return nil, fmt.Errorf("repo.taskReportReader.FindByPlanRun: unmarshal envelope %q: %w", row.ID, err)
		}
		report, err := domain.RehydrateTaskReport(domain.TaskReportSnapshot{
			ID: domain.TaskReportID(row.ID), PlanRunID: domain.PlanRunID(row.PlanRunID), TaskRef: row.TaskRef,
			Agent: row.Agent, Envelope: env, Raw: row.Raw, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		})
		if err != nil {
			return nil, fmt.Errorf("repo.taskReportReader.FindByPlanRun: rehydrate %q: %w", row.ID, err)
		}
		reports = append(reports, report)
	}
	return reports, nil
}

// staticcheck: ensure the repo types satisfy the domain ports.
var (
	_ domain.TaskReportWriter = taskReportWriter{}
	_ domain.TaskReportReader = taskReportReader{}
)
