/**
 * cirius-harness — Pi client integration (ADR-0008).
 *
 * Direction: Pi launches the harness. On session_start this extension spawns
 * `harness serve` as a child process and performs a hello/ready handshake over
 * stdio (newline-delimited JSON). The child is killed on session_shutdown, so
 * each Pi session owns exactly one harness process.
 *
 * Slice 1 is connect-only: it proves the channel is live (reporting the DB schema
 * version) and surfaces it in the footer. No governance yet — model handoff,
 * permission gating, and tool grants come later and will ride this same channel.
 *
 * Framing note: the harness speaks strict LF-delimited JSON. We split child
 * stdout on "\n" by hand and never use Node `readline`, which also breaks on
 * U+2028/U+2029 (valid inside JSON strings) — see Pi's docs/rpc.md.
 */
import { type ChildProcess, spawn } from "node:child_process";
import * as fs from "node:fs";
import * as path from "node:path";
import type { ExtensionAPI, ExtensionContext } from "@earendil-works/pi-coding-agent";

const STATUS_KEY = "harness";
const BINARY_REL = ".cirius-harness/bin/harness";
const HANDSHAKE_TIMEOUT_MS = 5000;

interface ReadyResp {
	type: "ready";
	schemaVersion: number;
	dbPath: string;
	pid: number;
}

export default function (pi: ExtensionAPI) {
	// At most one harness child per session. Tracked at module scope so
	// session_shutdown (and a re-fired session_start on /reload, /new, /resume)
	// can tear it down idempotently.
	let child: ChildProcess | undefined;

	const setStatus = (ctx: ExtensionContext, text: string | undefined) => {
		if (ctx.hasUI) ctx.ui.setStatus(STATUS_KEY, text);
	};
	const notify = (ctx: ExtensionContext, msg: string, level: "info" | "error") => {
		if (ctx.hasUI) ctx.ui.notify(msg, level);
	};

	const teardown = () => {
		const c = child;
		child = undefined;
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

		// Wait for the ready frame (or an error / timeout). Buffer is local to this
		// spawn so a superseded child can never bleed into the next one.
		const ready = new Promise<ReadyResp>((resolve, reject) => {
			const timer = setTimeout(
				() => reject(new Error("handshake timed out")),
				HANDSHAKE_TIMEOUT_MS,
			);
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
					let msg: { type?: string; message?: string; schemaVersion?: number };
					try {
						msg = JSON.parse(line);
					} catch {
						continue; // ignore non-JSON noise on the protocol channel
					}
					if (msg.type === "ready") {
						clearTimeout(timer);
						resolve(msg as ReadyResp);
					} else if (msg.type === "error") {
						clearTimeout(timer);
						reject(new Error(msg.message ?? "harness error"));
					}
				}
			});
		});

		proc.stdin?.write(`${JSON.stringify({ type: "hello", cwd: ctx.cwd, pid: process.pid })}\n`);

		try {
			const r = await ready;
			if (child !== proc) return; // session changed while we waited
			setStatus(ctx, `● harness · schema v${r.schemaVersion}`);
			notify(ctx, `harness connected (schema v${r.schemaVersion})`, "info");
		} catch (err) {
			if (child === proc) teardown();
			setStatus(ctx, undefined);
			notify(ctx, `harness: handshake failed (${(err as Error).message})`, "error");
		}
	});

	pi.on("session_shutdown", async (_event, ctx: ExtensionContext) => {
		teardown();
		setStatus(ctx, undefined);
	});
}
