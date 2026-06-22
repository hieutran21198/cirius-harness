// Package appctx carries small request-scoped values across the application layer
// without widening every handler signature. Today it carries the actor — who caused a
// command — which the delivery layer sets and the audit decorator reads.
package appctx

import "context"

type actorKey struct{}

// WithActor returns a context carrying the actor responsible for the work (e.g. the
// reporting client). An empty actor is allowed (it records as unknown).
func WithActor(ctx context.Context, actor string) context.Context {
	return context.WithValue(ctx, actorKey{}, actor)
}

// Actor returns the actor carried by ctx, or "" if none was set.
func Actor(ctx context.Context) string {
	actor, _ := ctx.Value(actorKey{}).(string)
	return actor
}
