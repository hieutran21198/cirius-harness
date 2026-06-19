package authz

// Action is a capability an agent may be authorized to perform. These mirror
// the permission keys of the original schema.
type Action string

// Known actions an agent may be authorized for.
const (
	ActionRead      Action = "read"
	ActionEdit      Action = "edit"
	ActionBash      Action = "bash"
	ActionWebFetch  Action = "webfetch"
	ActionWebSearch Action = "websearch"
)

// Valid reports whether a is a known action.
func (a Action) Valid() bool {
	switch a {
	case ActionRead, ActionEdit, ActionBash, ActionWebFetch, ActionWebSearch:
		return true
	default:
		return false
	}
}
