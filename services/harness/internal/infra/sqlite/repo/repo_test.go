package repo_test

import (
	"context"
	"path/filepath"
	"testing"

	"gorm.io/gorm"

	"harness-workspace/packages/go/gormdb"
	"harness-workspace/packages/go/gormdb/sqlite"
	"harness-workspace/services/harness/internal/domain/model"
	"harness-workspace/services/harness/internal/infra/sqlite/repo"
)

// newDB builds a fresh temp-file SQLite with the models table.
func newDB(t *testing.T) *gorm.DB {
	t.Helper()
	ctx := context.Background()
	dialect, err := sqlite.New(filepath.Join(t.TempDir(), "test.sqlite"))
	if err != nil {
		t.Fatalf("dialect: %v", err)
	}
	db, err := gormdb.New(ctx, dialect)
	if err != nil {
		t.Fatalf("gormdb.New: %v", err)
	}
	err = db.WithContext(ctx).Exec(`CREATE TABLE models (
		id TEXT PRIMARY KEY,
		provider TEXT NOT NULL,
		slug TEXT NOT NULL,
		enabled INTEGER NOT NULL DEFAULT 0,
		UNIQUE(provider, slug)
	)`).Error
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	return db
}

func TestModelWriterSaveAllExistingKeysCount(t *testing.T) {
	ctx := context.Background()
	w := repo.NewModelWriter(newDB(t))

	m1, err := model.New("1", "openai", "gpt-5.5")
	if err != nil {
		t.Fatalf("model.New: %v", err)
	}
	if err := w.SaveAll(ctx, []model.Model{m1}); err != nil {
		t.Fatalf("SaveAll: %v", err)
	}
	keys, err := w.ExistingKeys(ctx)
	if err != nil {
		t.Fatalf("ExistingKeys: %v", err)
	}
	if _, ok := keys["openai/gpt-5.5"]; !ok {
		t.Fatalf("ExistingKeys = %v; want it to contain openai/gpt-5.5", keys)
	}
	if _, ok := keys["anthropic/claude-opus-4-8"]; ok {
		t.Fatalf("ExistingKeys = %v; should not contain an unsaved ref", keys)
	}

	// Re-save the same (provider, slug) with a different id → upsert, not a new row.
	m2, err := model.New("2", "openai", "gpt-5.5")
	if err != nil {
		t.Fatalf("model.New: %v", err)
	}
	if err := w.SaveAll(ctx, []model.Model{m2}); err != nil {
		t.Fatalf("re-SaveAll: %v", err)
	}
	n, err := w.Count(ctx)
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if n != 1 {
		t.Fatalf("Count = %d, want 1 (upsert on natural key)", n)
	}
}
