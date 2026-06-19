package agent

// Source identifies where an agent definition originated.
type Source string

const (
	// SourceSystem marks an agent shipped as a harness default.
	SourceSystem Source = "system"
	// SourceUser marks an agent defined by the user's workspace config.
	SourceUser Source = "user"
)

// Valid reports whether s is a known source.
func (s Source) Valid() bool {
	switch s {
	case SourceSystem, SourceUser:
		return true
	default:
		return false
	}
}
