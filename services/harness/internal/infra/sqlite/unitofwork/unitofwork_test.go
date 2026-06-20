package unitofwork_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"gorm.io/gorm"

	"harness-workspace/packages/go/gormdb"
	"harness-workspace/packages/go/gormdb/sqlite"
	"harness-workspace/services/harness/internal/app/command"
	"harness-workspace/services/harness/internal/domain/model"
	"harness-workspace/services/harness/internal/infra/sqlite/unitofwork"
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

func TestDoTxCommits(t *testing.T) {
	ctx := context.Background()
	uow := unitofwork.New(newDB(t))

	err := uow.DoTx(ctx, func(ctx context.Context, tx command.TransactionalUnitOfWork) error {
		return tx.Models().Save(ctx, model.Model{ID: "1", Provider: "openai", Slug: "gpt-5.5", Enabled: true})
	})
	if err != nil {
		t.Fatalf("DoTx: %v", err)
	}
	if n, _ := uow.Models().Count(ctx); n != 1 {
		t.Fatalf("after commit Count = %d, want 1", n)
	}
}

func TestDoTxRollsBackOnError(t *testing.T) {
	ctx := context.Background()
	uow := unitofwork.New(newDB(t))
	boom := errors.New("boom")

	err := uow.DoTx(ctx, func(ctx context.Context, tx command.TransactionalUnitOfWork) error {
		if err := tx.Models().Save(ctx, model.Model{ID: "1", Provider: "openai", Slug: "gpt-5.5", Enabled: true}); err != nil {
			return err
		}
		return boom // abort → the save above must roll back
	})
	if !errors.Is(err, boom) {
		t.Fatalf("DoTx err = %v, want %v", err, boom)
	}
	if n, _ := uow.Models().Count(ctx); n != 0 {
		t.Fatalf("after rollback Count = %d, want 0", n)
	}
}
