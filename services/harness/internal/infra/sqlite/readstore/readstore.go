// Package readstore is the GORM-backed implementation of the app's query.ReadStore
// (ADR-0013): it composes the repo readers over the base connection. Reads need no
// transaction, so — unlike the unitofwork — each reader autocommits per call.
package readstore

import (
	"gorm.io/gorm"

	"harness-workspace/services/harness/internal/app/query"
	"harness-workspace/services/harness/internal/domain"
	"harness-workspace/services/harness/internal/infra/sqlite/repo"
)

// ReadStore is the query.ReadStore. Its readers run over the base connection.
type ReadStore struct {
	db *gorm.DB
}

// New builds a ReadStore over db.
func New(db *gorm.DB) *ReadStore { return &ReadStore{db: db} }

// Agents returns the agent reader bound to this store's handle.
func (s *ReadStore) Agents() domain.AgentReader { return repo.NewAgentReader(s.db) }

// staticcheck: ensure ReadStore satisfies the query port.
var _ query.ReadStore = (*ReadStore)(nil)
