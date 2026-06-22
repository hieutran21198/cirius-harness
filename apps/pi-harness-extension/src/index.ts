/**
 * cirius-harness — Pi client integration (ADR-0008, ADR-0011).
 *
 * Direction: Pi launches the harness. On session_start this extension spawns
 * `harness serve` as a child process, performs a hello/ready handshake, then syncs
 * Pi's enabled models into the harness catalog — all over stdio (newline-delimited
 * JSON). The child is killed on session_shutdown, so each Pi session owns exactly
 * one harness process.
 *
 * Slice 1: connect + model sync. The harness learns which models exist from the
 * client (ADR-0011) rather than shipping a hardcoded list.
 *
 * Governance, first slice: the `/council` command (ADR-0016). The harness governs
 * (resolves the council agent's persona); Pi executes (runs one turn under it). The
 * command asks the harness to resolve council, then drives a single turn — it sets a
 * one-shot flag and sends the user's message; a `before_agent_start` hook swaps in the
 * council persona for that turn only, then reverts. The harness never calls a model
 * (AGENTS.md) — it only says who council is. Model/permission governance come later.
 *
 * Framing note: the harness speaks strict LF-delimited JSON. We split child stdout
 * on "\n" by hand and never use Node `readline`, which also breaks on U+2028/U+2029
 * (valid inside JSON strings) — see Pi's docs/rpc.md. Requests carry an `id`; the
 * harness echoes it on the reply, so a small pending-request map correlates them.
 */
import { type ChildProcess, spawn } from "node:child_process";
import * as fs from "node:fs";
import * as path from "node:path";
import type {
	ExtensionAPI,
	ExtensionCommandContext,
	ExtensionContext,
} from "@earendil-works/pi-coding-agent";

const STATUS_KEY = "harness";
const BINARY_REL = ".cirius-harness/bin/harness";
const REQUEST_TIMEOUT_MS = 5000;

interface ReadyResp {
	schemaVersion: number;
	dbPath: string;
	pid: number;
}

interface ModelsSyncedResp {
	added: number;
	total: number;
}

interface AgentResolvedResp {
	name: string;
	persona: string;
	model?: string;
}

// A harness request function bound to one child's stdio (id-correlated reply/timeout).
type RequestFn = <T>(frame: Record<string, unknown>) => Promise<T>;

