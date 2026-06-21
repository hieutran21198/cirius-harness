package domain

import (
	"errors"
	"fmt"
)

// ErrInvalidTool is returned by Validate for a structurally invalid tool.
var ErrInvalidTool = errors.New("tool: invalid")

// ToolName identifies a capability in the catalog.
type ToolName string

// Known capability names.
const (
	ToolRead      ToolName = "read"
	ToolGrep      ToolName = "grep"
	ToolGlob      ToolName = "glob"
	ToolList      ToolName = "list"
	ToolEdit      ToolName = "edit"
	ToolBash      ToolName = "bash"
	ToolWebFetch  ToolName = "webfetch"
	ToolWebSearch ToolName = "websearch"
)

// Valid reports whether n is a known capability name.
func (n ToolName) Valid() bool {
	switch n {
	case ToolRead, ToolGrep, ToolGlob, ToolList, ToolEdit, ToolBash, ToolWebFetch, ToolWebSearch:
		return true
	default:
		return false
	}
}

// ToolID is a Tool's surrogate identity (a UUID v7). A named string type so a tool id
// can't be silently passed where another aggregate's id is expected; an agent's tool
// grants are a []ToolID.
type ToolID string

// Tool is the aggregate root describing one entry in the capability catalog.
// Agents are granted tools from this catalog via the agent_tools junction.
type Tool struct {
	id          ToolID
	name        ToolName
	description string
}

// NewTool assembles a fresh tool catalog entry from its attributes, minting its own
// identity (UUID v7), and validates it.
func NewTool(name ToolName, description string) (Tool, error) {
	t := Tool{id: newID[ToolID](), name: name, description: description}
	return t, t.Validate()
}

// RehydrateTool reconstitutes a Tool from its persisted state and validates it.
func RehydrateTool(id ToolID, name ToolName, description string) (Tool, error) {
	t := Tool{id: id, name: name, description: description}
	return t, t.Validate()
}

// Validate checks the tool's invariants.
func (t Tool) Validate() error {
	if t.id == "" {
		return fmt.Errorf("%w: id is required", ErrInvalidTool)
	}
	if !t.name.Valid() {
		return fmt.Errorf("%w: unknown name %q", ErrInvalidTool, t.name)
	}
	return nil
}
