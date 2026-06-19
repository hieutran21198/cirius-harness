// Package casbinauthz is the Casbin-backed implementation of the harness
// authorization port (outbound.Authorizer). Its policy is persisted in the
// shared database's casbin_rule table.
package casbinauthz

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/casbin/casbin/v3"
	"gorm.io/gorm"

	"harness-workspace/packages/go/casbinx"
	"harness-workspace/services/harness/internal/domain/authz"
	"harness-workspace/services/harness/internal/port/outbound"
)

//go:embed model.conf
var modelConf string

// Enforcer is a Casbin-backed outbound.Authorizer.
type Enforcer struct {
	enforcer *casbin.Enforcer
}

// New builds an Enforcer over db, loading the embedded ABAC model and the
// persisted policy.
func New(db *gorm.DB) (*Enforcer, error) {
	e, err := casbinx.NewEnforcer(db, modelConf)
	if err != nil {
		return nil, fmt.Errorf("casbinauthz.New: %w", err)
	}
	return &Enforcer{enforcer: e}, nil
}

// Decide implements outbound.Authorizer. It reads the decision stored on the
// matched policy line (the 4th field, dec) via EnforceEx. With no matching
// policy it returns authz.DecisionDeny.
func (e *Enforcer) Decide(ctx context.Context, principal, resource string, action authz.Action) (authz.Decision, error) {
	matched, explain, err := e.enforcer.EnforceEx(principal, resource, string(action))
	if err != nil {
		return authz.DecisionDeny, fmt.Errorf("casbinauthz.Decide: %w", err)
	}
	if !matched || len(explain) < 4 {
		return authz.DecisionDeny, nil
	}
	dec := authz.Decision(explain[3])
	if !dec.Valid() {
		return authz.DecisionDeny, fmt.Errorf("casbinauthz.Decide: invalid stored decision %q", explain[3])
	}
	return dec, nil
}

// staticcheck: ensure Enforcer satisfies the port.
var _ outbound.Authorizer = (*Enforcer)(nil)
