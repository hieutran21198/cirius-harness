/**
 * Compose the prompt handed to a task's headless worker (ADR-0021). The agent's persona is the
 * worker's system prompt (passed via --system-prompt); this is the user-message body: the task's
 * objective, its inputs/expected-output/definition-of-done, and the outputs of the tasks it
 * depends on (truncated) so a dependent task has its upstream context.
 */
import type { PlannedTask } from "./types";

const MAX_CONTEXT = 4000;

/** truncate caps a string to n chars with an explicit marker, so threaded context stays bounded. */
export function truncate(s: string, n: number): string {
	return s.length <= n ? s : `${s.slice(0, n)}\n…[truncated]`;
}

/** firstLine returns the first non-empty line of s (used for a short task summary). */
export function firstLine(s: string): string {
	for (const line of s.split("\n")) {
		const t = line.trim();
		if (t) return t;
	}
	return "";
}

export function composeTaskPrompt(
	task: PlannedTask,
	depOutputs: { ref: string; output: string }[],
): string {
	const lines: string[] = [];
	lines.push(`Task ${task.id}: ${(task.objective ?? "").trim()}`.trim());
	if (task.assignee?.lens) lines.push(`Focus (lens): ${task.assignee.lens}`);
	if (task.inputs?.length) lines.push(`Inputs: ${task.inputs.join(", ")}`);
	if (task.expected_output) lines.push(`Expected output: ${task.expected_output}`);
	if (task.dod?.length) {
		lines.push(`Definition of done:\n${task.dod.map((d) => `- ${d}`).join("\n")}`);
	}
	if (depOutputs.length) {
		lines.push("\nContext from completed upstream tasks:");
		for (const d of depOutputs) {
			lines.push(`\n## ${d.ref}\n${truncate(d.output, MAX_CONTEXT)}`);
		}
	}
	lines.push("\nStay within this task's objective and produce the expected output.");
	return lines.join("\n");
}
