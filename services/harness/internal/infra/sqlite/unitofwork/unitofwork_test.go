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
	"harness-workspace/services/harness/internal/domain"
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

func TestDoTxCommits(t *testing.T) {
	ctx := context.Background()
	uow := unitofwork.New(newDB(t))

	m1, err := domain.NewModel(domain.ClientPi, "openai", "gpt-5.5")
	if err != nil {
		t.Fatalf("domain.NewModel: %v", err)
	}
	err = uow.DoTx(ctx, func(ctx context.Context, tx command.TransactionalUnitOfWork) error {
		return tx.Models().SaveAll(ctx, []domain.Model{m1})
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

	m1, err := domain.NewModel(domain.ClientPi, "openai", "gpt-5.5")
	if err != nil {
		t.Fatalf("domain.NewModel: %v", err)
	}
	err = uow.DoTx(ctx, func(ctx context.Context, tx command.TransactionalUnitOfWork) error {
		if saveErr := tx.Models().SaveAll(ctx, []domain.Model{m1}); saveErr != nil {
			return saveErr
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
