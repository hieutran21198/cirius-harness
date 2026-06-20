package model

// Ref is the natural key of a catalog entry — a (provider, slug) pair. It is a
// lightweight lookup/command-input value, distinct from the Model aggregate: it
// carries no id and is comparable, so it doubles as a map key for membership checks.
type Ref struct {
	// Provider is the model vendor (e.g. "anthropic", "openai").
	Provider string
	// Slug is the provider-scoped model name (e.g. "claude-opus-4-7").
	Slug string
}

// String returns the canonical "provider/slug" form.
func (r Ref) String() string { return r.Provider + "/" + r.Slug }
