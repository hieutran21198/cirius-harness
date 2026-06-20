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

	"harness-workspace/services/harness/internal/domain/model"
)

// modelRow maps the `models` table.
type modelRow struct {
	ID       string `gorm:"column:id;primaryKey"`
	Provider string `gorm:"column:provider"`
	Slug     string `gorm:"column:slug"`
	Enabled  bool   `gorm:"column:enabled"`
}

func (modelRow) TableName() string { return "models" }

func toRow(m model.Model) modelRow {
	return modelRow{ID: m.ID, Provider: m.Provider, Slug: m.Slug, Enabled: m.Enabled}
}

// modelWriter is a GORM-backed model.Writer bound to a db handle (the open
// transaction inside DoTx, or the base connection otherwise).
type modelWriter struct {
	db *gorm.DB
}

// NewModelWriter builds a model.Writer over db. Callers (the unitofwork) pass the
// base connection or an open transaction.
func NewModelWriter(db *gorm.DB) model.Writer { return modelWriter{db: db} }

// ExistingKeys returns the natural keys ("provider/slug") present in the catalog as a
// set, for an in-memory membership check before a batch upsert.
func (w modelWriter) ExistingKeys(ctx context.Context) (map[string]struct{}, error) {
	var rows []modelRow
	err := w.db.WithContext(ctx).Model(&modelRow{}).
		Select("provider", "slug").Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("repo.modelWriter.ExistingKeys: %w", err)
	}
	keys := make(map[string]struct{}, len(rows))
	for _, r := range rows {
		keys[r.Provider+"/"+r.Slug] = struct{}{}
	}
	return keys, nil
}

// SaveAll upserts the models on their natural key (provider, slug) in one batch: an
// existing row keeps its id and only `enabled` is updated; a new (provider, slug)
// inserts with its id.
func (w modelWriter) SaveAll(ctx context.Context, ms []model.Model) error {
	if len(ms) == 0 {
		return nil
	}
	rows := make([]modelRow, 0, len(ms))
	for _, m := range ms {
		if err := m.Validate(); err != nil {
			return err
		}
		rows = append(rows, toRow(m))
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
var _ model.Writer = modelWriter{}
