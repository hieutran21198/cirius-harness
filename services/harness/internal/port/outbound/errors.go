// Package outbound declares the service's driven (outbound) ports: the
// interfaces the application calls to reach infrastructure — aggregate
// repositories and the authorizer. Implementations live in
// internal/adapter/outbound. A port is added here when the application requires
// the dependency, not speculatively.
package outbound

import "errors"

// ErrNotFound is returned by a repository when no aggregate matches the given key.
var ErrNotFound = errors.New("outbound: not found")
