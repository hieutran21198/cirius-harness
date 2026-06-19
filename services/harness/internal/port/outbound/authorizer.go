package outbound

import (
	"context"

	"harness-workspace/services/harness/internal/domain/authz"
)

// Authorizer answers authorization questions for a principal (an agent name).
type Authorizer interface {
	// Decide returns the policy decision for the principal performing action on
	// resource. resource is "*" for the current coarse model; path/url patterns
	// become meaningful once fine-grained policies are seeded.
	Decide(ctx context.Context, principal, resource string, action authz.Action) (authz.Decision, error)
}
