package project

// Kind classifies a project by its repository layout.
type Kind string

const (
	// KindSingle is a single-purpose repository.
	KindSingle Kind = "single"
	// KindMonorepo is a repository hosting multiple modules/services.
	KindMonorepo Kind = "monorepo"
)

// Valid reports whether k is a known kind.
func (k Kind) Valid() bool {
	switch k {
	case KindSingle, KindMonorepo:
		return true
	default:
		return false
	}
}
