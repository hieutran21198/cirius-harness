// Package app is the service's use-case layer, organized as CQRS (ADR-0012):
// Application groups the write-side Commands and read-side Queries, each a handler
// defined under command/ or query/. Driving adapters (the Pi wire today, via
// delivery/pilink) depend on the concrete Application and call its handlers' Handle
// methods; the handlers are wired to the app-owned driven ports they need (ADR-0013).
package app

import (
	"log/slog"

	"harness-workspace/services/harness/internal/app/command"
	"harness-workspace/services/harness/internal/app/query"
)

// Application is the use-case entrypoint: the set of command and query handlers.
type Application struct {
	Commands Commands
	Queries  Queries
}

// Commands groups the write-side use cases.
type Commands struct {
	SyncModels     command.SyncModelsHandler
	StartSession   command.StartSessionHandler
	RecordAgentRun command.RecordAgentRunHandler
	SubmitPlan     command.SubmitPlanHandler
}

// Queries groups the read-side use cases.
type Queries struct {
	ResolveAgent query.ResolveAgentHandler
}

// New wires the application's handlers over the given driven ports.
func New(uow command.UnitOfWork, rs query.ReadStore, logger *slog.Logger) Application {
	return Application{
		Commands: Commands{
			SyncModels:     command.NewSyncModelsHandler(uow, logger),
			StartSession:   command.NewStartSessionHandler(uow, logger),
			RecordAgentRun: command.NewRecordAgentRunHandler(uow, logger),
			SubmitPlan:     command.NewSubmitPlanHandler(uow, logger),
		},
		Queries: Queries{
			ResolveAgent: query.NewResolveAgentHandler(rs, logger),
		},
	}
}
