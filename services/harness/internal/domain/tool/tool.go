// Package tool is the tool bounded context: the Tool aggregate (the capability
// catalog) and its repository port. Agents are granted tools from this catalog
// via the agent_tools junction (see internal/domain/agent).
package tool

import (
	"errors"
	"fmt"
)

// ErrInvalidTool is returned by Validate for a structurally invalid tool.
var ErrInvalidTool = errors.New("tool: invalid")

// Name identifies a capability in the catalog.
type Name string

// Known capability names.
const (
	NameRead      Name = "read"
	NameGrep      Name = "grep"
	NameGlob      Name = "glob"
	NameList      Name = "list"
	NameEdit      Name = "edit"
	NameBash      Name = "bash"
	NameWebFetch  Name = "webfetch"
	NameWebSearch Name = "websearch"
)

// Valid reports whether n is a known capability name.
func (n Name) Valid() bool {
	switch n {
	case NameRead, NameGrep, NameGlob, NameList, NameEdit, NameBash, NameWebFetch, NameWebSearch:
		return true
	default:
		return false
	}
}

// Tool is the aggregate root describing one entry in the capability catalog.
type Tool struct {
	// ID is the surrogate identity (UUID v7), assigned by the application/adapter.
	ID string
	// Name is the capability's unique business key.
	Name Name
	// Description is a human-facing summary of the capability.
	Description string
}

// Validate checks the tool's invariants.
func (t Tool) Validate() error {
	if !t.Name.Valid() {
		return fmt.Errorf("%w: unknown name %q", ErrInvalidTool, t.Name)
	}
	return nil
}
