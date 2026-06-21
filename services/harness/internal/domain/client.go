package domain

// ClientKind identifies the AI coding client (the "citizen") the harness governs.
// Model names are a property of the client's own registry — Pi and opencode report
// the same underlying model under different (provider, slug) names — so the client is
// part of a catalog entry's natural key (see Ref).
type ClientKind string

const (
	// ClientPi is the Pi coding agent (the first citizen).
	ClientPi ClientKind = "pi"
	// ClientOpencode is the opencode client.
	ClientOpencode ClientKind = "opencode"
)

// Valid reports whether c is a known client.
func (c ClientKind) Valid() bool {
	switch c {
	case ClientPi, ClientOpencode:
		return true
	default:
		return false
	}
}
