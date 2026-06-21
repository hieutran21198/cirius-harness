package domain

import (
	"errors"
	"fmt"
)

// ErrInvalidModel is returned by Validate for a structurally invalid model.
var ErrInvalidModel = errors.New("model: invalid")

// ModelID is a Model's surrogate identity (a UUID v7). A named string type so a model
// id can't be silently passed where another aggregate's id is expected.
type ModelID string

// Model is the aggregate root describing one entry in the model catalog. Agents
// reference models by id rather than embedding the "provider/slug" string; which
// model plays an agent's role is bound per session.
type Model struct {
	id       ModelID
	provider string
	slug     string
	enabled  bool
}

// NewModel assembles a fresh catalog entry from its natural key, minting its own
// identity (UUID v7), enabled by default, and validates it. The app supplies only the
// business attributes and reads the id back from the aggregate when it needs it.
func NewModel(provider, slug string) (Model, error) {
	m := Model{id: newID[ModelID](), provider: provider, slug: slug, enabled: true}
	return m, m.Validate()
}

// RehydrateModel reconstitutes a Model from its persisted state (the repository's
// inbound constructor): it takes every stored field as-is — no creation defaults —
// and validates structural integrity.
func RehydrateModel(id ModelID, provider, slug string, enabled bool) (Model, error) {
	m := Model{id: id, provider: provider, slug: slug, enabled: enabled}
	return m, m.Validate()
}

// Reference is the model's natural key (provider, slug) — the value other contexts
// use to refer to it without depending on its identity.
func (m Model) Reference() Ref { return Ref{Provider: m.provider, Slug: m.slug} }

// String is the canonical "provider/slug" display form.
func (m Model) String() string { return m.provider + "/" + m.slug }

// ModelSnapshot is the persistence grouped view of a Model: its whole state,
// grouped for storage and reconstitution. It is the only way a Model's state
// leaves the domain; the repository maps it to a row, and RehydrateModel mirrors
// its fields back.
type ModelSnapshot struct {
	ID       ModelID
	Provider string
	Slug     string
	Enabled  bool
}

// Snapshot returns the model's persistence view.
func (m Model) Snapshot() ModelSnapshot {
	return ModelSnapshot{ID: m.id, Provider: m.provider, Slug: m.slug, Enabled: m.enabled}
}

// Validate checks the model's invariants.
func (m Model) Validate() error {
	if m.id == "" {
		return fmt.Errorf("%w: id is required", ErrInvalidModel)
	}
	if m.provider == "" {
		return fmt.Errorf("%w: provider is required", ErrInvalidModel)
	}
	if m.slug == "" {
		return fmt.Errorf("%w: slug is required", ErrInvalidModel)
	}
	return nil
}
