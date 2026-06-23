/**
 * Shared types for the coordination engine (ADR-0021). The plan shapes mirror the harness
 * OrchestrationPlan contract returned over the `get_plan` frame; the run-state types mirror the
 * `report_run` frame. Kept structural (optional fields) so a partial plan never throws.
 */

export interface Assignee {
	agent: string;
	lens?: string;
}

export interface PlannedTask {
	id: string;
	category?: string;
	assignee?: Assignee;
	objective?: string;
	inputs?: string[];
	expected_output?: string;
	depends_on?: string[];
	wave?: number;
	dod?: string[];
	gate?: string;
	risk_level?: string;
}

export interface OrchestrationPlan {
	intent?: string;
	goal?: string;
	tasks: PlannedTask[];
	waves?: { wave: number; tasks: string[] }[];
}

/** Reply to the `get_plan` frame. */
export interface PlanResp {
	planId: string;
	status: string;
	plan: OrchestrationPlan;
	taskIds: Record<string, string>;
}

/** Reply to the `resolve_agent` frame. */
export interface AgentResolvedResp {
	name: string;
	persona: string;
	model?: string;
}

/** Reply to the `report_run` frame. */
export interface RunReportedResp {
	planRunId: string;
	status: string;
}

export type TaskStatus = "pending" | "running" | "done" | "failed" | "skipped";

/**
 * TaskReportEnvelope mirrors the harness's report contract (ADR-0023): the structured result a
 * worker emits, which the coordinator extracts and submits on the report_run frame. Kept
 * structural (optional fields) so a partial or absent envelope never throws — the coordinator
 * normalizes it before reporting.
 */
export interface TaskReportEnvelope {
	status?: string;
	summary?: string;
	dod_met?: boolean;
	confidence?: string;
	outputs?: { kind?: string; ref?: string; description?: string }[];
	findings?: { severity?: string; location?: string; issue?: string; suggestion?: string }[];
	verification?: string[];
	follow_ups?: string[];
	open_questions?: string[];
}

/** The outcome of running one task: where it ended up, its normalized summary, and the envelope. */
export interface TaskOutcome {
	status: TaskStatus;
	output: string;
}

/** One task's normalized report in a `get_reports` reply (envelope as the harness stored it). */
export interface ReportView {
	ref: string;
	agent: string;
	envelope: TaskReportEnvelope;
}

/** Reply to the `get_reports` frame. */
export interface ReportsResp {
	planRunId: string;
	reports: ReportView[];
}

/** Reply to the `submit_decision` frame. */
export interface DecisionRecordedResp {
	decisionId: string;
	planRunId: string;
}
