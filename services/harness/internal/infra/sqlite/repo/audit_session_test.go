package repo_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"gorm.io/gorm"

	"harness-workspace/packages/go/gormdb"
	"harness-workspace/packages/go/gormdb/sqlite"
	"harness-workspace/packages/go/migrate"
	"harness-workspace/services/harness/internal/domain"
	"harness-workspace/services/harness/internal/infra/sqlite/repo"
	"harness-workspace/services/harness/migrations"
)

// newMigratedDB builds a temp SQLite with the full schema applied (so FKs and the
// seeded agents exist), for the writers that span real tables.
func newMigratedDB(t *testing.T) *gorm.DB {
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
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sql handle: %v", err)
	}
	m, err := migrate.New(sqlDB, migrations.FS, migrate.DialectSQLite)
	if err != nil {
		t.Fatalf("migrate.New: %v", err)
	}
	if err := m.Up(ctx); err != nil {
		t.Fatalf("migrate up: %v", err)
	}
	return db
}

func TestEventWriterAppend(t *testing.T) {
	ctx := context.Background()
	db := newMigratedDB(t)
	w := repo.NewEventWriter(db)

	ev, err := domain.NewEvent("SyncModels", "pi", domain.EventOK, "", "", time.Now())
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}
	if err := w.Append(ctx, ev); err != nil {
		t.Fatalf("Append: %v", err)
	}
	var n int64
	if err := db.WithContext(ctx).Table("events").Count(&n).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Fatalf("events count = %d, want 1", n)
	}
}

func TestProjectWriterEnsureByRootIsIdempotent(t *testing.T) {
	ctx := context.Background()
	w := repo.NewProjectWriter(newMigratedDB(t))

	id1, err := w.EnsureByRoot(ctx, "/proj/demo", "demo")
	if err != nil {
		t.Fatalf("EnsureByRoot: %v", err)
	}
	id2, err := w.EnsureByRoot(ctx, "/proj/demo", "demo")
	if err != nil {
		t.Fatalf("EnsureByRoot again: %v", err)
	}
	if id1 == "" || id1 != id2 {
		t.Fatalf("EnsureByRoot ids = %q, %q; want equal non-empty (idempotent on root)", id1, id2)
	}
}

func TestSessionWriterSaveAndAddMember(t *testing.T) {
	ctx := context.Background()
	db := newMigratedDB(t)
	projects := repo.NewProjectWriter(db)
	sessions := repo.NewSessionWriter(db)

	projectID, err := projects.EnsureByRoot(ctx, "/proj/demo", "demo")
	if err != nil {
		t.Fatalf("EnsureByRoot: %v", err)
	}
	s, err := domain.RehydrateSession("sess-1", projectID, domain.EnvUnset, "", "", domain.SessionRunning, time.Now(), nil, nil, nil)
	if err != nil {
		t.Fatalf("RehydrateSession: %v", err)
	}
	if err = sessions.Save(ctx, s); err != nil {
		t.Fatalf("Save: %v", err)
	}
	// Re-save the same id is idempotent (a re-sent hello).
	if err = sessions.Save(ctx, s); err != nil {
		t.Fatalf("Save again: %v", err)
	}

	// A real agent id is needed for the FK; the seed migration created council.
	var agentID string
	if err = db.WithContext(ctx).Table("agents").Select("id").Where("name = ?", "council").Take(&agentID).Error; err != nil {
		t.Fatalf("lookup council: %v", err)
	}
	m, err := domain.NewMember(domain.AgentID(agentID), "") // model-less → stored NULL
	if err != nil {
		t.Fatalf("NewMember: %v", err)
	}
	if err = sessions.AddMember(ctx, "sess-1", m); err != nil {
		t.Fatalf("AddMember: %v", err)
	}
	// Recording the same agent again is idempotent on (session, agent).
	if err = sessions.AddMember(ctx, "sess-1", m); err != nil {
		t.Fatalf("AddMember again: %v", err)
	}

	var members int64
	if err = db.WithContext(ctx).Table("session_agents").Where("session_id = ?", "sess-1").Count(&members).Error; err != nil {
		t.Fatalf("count members: %v", err)
	}
	if members != 1 {
		t.Fatalf("session_agents count = %d, want 1 (idempotent)", members)
	}
}
