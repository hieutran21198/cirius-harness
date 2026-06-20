// Package authz holds the authorization value objects: the three-valued
// Decision and the set of Actions an agent can be authorized for. The Casbin
// authorizer lives in internal/infra/casbin (its interface is defined by the
// consuming use case when one lands — ADR-0013); policies live in Casbin, not in
// the agent table.
package authz

// Decision is the three-valued outcome of an authorization check.
type Decision string

const (
	// DecisionAllow permits the action without prompting.
	DecisionAllow Decision = "allow"
	// DecisionAsk permits the action only after asking the user.
	DecisionAsk Decision = "ask"
	// DecisionDeny refuses the action.
	DecisionDeny Decision = "deny"
)

// Valid reports whether d is a known decision.
func (d Decision) Valid() bool {
	switch d {
	case DecisionAllow, DecisionAsk, DecisionDeny:
		return true
	default:
		return false
	}
}
