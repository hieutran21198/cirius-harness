/**
 * Parse the stdout of a headless `pi --mode json` run (ADR-0021): a stream of AgentSessionEvent
 * JSON objects, one per line. The final assistant result is the text of the last assistant
 * `message_end` event; a top-level `error` event or an assistant message that stopped on
 * error/aborted surfaces as an error. Split on "\n" by hand (never readline — U+2028/U+2029 are
 * valid inside JSON strings), matching the harness link's framing discipline.
 */

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
