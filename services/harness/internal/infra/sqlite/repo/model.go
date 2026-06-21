// Package repo holds the GORM repository implementations of the domain
// Reader/Writer interfaces, each bound to a *gorm.DB (the base connection or an
// open transaction). The unitofwork (writers) and readstore (readers) packages
// compose these into the app's UnitOfWork / ReadStore ports (ADR-0013).
package repo

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"harness-workspace/services/harness/internal/domain"
)

// modelRow maps the `models` table.
type modelRow struct {
	ID       string `gorm:"column:id;primaryKey"`
	Provider string `gorm:"column:provider"`
	Slug     string `gorm:"column:slug"`
	Enabled  bool   `gorm:"column:enabled"`
}

func (modelRow) TableName() string { return "models" }

// toRow maps a model's persistence view (its grouped Snapshot) to a table row —
// the repo never reaches into the aggregate's fields directly.
func toRow(snap domain.ModelSnapshot) modelRow {
	// The row's id column is a plain string; snap.ID is the typed domain.ModelID. This
	// is the one seam where the typed id is flattened for storage.
	return modelRow{ID: string(snap.ID), Provider: snap.Provider, Slug: snap.Slug, Enabled: snap.Enabled}
}

// modelWriter is a GORM-backed model.Writer bound to a db handle (the open
// transaction inside DoTx, or the base connection otherwise).
type modelWriter struct {
	db *gorm.DB
}

// NewModelWriter builds a domain.ModelWriter over db. Callers (the unitofwork) pass
// the base connection or an open transaction.
func NewModelWriter(db *gorm.DB) domain.ModelWriter { return modelWriter{db: db} }

// Existing returns which of the given refs are already in the catalog, as a set
// keyed by Ref. The lookup is a single (provider, slug) tuple IN query scoped to the
// refs, so it pulls only matching rows — its cost scales with the request size.
func (w modelWriter) Existing(ctx context.Context, refs []domain.Ref) (map[domain.Ref]struct{}, error) {
	out := make(map[domain.Ref]struct{}, len(refs))
	if len(refs) == 0 {
		return out, nil
	}
	tuples := make([][]any, len(refs))
	for i, r := range refs {
		tuples[i] = []any{r.Provider, r.Slug}
	}
	var rows []modelRow
	err := w.db.WithContext(ctx).Model(&modelRow{}).
		Select("provider", "slug").
		Where("(provider, slug) IN ?", tuples).
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("repo.modelWriter.Existing: %w", err)
	}
	for _, r := range rows {
		out[domain.Ref{Provider: r.Provider, Slug: r.Slug}] = struct{}{}
	}
	return out, nil
}

// SaveAll upserts the models on their natural key (provider, slug) in one batch: an
// existing row keeps its id and only `enabled` is updated; a new (provider, slug)
// inserts with its id.
func (w modelWriter) SaveAll(ctx context.Context, ms []domain.Model) error {
	if len(ms) == 0 {
		return nil
	}
	rows := make([]modelRow, 0, len(ms))
	for _, m := range ms {
		// New/Rehydrate already validated the aggregate; map its persistence view.
		rows = append(rows, toRow(m.Snapshot()))
	}
	err := w.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "provider"}, {Name: "slug"}},
		DoUpdates: clause.AssignmentColumns([]string{"enabled"}),
	}).Create(&rows).Error
	if err != nil {
		return fmt.Errorf("repo.modelWriter.SaveAll: %w", err)
	}
	return nil
}

// Count returns the number of models in the catalog.
func (w modelWriter) Count(ctx context.Context) (int, error) {
	var n int64
	if err := w.db.WithContext(ctx).Model(&modelRow{}).Count(&n).Error; err != nil {
		return 0, fmt.Errorf("repo.modelWriter.Count: %w", err)
	}
	return int(n), nil
}

// staticcheck: ensure modelWriter satisfies the domain port.
var _ domain.ModelWriter = modelWriter{}
