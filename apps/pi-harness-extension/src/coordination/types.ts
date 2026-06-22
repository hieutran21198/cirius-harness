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

/** The outcome of running one task: where it ended up and what it produced. */
export interface TaskOutcome {
	status: TaskStatus;
	output: string;
}
