// Package app is the service's use-case layer, organized as CQRS (ADR-0012):
// Application groups the write-side Commands and read-side Queries, each a handler
// defined under command/ or query/. Driving adapters (the Pi wire today, via
// delivery/pilink) depend on the concrete Application and call its handlers' Handle
// methods; the handlers are wired to the app-owned driven ports they need (ADR-0013).
package app

import (
	"harness-workspace/services/harness/internal/app/command"
	"log/slog"
)

// Application is the use-case entrypoint: the set of command and query handlers.
type Application struct {
	Commands Commands
	Queries  Queries
}

// Commands groups the write-side use cases.
type Commands struct {
	SyncModels command.SyncModelsHandler
}

// Queries groups the read-side use cases (none yet).
type Queries struct{}

// New wires the application's handlers over the given driven ports.
func New(uow command.UnitOfWork, logger *slog.Logger) Application {
	return Application{
		Commands: Commands{
			SyncModels: command.NewSyncModelsHandler(uow, logger),
		},
	}
}
