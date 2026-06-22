package repo

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"harness-workspace/services/harness/internal/domain"
)

// agentRow maps the `agents` table. ToolIDs live in the agent_tools junction and are
// not hydrated here — the read side that needs them composes a separate lookup; the
// agent-resolve query only needs the role's identity (its persona is a harness-owned
// domain constant, not a column — ADR-0016).
type agentRow struct {
	ID             string `gorm:"column:id;primaryKey"`
	Name           string `gorm:"column:name"`
	Archetype      string `gorm:"column:archetype"`
	Responsibility string `gorm:"column:responsibility"`
	Description    string `gorm:"column:description"`
	Source         string `gorm:"column:source"`
	Enabled        bool   `gorm:"column:enabled"`
}

func (agentRow) TableName() string { return "agents" }

// agentReader is a GORM-backed domain.AgentReader bound to a db handle.
type agentReader struct {
	db *gorm.DB
}

// NewAgentReader builds a domain.AgentReader over db.
func NewAgentReader(db *gorm.DB) domain.AgentReader { return agentReader{db: db} }

// FindByName returns the agent with the given name, or domain.ErrAgentNotFound.
func (r agentReader) FindByName(ctx context.Context, name string) (domain.Agent, error) {
	var row agentRow
	err := r.db.WithContext(ctx).Model(&agentRow{}).Where("name = ?", name).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.Agent{}, fmt.Errorf("%w: %q", domain.ErrAgentNotFound, name)
	}
	if err != nil {
		return domain.Agent{}, fmt.Errorf("repo.agentReader.FindByName: %w", err)
	}
	// ToolIDs are not part of this read; pass nil — the role's grants are looked up
	// separately when a use case needs them.
	return domain.RehydrateAgent(
		domain.AgentID(row.ID),
		row.Name,
		domain.Archetype(row.Archetype),
		row.Responsibility,
		row.Description,
		domain.Source(row.Source),
		row.Enabled,
		nil,
	)
}

// staticcheck: ensure agentReader satisfies the domain port.
var _ domain.AgentReader = agentReader{}
