package domain

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
