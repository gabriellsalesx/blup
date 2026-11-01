// The panel's calls to the Worker's own API (/api), besides Better Auth's.

import type { Usage } from "./usage.ts";

export type Limits = { storageMB: number; maxFileMB: number; sites: number; customDomain: boolean; devices: number; live: boolean; historyDays: number; aiUsage: number };
export type PlanInfo = { name: string; label: string; priceCents: number; yearlyCents: number; currency: "brl"; limits: Limits; perks: string[]; suggested: boolean };
export type Me = {
	id: string;
	email: string | null; // null: an account made with a passkey, no e-mail linked yet
	name: string;
	createdAt: string;
	usage: Usage;
	// AI through Blup Cloud: the share of this month's allowance used.
	ai: AIShare;
	plan: { name: string; label: string; limits: Limits; gift?: boolean; status: string | null; renewsAt: string | null; cancelsAtPeriodEnd: boolean };
};

async function get<T>(path: string): Promise<T> {
	const r = await fetch(`/api${path}`, { credentials: "same-origin" });
	if (!r.ok) throw new Error(`${path}: ${r.status}`);
	return r.json() as Promise<T>;
}

export type AIShare = { percent: number; resetsAt: string; included: boolean };

export type Device = { id: string; name: string; createdAt: string; lastSeen: string };

export const api = {
	me: () => get<Me>("/me"),
	plans: () => get<PlanInfo[]>("/plans"),
	devices: () => get<{ devices: Device[] }>("/devices"),
	disconnect: async (id: string) => {
		const r = await fetch(`/api/devices/${encodeURIComponent(id)}`, { method: "DELETE", credentials: "same-origin" });
		if (!r.ok) throw new Error(`disconnect: ${r.status}`);
	},
};

export const money = (cents: number, currency: string = "brl") => (cents / 100).toLocaleString("pt-BR", { style: "currency", currency: currency.toUpperCase() });
