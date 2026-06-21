package domain

// EnvType identifies the kind of environment a session runs in. It keys the
// polymorphic EnvID reference (a container id, a worktree id, or none).
type EnvType string

const (
	// EnvUnset marks a session whose environment is not yet provisioned.
	EnvUnset EnvType = "unset"
	// EnvContainer marks a session running in a container (EnvID is a container id).
	EnvContainer EnvType = "container"
	// EnvWorktree marks a session running in a worktree (EnvID is a worktree id).
	EnvWorktree EnvType = "worktree"
)

// Valid reports whether e is a known environment type.
func (e EnvType) Valid() bool {
	switch e {
	case EnvUnset, EnvContainer, EnvWorktree:
		return true
	default:
		return false
	}
}
