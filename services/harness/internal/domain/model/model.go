// Package model is the model bounded context: the Model aggregate (the first-class
// catalog of provider/model-ids) and its repository port. Agent profiles
// (see internal/domain/agent) reference models by id rather than embedding the
// "provider/model-id" string.
package model

import (
	"errors"
	"fmt"
)

// ErrInvalidModel is returned by Validate for a structurally invalid model.
var ErrInvalidModel = errors.New("model: invalid")

// Model is the aggregate root describing one entry in the model catalog.
type Model struct {
	// ID is the surrogate identity (UUID v7), assigned by the application/adapter.
	ID string
	// Provider is the model vendor (e.g. "anthropic", "openai"). Unique with Slug.
	Provider string
	// Slug is the provider-scoped model name (e.g. "claude-opus-4-7").
	Slug string
	// Enabled reports whether the model may be assigned to an agent in a session.
	Enabled bool
}

// Ref returns the canonical "provider/slug" reference.
func (m Model) Ref() string { return m.Provider + "/" + m.Slug }

// Validate checks the model's invariants.
func (m Model) Validate() error {
	if m.Provider == "" {
		return fmt.Errorf("%w: provider is required", ErrInvalidModel)
	}
	if m.Slug == "" {
		return fmt.Errorf("%w: slug is required", ErrInvalidModel)
	}
	return nil
}
