package repo_test

import (
	"context"
	"path/filepath"
	"testing"

	"gorm.io/gorm"

	"harness-workspace/packages/go/gormdb"
	"harness-workspace/packages/go/gormdb/sqlite"
	"harness-workspace/services/harness/internal/domain"
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
		client TEXT NOT NULL,
		provider TEXT NOT NULL,
		slug TEXT NOT NULL,
		enabled INTEGER NOT NULL DEFAULT 0,
		UNIQUE(client, provider, slug)
	)`).Error
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	return db
}

func TestModelWriterSaveAllExistingCount(t *testing.T) {
	ctx := context.Background()
	w := repo.NewModelWriter(newDB(t))

	m1, err := domain.NewModel(domain.ClientPi, "openai", "gpt-5.5")
	if err != nil {
		t.Fatalf("domain.NewModel: %v", err)
	}
	if err = w.SaveAll(ctx, []domain.Model{m1}); err != nil {
		t.Fatalf("SaveAll: %v", err)
	}

	gpt := domain.Ref{Client: domain.ClientPi, Provider: "openai", Slug: "gpt-5.5"}
	absent := domain.Ref{Client: domain.ClientPi, Provider: "anthropic", Slug: "claude-opus-4-8"}
	// Targeted lookup: returns only the queried refs that exist (the present one,
	// not the absent one).
	existing, err := w.Existing(ctx, []domain.Ref{gpt, absent})
	if err != nil {
		t.Fatalf("Existing: %v", err)
	}
	if _, ok := existing[gpt]; !ok {
		t.Fatalf("Existing = %v; want it to contain %s", existing, gpt)
	}
	if _, ok := existing[absent]; ok {
		t.Fatalf("Existing = %v; should not contain the absent ref %s", existing, absent)
	}

	// Re-save the same (client, provider, slug) → upsert on the natural key, not a new
	// row (NewModel mints a fresh id, so this is a distinct aggregate with the same key).
	m2, err := domain.NewModel(domain.ClientPi, "openai", "gpt-5.5")
	if err != nil {
		t.Fatalf("domain.NewModel: %v", err)
	}
	if err = w.SaveAll(ctx, []domain.Model{m2}); err != nil {
		t.Fatalf("re-SaveAll: %v", err)
	}
	n, err := w.Count(ctx)
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if n != 1 {
		t.Fatalf("Count = %d, want 1 (upsert on natural key)", n)
	}

	// Same (provider, slug) under a DIFFERENT client is a distinct row — client is part
	// of the natural key (ADR-0015). It must not upsert onto the pi row.
	oc, err := domain.NewModel(domain.ClientOpencode, "openai", "gpt-5.5")
	if err != nil {
		t.Fatalf("domain.NewModel: %v", err)
	}
	if err = w.SaveAll(ctx, []domain.Model{oc}); err != nil {
		t.Fatalf("SaveAll opencode: %v", err)
	}
	n, err = w.Count(ctx)
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if n != 2 {
		t.Fatalf("Count = %d, want 2 (pi and opencode are distinct catalog entries)", n)
	}
}
