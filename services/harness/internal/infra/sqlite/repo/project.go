package repo

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"harness-workspace/services/harness/internal/domain"
)

// projectRow maps the `projects` table.
type projectRow struct {
	ID          string `gorm:"column:id;primaryKey"`
	Name        string `gorm:"column:name"`
	RootPath    string `gorm:"column:root_path"`
	Kind        string `gorm:"column:kind"`
	Description string `gorm:"column:description"`
}

func (projectRow) TableName() string { return "projects" }

// projectWriter is a GORM-backed domain.ProjectWriter bound to a db handle.
type projectWriter struct {
	db *gorm.DB
}

// NewProjectWriter builds a domain.ProjectWriter over db.
func NewProjectWriter(db *gorm.DB) domain.ProjectWriter { return projectWriter{db: db} }

// EnsureByRoot returns the id of the project at rootPath, creating it (a single-kind
// project named name) if absent. rootPath is the unique business key.
func (w projectWriter) EnsureByRoot(ctx context.Context, rootPath, name string) (domain.ProjectID, error) {
	var row projectRow
	err := w.db.WithContext(ctx).Model(&projectRow{}).Where("root_path = ?", rootPath).Take(&row).Error
	if err == nil {
		return domain.ProjectID(row.ID), nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", fmt.Errorf("repo.projectWriter.EnsureByRoot lookup: %w", err)
	}

	p, err := domain.NewProject(name, rootPath, domain.KindSingle, "")
	if err != nil {
		return "", err
	}
	s := p.Snapshot()
	newRow := projectRow{ID: string(s.ID), Name: s.Name, RootPath: s.RootPath, Kind: string(s.Kind), Description: s.Description}
	if err := w.db.WithContext(ctx).Create(&newRow).Error; err != nil {
		return "", fmt.Errorf("repo.projectWriter.EnsureByRoot create: %w", err)
	}
	return s.ID, nil
}

// staticcheck: ensure projectWriter satisfies the domain port.
var _ domain.ProjectWriter = projectWriter{}
