// Blup Cloud's plans. Prices live in Stripe and are found by lookup key
// (scripts/stripe-setup.ts creates them), so no Stripe ID is kept here.
// A limit of -1 means no limit.

// live: the Studio sends each note as soon as the person stops typing and
// the other computers hear of it at once; without it, sync is by button.
// historyDays: how long old versions are kept (they count in storageMB).
export type Limits = { storageMB: number; sites: number; aiCredits: number; live: boolean; historyDays: number };

export type Plan = {
	name: string; // as Better Auth stores it (lowercase)
	label: string;
	priceCents: number; // in BRL, per month
	lookupKey?: string; // the Stripe price (paid plans only)
	limits: Limits;
	perks: string[];
};

export const FREE: Plan = {
	name: "free",
	label: "Grátis",
	priceCents: 0,
	limits: { storageMB: 100, sites: 1, aiCredits: 0, live: false, historyDays: 0 },
	perks: ["Notas sincronizadas até 100 MB, pelo botão Sincronizar", "1 site", "Criptografia de ponta a ponta"],
};

export const PLANS: Plan[] = [
	FREE,
	{
		name: "pro",
		label: "Pro",
		priceCents: 1900,
		lookupKey: "blup_pro_monthly",
		limits: { storageMB: 5120, sites: -1, aiCredits: 50, live: true, historyDays: 30 },
		perks: ["Notas sincronizadas até 5 GB, em tempo real", "Histórico de versões de 1 mês", "Sites sem limite", "Publicação automática e domínio próprio", "50 créditos de IA por mês (narração e tradução)"],
	},
];

export const planNamed = (name: string | null | undefined) => PLANS.find((p) => p.name === name) ?? FREE;
