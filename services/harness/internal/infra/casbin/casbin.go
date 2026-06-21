// Package casbin is the Casbin-backed authorizer (a driven adapter — ADR-0013).
// Its policy is persisted in the shared database's casbin_rule table. It exposes a
// concrete Decide; the authorizer interface is defined by the consuming use case
// when one lands.
package casbin

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/casbin/casbin/v3"
	"gorm.io/gorm"

	"harness-workspace/packages/go/casbinx"
	"harness-workspace/services/harness/internal/domain"
)

//go:embed model.conf
var modelConf string

// Enforcer is a Casbin-backed authorizer.
type Enforcer struct {
	enforcer *casbin.Enforcer
}

// New builds an Enforcer over db, loading the embedded ABAC model and the
// persisted policy.
func New(db *gorm.DB) (*Enforcer, error) {
	e, err := casbinx.NewEnforcer(db, modelConf)
	if err != nil {
		return nil, fmt.Errorf("casbin.New: %w", err)
	}
	return &Enforcer{enforcer: e}, nil
}

// Decide reads the decision stored on the matched policy line (the 4th field, dec)
// via EnforceEx. With no matching policy it returns domain.DecisionDeny.
func (e *Enforcer) Decide(ctx context.Context, principal, resource string, action domain.Action) (domain.Decision, error) {
	matched, explain, err := e.enforcer.EnforceEx(principal, resource, string(action))
	if err != nil {
		return domain.DecisionDeny, fmt.Errorf("casbin.Decide: %w", err)
	}
	if !matched || len(explain) < 4 {
		return domain.DecisionDeny, nil
	}
	dec := domain.Decision(explain[3])
	if !dec.Valid() {
		return domain.DecisionDeny, fmt.Errorf("casbin.Decide: invalid stored decision %q", explain[3])
	}
	return dec, nil
}
