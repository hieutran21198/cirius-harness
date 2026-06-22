package repo

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"harness-workspace/services/harness/internal/domain"
)

// sessionRow maps the `sessions` table.
type sessionRow struct {
	ID        string     `gorm:"column:id;primaryKey"`
	ProjectID string     `gorm:"column:project_id"`
	EnvType   string     `gorm:"column:env_type"`
	EnvID     string     `gorm:"column:env_id"`
	Title     string     `gorm:"column:title"`
	Status    string     `gorm:"column:status"`
	CreatedAt time.Time  `gorm:"column:created_at"`
	StartedAt *time.Time `gorm:"column:started_at"`
	EndedAt   *time.Time `gorm:"column:ended_at"`
}

func (sessionRow) TableName() string { return "sessions" }

// sessionAgentRow maps the `session_agents` join. ModelID is a pointer so a model-less
// run is stored as NULL (the column is a nullable FK to models.id).
type sessionAgentRow struct {
	ID        string  `gorm:"column:id;primaryKey"`
	SessionID string  `gorm:"column:session_id"`
	AgentID   string  `gorm:"column:agent_id"`
	ModelID   *string `gorm:"column:model_id"`
}

func (sessionAgentRow) TableName() string { return "session_agents" }

// sessionWriter is a GORM-backed domain.SessionWriter bound to a db handle.
type sessionWriter struct {
	db *gorm.DB
}

// NewSessionWriter builds a domain.SessionWriter over db.
func NewSessionWriter(db *gorm.DB) domain.SessionWriter { return sessionWriter{db: db} }

// Save inserts the session row, idempotent on id (a re-sent hello is a no-op).
func (w sessionWriter) Save(ctx context.Context, s domain.Session) error {
	snap := s.Snapshot()
	row := sessionRow{
		ID:        string(snap.ID),
		ProjectID: string(snap.ProjectID),
		EnvType:   string(snap.EnvType),
		EnvID:     snap.EnvID,
		Title:     snap.Title,
		Status:    string(snap.Status),
		CreatedAt: snap.CreatedAt,
		StartedAt: snap.StartedAt,
		EndedAt:   snap.EndedAt,
	}
	err := w.db.WithContext(ctx).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "id"}}, DoNothing: true}).
		Create(&row).Error
	if err != nil {
		return fmt.Errorf("repo.sessionWriter.Save: %w", err)
	}
	return nil
}

// AddMember inserts one session_agents row, idempotent on (session_id, agent_id).
func (w sessionWriter) AddMember(ctx context.Context, sessionID domain.SessionID, m domain.Member) error {
	ms := m.Snapshot()
	var modelID *string
	if ms.ModelID != "" {
		s := string(ms.ModelID)
		modelID = &s
	}
	row := sessionAgentRow{
		ID:        string(ms.ID),
		SessionID: string(sessionID),
		AgentID:   string(ms.AgentID),
		ModelID:   modelID,
	}
	err := w.db.WithContext(ctx).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "session_id"}, {Name: "agent_id"}}, DoNothing: true}).
		Create(&row).Error
	if err != nil {
		return fmt.Errorf("repo.sessionWriter.AddMember: %w", err)
	}
	return nil
}

// staticcheck: ensure sessionWriter satisfies the domain port.
var _ domain.SessionWriter = sessionWriter{}
