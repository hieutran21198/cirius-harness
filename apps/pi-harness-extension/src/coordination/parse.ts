/**
 * Parse the stdout of a headless `pi --mode json` run (ADR-0021): a stream of AgentSessionEvent
 * JSON objects, one per line. The final assistant result is the text of the last assistant
 * `message_end` event; a top-level `error` event or an assistant message that stopped on
 * error/aborted surfaces as an error. Split on "\n" by hand (never readline — U+2028/U+2029 are
 * valid inside JSON strings), matching the harness link's framing discipline.
 */

import type { TaskReportEnvelope } from "./types";

interface ParsedResult {
	text: string;
	error?: string;
}

function asRecord(v: unknown): Record<string, unknown> | undefined {
	return v !== null && typeof v === "object" && !Array.isArray(v)
		? (v as Record<string, unknown>)
		: undefined;
}

function strOf(v: unknown): string | undefined {
	return typeof v === "string" ? v : undefined;
}

function messageText(m: Record<string, unknown>): string {
	const content = m.content;
	if (typeof content === "string") return content;
	if (Array.isArray(content)) {
		return content
			.map((b) => {
				const block = asRecord(b);
				return block && block.type === "text" ? (strOf(block.text) ?? "") : "";
			})
			.join("");
	}
	return "";
}

// extractReport pulls a worker's structured report envelope (ADR-0023) out of its final assistant
// text. It prefers a fenced ```json block (what the report contract instructs the worker to emit),
// then any balanced {…} object, and accepts the first that parses to an object carrying a `summary`
// string — so prose and unrelated JSON yield nothing. Returns undefined when no envelope is found
// (the coordinator then synthesizes a minimal one so the drive never breaks).
export function extractReport(text: string): TaskReportEnvelope | undefined {
	const asEnvelope = (s: string): TaskReportEnvelope | undefined => {
		try {
			const v: unknown = JSON.parse(s);
			if (v && typeof v === "object" && !Array.isArray(v)) {
				const obj = v as Record<string, unknown>;
				if (typeof obj.summary === "string") return obj as TaskReportEnvelope;
			}
		} catch {
			/* not JSON — keep scanning */
		}
		return undefined;
	};
	const fenced = [...text.matchAll(/```json\s*([\s\S]*?)```/gi)].map((m) => m[1].trim());
	for (const c of [...fenced, ...jsonObjects(text)]) {
		const e = asEnvelope(c);
		if (e) return e;
	}
	return undefined;
}

// jsonObjects returns every balanced top-level {…} substring in text, scanning string/escape-aware
// so braces inside JSON strings don't throw off the depth count (mirrors index.ts's scanner).
function jsonObjects(text: string): string[] {
	const out: string[] = [];
	let depth = 0;
	let start = -1;
	let inStr = false;
	let esc = false;
	for (let i = 0; i < text.length; i++) {
		const c = text[i];
		if (inStr) {
			if (esc) esc = false;
			else if (c === "\\") esc = true;
			else if (c === '"') inStr = false;
			continue;
		}
		if (c === '"') inStr = true;
		else if (c === "{") {
			if (depth === 0) start = i;
			depth++;
		} else if (c === "}" && depth > 0) {
			depth--;
			if (depth === 0 && start >= 0) {
				out.push(text.slice(start, i + 1));
				start = -1;
			}
		}
	}
	return out;
}

export function parseJsonModeStdout(stdout: string): ParsedResult {
	let text = "";
	let error: string | undefined;
	for (const line of stdout.split("\n")) {
		const s = line.trim();
		if (!s) continue;
		let ev: Record<string, unknown> | undefined;
		try {
			ev = asRecord(JSON.parse(s));
		} catch {
			continue; // ignore non-JSON noise
		}
		if (!ev) continue;
		if (ev.type === "error") {
			error = error ?? strOf(ev.message) ?? strOf(ev.error) ?? "error";
			continue;
		}
		if (ev.type === "message_end") {
			const m = asRecord(ev.message);
			if (m && m.role === "assistant") {
				const t = messageText(m);
				if (t) text = t; // keep the latest assistant text
				const stop = strOf(m.stopReason);
				if (stop === "error" || stop === "aborted") {
					error = error ?? strOf(m.errorMessage) ?? stop;
				}
			}
		}
	}
	return { text, error };
}
