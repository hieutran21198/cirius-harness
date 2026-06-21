package domain

// Ref is the natural key of a catalog entry — a (client, provider, slug) triple. The
// client is part of the key because model names are client-specific: Pi and opencode
// report the same underlying model under different (provider, slug) names (ADR-0015).
// It is a value object, not an aggregate: immutable, comparable (so it doubles as a map
// key for membership checks), and equal to its value. It is a lightweight
// lookup/command-input type distinct from the Model aggregate and carries no id.
type Ref struct {
	// Client is the reporting client whose registry named the model.
	Client ClientKind
	// Provider is the model vendor (e.g. "anthropic", "openai").
	Provider string
	// Slug is the provider-scoped model name (e.g. "claude-opus-4-7").
	Slug string
}

// String returns the canonical "client:provider/slug" form.
func (r Ref) String() string { return string(r.Client) + ":" + r.Provider + "/" + r.Slug }
