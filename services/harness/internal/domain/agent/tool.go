package agent

// Tool names a capability an agent may be granted in its toolset.
type Tool string

// Known tools an agent may use.
const (
	ToolRead      Tool = "read"
	ToolGrep      Tool = "grep"
	ToolGlob      Tool = "glob"
	ToolList      Tool = "list"
	ToolEdit      Tool = "edit"
	ToolBash      Tool = "bash"
	ToolWebFetch  Tool = "webfetch"
	ToolWebSearch Tool = "websearch"
)

// Valid reports whether t is a known tool.
func (t Tool) Valid() bool {
	switch t {
	case ToolRead, ToolGrep, ToolGlob, ToolList, ToolEdit, ToolBash, ToolWebFetch, ToolWebSearch:
		return true
	default:
		return false
	}
}