export default function (pi: ExtensionAPI) {
	// At most one harness child per session. Tracked at module scope so
	// session_shutdown (and a re-fired session_start on /reload, /new, /resume)
	// can tear it down idempotently.
	let child: ChildProcess | undefined;
	// The request fn for the live child, published for the command handlers (which are
	// registered once at factory scope, outside session_start). Cleared on teardown.
	let harnessRequest: RequestFn | undefined;

	// Council one-shot turn state: the command arms these, the before_agent_start hook
	// consumes them for exactly one turn (ADR-0016).
	let councilPending = false;
	let councilPersona = "";

	const setStatus = (ctx: ExtensionContext, text: string | undefined) => {
		if (ctx.hasUI) ctx.ui.setStatus(STATUS_KEY, text);
	};
	const notify = (ctx: ExtensionContext, msg: string, level: "info" | "error") => {
		if (ctx.hasUI) ctx.ui.notify(msg, level);
	};

	const teardown = () => {
		const c = child;
		child = undefined;
		harnessRequest = undefined;
		if (!c) return;
		try {
			c.stdin?.end();
		} catch {
			/* already closed */
		}
		try {
			c.kill();
		} catch {
			/* already gone */
		}
	};

	pi.on("session_start", async (_event, ctx: ExtensionContext) => {
		teardown(); // a reload/new/resume re-fires session_start — start clean

		const bin = path.join(ctx.cwd, BINARY_REL);
		if (!fs.existsSync(bin)) {
			setStatus(ctx, undefined);
			notify(ctx, "harness: binary missing — run: devenv tasks run harness:build", "error");
			return;
		}

		setStatus(ctx, "○ harness connecting…");

		let proc: ChildProcess;
		try {
			proc = spawn(bin, ["serve"], { cwd: ctx.cwd, stdio: ["pipe", "pipe", "pipe"] });
		} catch (err) {
			setStatus(ctx, undefined);
			notify(ctx, `harness: failed to launch (${(err as Error).message})`, "error");
			return;
		}
		child = proc;

		proc.on("error", (err: Error) => {
			if (child !== proc) return;
			child = undefined;
			setStatus(ctx, undefined);
			notify(ctx, `harness: process error (${err.message})`, "error");
		});
		proc.on("exit", (code: number | null) => {
			if (child !== proc) return; // superseded by a newer child
			child = undefined;
			setStatus(ctx, undefined);
			if (code) notify(ctx, `harness: exited (code ${code})`, "error");
		});
		proc.stderr?.setEncoding("utf8");
		proc.stderr?.on("data", (chunk: string) => console.error(`[harness] ${chunk.trimEnd()}`));

		// Request router: one LF line-reader over this child's stdout resolves pending
		// requests by `id`. State is local to this spawn so a superseded child can
		// never bleed into the next one.
		const pending = new Map<
			string,
			{ resolve: (v: unknown) => void; reject: (e: Error) => void }
		>();
		let seq = 0;
		let buf = "";
		proc.stdout?.setEncoding("utf8");
		proc.stdout?.on("data", (chunk: string) => {
			buf += chunk;
			let nl: number;
			// biome-ignore lint/suspicious/noAssignInExpressions: standard line-split loop
			while ((nl = buf.indexOf("\n")) >= 0) {
				const line = buf.slice(0, nl);
				buf = buf.slice(nl + 1);
				if (!line) continue;
				let msg: { type?: string; id?: string; message?: string };
				try {
					msg = JSON.parse(line);
				} catch {
					continue; // ignore non-JSON noise on the protocol channel
				}
				const waiter = msg.id ? pending.get(msg.id) : undefined;
				if (!waiter || !msg.id) continue;
				pending.delete(msg.id);
				if (msg.type === "error") waiter.reject(new Error(msg.message ?? "harness error"));
				else waiter.resolve(msg);
			}
		});

		// request writes a frame with a fresh id and resolves on the matching reply
		// (or rejects on an error frame / timeout).
		const request = <T>(frame: Record<string, unknown>): Promise<T> => {
			const id = `r${++seq}`;
			return new Promise<T>((resolve, reject) => {
				const timer = setTimeout(() => {
					pending.delete(id);
					reject(new Error("request timed out"));
				}, REQUEST_TIMEOUT_MS);
				pending.set(id, {
					resolve: (v) => {
						clearTimeout(timer);
						resolve(v as T);
					},
					reject: (e) => {
						clearTimeout(timer);
						reject(e);
					},
				});
				try {
					proc.stdin?.write(`${JSON.stringify({ ...frame, id })}\n`);
				} catch (err) {
					clearTimeout(timer);
					pending.delete(id);
					reject(err as Error);
				}
			});
		};

		// Publish this child's request fn for the command handlers (e.g. /council).
		harnessRequest = request;

		try {
			const ready = await request<ReadyResp>({ type: "hello", cwd: ctx.cwd, pid: process.pid });
			if (child !== proc) return; // session changed while we waited
			setStatus(ctx, `● harness · schema v${ready.schemaVersion}`);

			// Sync Pi's enabled models (those with configured auth) into the catalog.
			const models = ctx.modelRegistry
				.getAvailable()
				.map((m) => ({ provider: m.provider, slug: m.id }));
			const synced = await request<ModelsSyncedResp>({ type: "models", client: "pi", models });
			if (child !== proc) return;

			setStatus(ctx, `● harness · schema v${ready.schemaVersion} · ${synced.total} models`);
			notify(
				ctx,
				`harness connected (schema v${ready.schemaVersion}; synced ${synced.added} new / ${synced.total} models)`,
				"info",
			);
		} catch (err) {
			if (child === proc) teardown();
			setStatus(ctx, undefined);
			notify(ctx, `harness: handshake failed (${(err as Error).message})`, "error");
		}
	});

	// /council: weigh the request, produce a strategy plan. The harness resolves the
	// council agent's persona; we run one turn under it (ADR-0016). Registered once at
	// factory scope — the before_agent_start hook below must not be re-added per session.
	pi.registerCommand("council", {
		description: "Weigh the request and produce a strategy plan (harness council agent).",
		handler: async (args: string, ctx: ExtensionCommandContext) => {
			const message = args.trim();
			if (!message) {
				notify(ctx, "usage: /council <what you want planned>", "error");
				return;
			}
			if (!harnessRequest) {
				notify(ctx, "harness: not connected — cannot resolve council", "error");
				return;
			}
			let resolved: AgentResolvedResp;
			try {
				resolved = await harnessRequest<AgentResolvedResp>({
					type: "resolve_agent",
					agent: "council",
					client: "pi",
				});
			} catch (err) {
				notify(ctx, `harness: council resolve failed (${(err as Error).message})`, "error");
				return;
			}
			// Arm the one-shot persona, then trigger the turn. before_agent_start swaps the
			// system prompt for this turn only; the next turn reverts to the default.
			councilPersona = resolved.persona;
			councilPending = true;
			pi.sendUserMessage(message);
		},
	});

	pi.on("before_agent_start", () => {
		if (!councilPending) return undefined;
		councilPending = false; // one-shot — only the /council turn runs as council
		return { systemPrompt: councilPersona };
	});

	pi.on("session_shutdown", async (_event, ctx: ExtensionContext) => {
		teardown();
		setStatus(ctx, undefined);
	});
}
