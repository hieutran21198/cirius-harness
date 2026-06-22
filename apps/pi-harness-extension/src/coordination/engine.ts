/**
 * The coordination engine (ADR-0021): drive a persisted council plan by spawning one headless
 * `pi` worker per task, sequenced by waves and depends_on. Waves run sequentially; tasks within a
 * wave run concurrently (Promise.all), each awaiting its dependencies' outputs. Blocking-gated
 * tasks pause for human approval; edit-capable tasks are serialized; progress is reported to the
 * harness via report_run. The harness governs (serves the plan, records progress); Pi executes.
 *
 * Dependencies are injected (exec/request/notify/confirm) so the loop is unit-testable without a
 * real Pi or harness — see engine.test.ts.
 */
import { gateMessage, isBlockingGate, isEditCapable } from "./gate";
import { parseJsonModeStdout } from "./parse";
import { composeTaskPrompt, firstLine, truncate } from "./prompt";
import { assertAcyclic, buildWaves } from "./scheduler";
import type {
	AgentResolvedResp,
	OrchestrationPlan,
	PlannedTask,
	PlanResp,
	RunReportedResp,
	TaskOutcome,
} from "./types";

const SUMMARY_MAX = 500;

/** ExecResult mirrors the Pi extension SDK's pi.exec return shape. */
export interface ExecResult {
	stdout: string;
	stderr: string;
	code: number;
	killed: boolean;
}

/** ExecFn mirrors pi.exec: spawn a command and resolve on exit. */
export type ExecFn = (
	command: string,
	args: string[],
	opts?: { cwd?: string; timeout?: number; signal?: AbortSignal },
) => Promise<ExecResult>;

export interface CoordinatorDeps {
	/** Spawn a headless worker (pi.exec). */
	exec: ExecFn;
	/** Send a harness frame and await its reply (the session's harnessRequest). */
	request: <T>(frame: Record<string, unknown>) => Promise<T>;
	/** Surface a message to the user. */
	notify: (msg: string, level: "info" | "error") => void;
	/** Update the live progress line (best-effort; may be a no-op without a UI). */
	progress: (text: string | undefined) => void;
	/** Ask the human to approve a gated task. */
	confirm: (title: string, message: string) => Promise<boolean>;
	/** Working directory for the workers. */
	cwd: string;
	/** The Pi binary to spawn (default "pi"). */
	piBin: string;
	/** Per-task timeout (ms). */
	taskTimeoutMs: number;
	/** When true, skip spawning and echo the prompt — exercises the loop without models. */
	dryRun: boolean;
}

/** A tiny async mutex: run() serializes its callbacks in call order. */
class Mutex {
	private tail: Promise<unknown> = Promise.resolve();
	run<T>(fn: () => Promise<T>): Promise<T> {
		const result = this.tail.then(fn, fn);
		this.tail = result.then(
			() => undefined,
			() => undefined,
		);
		return result;
	}
}

