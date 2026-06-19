// Package casbinx bootstraps a Casbin enforcer whose policy is persisted in a
// shared GORM database via the official gorm-adapter. It is provider-agnostic:
// the model definition and decision semantics belong to the caller.
package casbinx

import (
	"fmt"

	"github.com/casbin/casbin/v3"
	"github.com/casbin/casbin/v3/model"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"gorm.io/gorm"
)

// NewEnforcer builds a Casbin enforcer that stores its policy in db (the
// gorm-adapter creates and manages the casbin_rule table) using the model
// described by modelText. The policy is loaded before the enforcer is returned.
func NewEnforcer(db *gorm.DB, modelText string) (*casbin.Enforcer, error) {
	adapter, err := gormadapter.NewAdapterByDB(db)
	if err != nil {
		return nil, fmt.Errorf("casbinx.NewEnforcer: adapter: %w", err)
	}

	m, err := model.NewModelFromString(modelText)
	if err != nil {
		return nil, fmt.Errorf("casbinx.NewEnforcer: model: %w", err)
	}

	enforcer, err := casbin.NewEnforcer(m, adapter)
	if err != nil {
		return nil, fmt.Errorf("casbinx.NewEnforcer: enforcer: %w", err)
	}

	if err := enforcer.LoadPolicy(); err != nil {
		return nil, fmt.Errorf("casbinx.NewEnforcer: load policy: %w", err)
	}
	return enforcer, nil
}
