/**
 * Scheduling: turn a plan's tasks into ordered waves (ADR-0021). Waves run sequentially; tasks
 * within a wave run concurrently. The engine additionally awaits each task's `depends_on` via a
 * promise map, so a dependency in an earlier wave is honored regardless of grouping. assertAcyclic
 * guards against a dependency cycle, which would otherwise deadlock those awaits.
 */
import type { PlannedTask } from "./types";

export interface Wave {
	wave: number;
	refs: string[];
}

/**
 * buildWaves groups tasks by ascending wave number (the get_plan contract sets each task's wave;
 * a task with no wave falls into wave 0). Refs within a wave are sorted for deterministic order.
 */
export function buildWaves(tasks: PlannedTask[]): Wave[] {
	const byWave = new Map<number, string[]>();
	for (const t of tasks) {
		const w = t.wave ?? 0;
		const arr = byWave.get(w) ?? [];
		arr.push(t.id);
		byWave.set(w, arr);
	}
	return [...byWave.entries()]
		.sort((a, b) => a[0] - b[0])
		.map(([wave, refs]) => ({ wave, refs: [...refs].sort() }));
}

/**
 * assertAcyclic throws if the tasks' depends_on edges contain a cycle. Unknown deps (refs not in
 * the plan) are ignored — server-side validation rejects dangling deps, and ignoring them here
 * keeps the check robust to partial input.
 */
export function assertAcyclic(tasks: PlannedTask[]): void {
	const byRef = new Map(tasks.map((t) => [t.id, t]));
	const state = new Map<string, "visiting" | "done">();

	const visit = (ref: string, trail: string[]): void => {
		const s = state.get(ref);
		if (s === "done") return;
		if (s === "visiting") {
			throw new Error(`dependency cycle: ${[...trail, ref].join(" → ")}`);
		}
		state.set(ref, "visiting");
		const task = byRef.get(ref);
		for (const dep of task?.depends_on ?? []) {
			if (byRef.has(dep)) visit(dep, [...trail, ref]);
		}
		state.set(ref, "done");
	};

	for (const t of tasks) visit(t.id, []);
}
