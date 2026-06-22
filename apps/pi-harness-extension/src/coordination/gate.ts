/**
 * Gate classification (ADR-0021). A blocking-gated task pauses for human approval before its
 * worker is spawned; an edit-capable task is serialized against other edit-capable tasks in the
 * same wave (concurrent writers in one cwd would corrupt the tree — worktree isolation is a
 * later milestone). Boundaries mirror council's four-gate model (advisory → validating →
 * blocking → escalating) and the task categories.
 */
import type { PlannedTask } from "./types";

const BLOCKING_GATES = new Set(["blocking", "escalating"]);
const EDIT_CATEGORIES = new Set(["implement", "test", "migration", "integrate", "devops", "docs"]);
const RISK_KEYWORDS =
	/\b(auth|payment|secret|security|prod|production|irreversible|delete|drop)\b/i;

/** isBlockingGate reports whether a task needs human approval before it runs. */
export function isBlockingGate(task: PlannedTask): boolean {
	if (BLOCKING_GATES.has((task.gate ?? "").toLowerCase())) return true;
	if ((task.risk_level ?? "").toLowerCase() === "high") return true;
	return RISK_KEYWORDS.test(task.objective ?? "");
}

/** isEditCapable reports whether a task may edit the working tree (so it must not run alongside
 * another edit-capable task in the shared cwd). */
export function isEditCapable(task: PlannedTask): boolean {
	return EDIT_CATEGORIES.has((task.category ?? "").toLowerCase());
}

/** gateMessage describes why a task is gated, for the approval dialog. */
export function gateMessage(task: PlannedTask): string {
	const parts = [`${task.assignee?.agent ?? "an agent"} will run: ${task.objective ?? task.id}`];
	if (task.gate) parts.push(`gate: ${task.gate}`);
	if (task.risk_level) parts.push(`risk: ${task.risk_level}`);
	return parts.join(" · ");
}
