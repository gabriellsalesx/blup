import type { Page } from "./pager.svelte.ts";
import type { Usage } from "./usage.ts";
import type { Limits } from "./api.ts";

// The staff area's calls (/api/staff). Cloudflare Access is in front in
// production; on this computer, STAFF_DEV_EMAIL lets you in.

export type Role = "support" | "admin";
export type StaffMe = { email: string; role: Role };
export type UserRow = { id: string; email: string | null; createdAt: string; banned: boolean; plan: string };
export type UserDetail = {
	user: { id: string; email: string | null; emailVerified: boolean; createdAt: string; banned: boolean; banReason: string | null };
	plan: { label: string; limits: Limits; status: string | null; periodEnd: string | null; cancelAtPeriodEnd: boolean };
	usage: Usage;
	billing: { id: number; type: string; amount: number | null; currency: string | null; createdAt: string }[];
	stripeUrl: string | null;
	sessions: { id: string; createdAt: string; expiresAt: string; ipAddress: string | null; userAgent: string | null }[];
	passkeys: { id: string; name: string | null; createdAt: string; deviceType: string }[];
	history: LogRow[];
};
export type LogRow = { id: number; staffEmail: string; action: string; userId: string | null; detail: string | null; createdAt: string; account?: { gone: boolean; email: string | null } | null };
export type Stats = { total: number; recent: number; banned: number; plans: { name: string; label: string; count: number }[] };
export type Team = { owner: string | null; staff: { email: string; role: Role; addedBy: string; createdAt: string }[] };

export class StaffError extends Error {
	constructor(public status: number, public code: string) {
		super(code);
	}
}

async function call<T>(method: string, path: string, body?: unknown): Promise<T> {
	const r = await fetch(`/api/staff${path}`, {
		method,
		credentials: "same-origin",
		headers: body ? { "content-type": "application/json" } : undefined,
		body: body ? JSON.stringify(body) : undefined,
	});
	const data = await r.json().catch(() => ({}));
	if (!r.ok) throw new StaffError(r.status, (data as { code?: string }).code ?? "ERROR");
	return data as T;
}

export const staffApi = {
	me: () => call<StaffMe>("GET", "/me"),
	stats: () => call<Stats>("GET", "/stats"),
	users: (q: string, cursor: string | null) => call<Page<UserRow>>("GET", `/users?q=${encodeURIComponent(q)}${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`),
	user: (id: string) => call<UserDetail>("GET", `/users/${encodeURIComponent(id)}`),
	revokeSessions: (id: string) => call<{ revoked: number }>("POST", `/users/${encodeURIComponent(id)}/sessions/revoke`),
	removePasskey: (id: string, pid: string) => call("DELETE", `/users/${encodeURIComponent(id)}/passkeys/${encodeURIComponent(pid)}`),
	ban: (id: string, reason: string) => call("POST", `/users/${encodeURIComponent(id)}/ban`, { reason }),
	unban: (id: string) => call("POST", `/users/${encodeURIComponent(id)}/unban`),
	remove: (id: string) => call("DELETE", `/users/${encodeURIComponent(id)}`),
	team: () => call<Team>("GET", "/team"),
	setMember: (email: string, role: Role) => call("POST", "/team", { email, role }),
	removeMember: (email: string) => call("DELETE", `/team/${encodeURIComponent(email)}`),
	log: (cursor: string | null) => call<Page<LogRow>>("GET", `/log${cursor ? `?cursor=${encodeURIComponent(cursor)}` : ""}`),
};

// A staff page's address: /staff/contas here, /contas on staff.getblup.io
// (hooks.ts maps it back).
export const staffHref = (path: string) =>
	typeof location !== "undefined" && location.hostname.startsWith("staff.") ? path.replace(/^\/staff/, "") || "/" : path;

export const roleLabel = (r: Role) => (r === "admin" ? "Admin" : "Suporte");

export const actionLabel: Record<string, string> = {
	"sessions.revoke": "Encerrou as sessões",
	"passkey.remove": "Removeu uma passkey",
	"user.ban": "Bloqueou a conta",
	"user.unban": "Desbloqueou a conta",
	"user.delete": "Excluiu a conta",
	"team.set": "Mudou a equipe",
	"team.remove": "Tirou da equipe",
};

// Stripe's money events, in the staff's words (and whether they need a look).
export const billingLabel: Record<string, { text: string; alert: boolean }> = {
	"invoice.paid": { text: "Pagamento recebido", alert: false },
	"invoice.payment_failed": { text: "Pagamento falhou", alert: true },
	"charge.refunded": { text: "Reembolso", alert: false },
	"charge.dispute.created": { text: "Contestação aberta", alert: true },
	"radar.early_fraud_warning.created": { text: "Alerta de fraude", alert: true },
};

export const money = (cents: number | null, currency: string | null) =>
	cents === null ? "" : (cents / 100).toLocaleString("pt-BR", { style: "currency", currency: (currency ?? "brl").toUpperCase() });

export const when = (d: string) => new Date(d).toLocaleString("pt-BR", { day: "2-digit", month: "short", year: "numeric", hour: "2-digit", minute: "2-digit" });
