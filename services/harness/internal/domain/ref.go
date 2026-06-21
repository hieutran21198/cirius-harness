package domain

// Ref is the natural key of a catalog entry — a (provider, slug) pair. It is a
// value object, not an aggregate: immutable, comparable (so it doubles as a map
// key for membership checks), and equal to its value. It is a lightweight
// lookup/command-input type distinct from the Model aggregate and carries no id.
type Ref struct {
	// Provider is the model vendor (e.g. "anthropic", "openai").
	Provider string
	// Slug is the provider-scoped model name (e.g. "claude-opus-4-7").
	Slug string
}

// String returns the canonical "provider/slug" form.
func (r Ref) String() string { return r.Provider + "/" + r.Slug }