/** drivePlan fetches the plan and runs it to completion, recording progress as it goes. */
export async function drivePlan(deps: CoordinatorDeps, planId: string | undefined): Promise<void> {
	const fetched = await deps.request<PlanResp>({
		type: "get_plan",
		planId: planId ?? "",
		client: "pi",
	});
	const plan: OrchestrationPlan = fetched.plan;
	const tasks = plan.tasks ?? [];
	if (tasks.length === 0) {
		deps.notify("harness: plan has no tasks to drive", "error");
		return;
	}
	assertAcyclic(tasks);
	const realPlanId = fetched.planId;

	const report = async (frame: Record<string, unknown>): Promise<void> => {
		try {
			await deps.request<RunReportedResp>({
				type: "report_run",
				client: "pi",
				planId: realPlanId,
				...frame,
			});
		} catch (err) {
			// Reporting is best-effort: a failure must never abort the drive.
			deps.notify(`harness: report_run failed (${(err as Error).message})`, "error");
		}
	};

	const byRef = new Map(tasks.map((t) => [t.id, t]));
	const personaCache = new Map<string, AgentResolvedResp>();
	const outcomes = new Map<string, TaskOutcome>();
	const taskPromises = new Map<string, Promise<void>>();
	const gateMutex = new Mutex();
	const editMutex = new Mutex();
	const waves = buildWaves(tasks);

	await report({ planStatus: "driving" });
	deps.notify(
		`harness: driving plan ${realPlanId} — ${tasks.length} tasks, ${waves.length} waves`,
		"info",
	);

	const resolvePersona = async (agent: string): Promise<AgentResolvedResp> => {
		const cached = personaCache.get(agent);
		if (cached) return cached;
		const resolved = await deps.request<AgentResolvedResp>({
			type: "resolve_agent",
			agent,
			client: "pi",
		});
		personaCache.set(agent, resolved);
		return resolved;
	};

	const runTask = async (task: PlannedTask): Promise<void> => {
		const depRefs = (task.depends_on ?? []).filter((d) => taskPromises.has(d));
		await Promise.all(depRefs.map((d) => taskPromises.get(d)));

		const blockedBy = depRefs.find((d) => outcomes.get(d)?.status !== "done");
		if (blockedBy) {
			outcomes.set(task.id, { status: "skipped", output: "" });
			await report({
				task: {
					ref: task.id,
					status: "skipped",
					summary: `dependency ${blockedBy} did not complete`,
				},
			});
			return;
		}

		if (isBlockingGate(task)) {
			const ok = await gateMutex.run(() => deps.confirm(`Approve ${task.id}?`, gateMessage(task)));
			if (!ok) {
				outcomes.set(task.id, { status: "skipped", output: "" });
				await report({
					task: { ref: task.id, status: "skipped", summary: "human declined the gate" },
				});
				return;
			}
		}

		let persona = "";
		let model: string | undefined;
		const agent = task.assignee?.agent ?? "";
		if (agent) {
			try {
				const resolved = await resolvePersona(agent);
				persona = resolved.persona;
				model = resolved.model;
			} catch (err) {
				outcomes.set(task.id, { status: "failed", output: "" });
				await report({
					task: {
						ref: task.id,
						status: "failed",
						summary: `resolve ${agent} failed: ${(err as Error).message}`,
					},
				});
				return;
			}
		}

		await report({ task: { ref: task.id, status: "running" } });
		deps.progress(`▶ ${task.id} (${agent || "agent"})`);

		const depOutputs = depRefs
			.map((d) => ({ ref: d, output: outcomes.get(d)?.output ?? "" }))
			.filter((d) => d.output !== "");
		const prompt = composeTaskPrompt(task, depOutputs);
		const exec = () => runChild(deps, persona, model, prompt);
		const result = isEditCapable(task) ? await editMutex.run(exec) : await exec();

		if (result.error) {
			outcomes.set(task.id, { status: "failed", output: "" });
			await report({
				task: { ref: task.id, status: "failed", summary: truncate(result.error, SUMMARY_MAX) },
			});
		} else {
			outcomes.set(task.id, { status: "done", output: result.text });
			await report({
				task: {
					ref: task.id,
					status: "done",
					summary: truncate(firstLine(result.text), SUMMARY_MAX),
				},
			});
		}
	};

	for (const wave of waves) {
		for (const ref of wave.refs) {
			const task = byRef.get(ref);
			if (task) taskPromises.set(ref, runTask(task));
		}
		await Promise.all(wave.refs.map((ref) => taskPromises.get(ref)));
	}

	await report({ planStatus: "done" });
	deps.progress(undefined);
	deps.notify(`harness: drive complete — ${summarize(outcomes)}`, "info");
}

/** runChild spawns one headless worker and parses its result. */
async function runChild(
	deps: CoordinatorDeps,
	persona: string,
	model: string | undefined,
	prompt: string,
): Promise<{ text: string; error?: string }> {
	if (deps.dryRun) {
		return { text: `[dry-run] ${firstLine(prompt)}` };
	}
	const args = ["--no-extensions", "--mode", "json"];
	if (persona) args.push("--system-prompt", persona);
	if (model) args.push("--model", model);
	args.push(prompt);

	const res = await deps.exec(deps.piBin, args, { cwd: deps.cwd, timeout: deps.taskTimeoutMs });
	if (res.killed) return { text: "", error: "task timed out" };
	const parsed = parseJsonModeStdout(res.stdout);
	if (parsed.error) return { text: parsed.text, error: parsed.error };
	if (res.code !== 0 && !parsed.text) {
		return { text: "", error: `pi exited ${res.code}: ${truncate(res.stderr.trim(), 300)}` };
	}
	return { text: parsed.text };
}

/** summarize counts task outcomes for the closing notification. */
function summarize(outcomes: Map<string, TaskOutcome>): string {
	const counts = { done: 0, failed: 0, skipped: 0 };
	for (const o of outcomes.values()) {
		if (o.status === "done") counts.done++;
		else if (o.status === "failed") counts.failed++;
		else if (o.status === "skipped") counts.skipped++;
	}
	return `${counts.done} done, ${counts.failed} failed, ${counts.skipped} skipped`;
}
