// Blup Cloud's plans. Prices live in Stripe and are found by lookup key
// (scripts/stripe-setup.ts creates them), so no Stripe ID is kept here.
// A limit of -1 means no limit.

export type Limits = { storageMB: number; sites: number; aiCredits: number };

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
	limits: { storageMB: 100, sites: 1, aiCredits: 0 },
	perks: ["Notas sincronizadas até 100 MB", "1 site", "Criptografia de ponta a ponta"],
};

export const PLANS: Plan[] = [
	FREE,
	{
		name: "pro",
		label: "Pro",
		priceCents: 1900,
		lookupKey: "blup_pro_monthly",
		limits: { storageMB: 5120, sites: -1, aiCredits: 50 },
		perks: ["Notas sincronizadas até 5 GB", "Sites sem limite", "Publicação automática e domínio próprio", "50 créditos de IA por mês (narração e tradução)"],
	},
];

export const planNamed = (name: string | null | undefined) => PLANS.find((p) => p.name === name) ?? FREE;
