package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"harness-workspace/services/harness/internal/domain"
)

// planRunRow maps the `plan_runs` table. SessionID is a pointer so a session-less run is NULL.
type planRunRow struct {
	ID        string    `gorm:"column:id;primaryKey"`
	PlanID    string    `gorm:"column:plan_id"`
	SessionID *string   `gorm:"column:session_id"`
	Status    string    `gorm:"column:status"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (planRunRow) TableName() string { return "plan_runs" }

// planTaskRunRow maps the `plan_task_runs` table, keyed by (plan_run_id, task_ref).
type planTaskRunRow struct {
	ID        string    `gorm:"column:id;primaryKey"`
	PlanRunID string    `gorm:"column:plan_run_id"`
	TaskRef   string    `gorm:"column:task_ref"`
	Status    string    `gorm:"column:status"`
	Summary   string    `gorm:"column:summary"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (planTaskRunRow) TableName() string { return "plan_task_runs" }

// planRunWriter is a GORM-backed domain.PlanRunWriter that UPSERTs run + task-run rows.
type planRunWriter struct {
	db *gorm.DB
}

// NewPlanRunWriter builds a domain.PlanRunWriter over db.
func NewPlanRunWriter(db *gorm.DB) domain.PlanRunWriter { return planRunWriter{db: db} }

// Save upserts the run row (on its id) and every task-run row (on plan_run_id+task_ref). A drive
// reports progress repeatedly, so this overwrites in place rather than inserting once.
func (w planRunWriter) Save(ctx context.Context, r domain.PlanRun) error {
	snap := r.Snapshot()
	db := w.db.WithContext(ctx)

	var sessionID *string
	if snap.SessionID != "" {
		s := string(snap.SessionID)
		sessionID = &s
	}
	row := planRunRow{
		ID: string(snap.ID), PlanID: string(snap.PlanID), SessionID: sessionID,
		Status: string(snap.Status), CreatedAt: snap.CreatedAt, UpdatedAt: snap.UpdatedAt,
	}
	if err := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"status", "updated_at"}),
	}).Create(&row).Error; err != nil {
		return fmt.Errorf("repo.planRunWriter.Save: %w", err)
	}

	if len(snap.Tasks) == 0 {
		return nil
	}
	rows := make([]planTaskRunRow, len(snap.Tasks))
	for i, t := range snap.Tasks {
		rows[i] = planTaskRunRow{
			ID: string(t.ID), PlanRunID: string(snap.ID), TaskRef: t.TaskRef,
			Status: string(t.Status), Summary: t.Summary, UpdatedAt: t.UpdatedAt,
		}
	}
	if err := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "plan_run_id"}, {Name: "task_ref"}},
		DoUpdates: clause.AssignmentColumns([]string{"status", "summary", "updated_at"}),
	}).Create(&rows).Error; err != nil {
		return fmt.Errorf("repo.planRunWriter.Save: tasks: %w", err)
	}
	return nil
}

// planRunReader is a GORM-backed domain.PlanRunReader.
type planRunReader struct {
	db *gorm.DB
}

// NewPlanRunReader builds a domain.PlanRunReader over db.
func NewPlanRunReader(db *gorm.DB) domain.PlanRunReader { return planRunReader{db: db} }

// FindByPlan loads the run for a plan, or ErrPlanRunNotFound.
func (r planRunReader) FindByPlan(ctx context.Context, planID domain.PlanID) (domain.PlanRun, error) {
	db := r.db.WithContext(ctx)
	var run planRunRow
	err := db.Where("plan_id = ?", string(planID)).Take(&run).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.PlanRun{}, domain.ErrPlanRunNotFound
	}
	if err != nil {
		return domain.PlanRun{}, fmt.Errorf("repo.planRunReader.FindByPlan: %w", err)
	}
	var taskRows []planTaskRunRow
	if err := db.Where("plan_run_id = ?", run.ID).Find(&taskRows).Error; err != nil {
		return domain.PlanRun{}, fmt.Errorf("repo.planRunReader.FindByPlan: tasks: %w", err)
	}
	tasks := make([]domain.TaskRunSnapshot, len(taskRows))
	for i, t := range taskRows {
		tasks[i] = domain.TaskRunSnapshot{
			ID: domain.TaskRunID(t.ID), TaskRef: t.TaskRef, Status: domain.TaskStatus(t.Status),
			Summary: t.Summary, UpdatedAt: t.UpdatedAt,
		}
	}
	var sessionID domain.SessionID
	if run.SessionID != nil {
		sessionID = domain.SessionID(*run.SessionID)
	}
	snap := domain.PlanRunSnapshot{
		ID: domain.PlanRunID(run.ID), PlanID: domain.PlanID(run.PlanID), SessionID: sessionID,
		Status: domain.PlanStatus(run.Status), CreatedAt: run.CreatedAt, UpdatedAt: run.UpdatedAt, Tasks: tasks,
	}
	return domain.RehydratePlanRun(snap)
}

// staticcheck: ensure the repo types satisfy the domain ports.
var (
	_ domain.PlanRunWriter = planRunWriter{}
	_ domain.PlanRunReader = planRunReader{}
)
