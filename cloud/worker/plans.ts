// Blup Cloud's plans. Prices live in Stripe and are found by lookup key
// (scripts/stripe-setup.ts creates them), so no Stripe ID is kept here.
// A limit of -1 means no limit.

// Every plan syncs any number of folders of notes (vaults).
// storageMB: notes in the cloud, old versions included. maxFileMB: larger
// files are not synced (the Studio says so). sites: sites published and
// hosted by Blup (not the vaults).
// devices: Blup Studio installs signed in at once.
// live: the Studio sends each note as soon as the person stops typing and
// the other computers hear of it at once; without it, sync is by button.
// historyDays: how long old versions are kept.
export type Limits = {
	storageMB: number;
	maxFileMB: number;
	sites: number;
	devices: number;
	live: boolean;
	historyDays: number;
};

export type Plan = {
	name: string; // as Better Auth stores it (lowercase)
	label: string;
	// Per month, in US dollars: Stripe's Adaptive Pricing shows and charges
	// it in the customer's currency.
	priceCents: number;
	currency: "usd";
	lookupKey?: string; // the Stripe price (paid plans only)
	limits: Limits;
	perks: string[];
};

export const FREE: Plan = {
	name: "free",
	label: "Grátis",
	priceCents: 0,
	currency: "usd",
	limits: { storageMB: 50, maxFileMB: 10, sites: 0, devices: 1, live: false, historyDays: 0 },
	perks: ["50 MB de notas, sincronizadas pelo botão", "1 computador conectado", "Arquivos de até 10 MB", "Criptografia de ponta a ponta"],
};

export const PLANS: Plan[] = [
	FREE,
	{
		name: "pro",
		label: "Pro",
		priceCents: 500,
		currency: "usd",
		lookupKey: "blup_pro_monthly_usd",
		limits: { storageMB: 1024, maxFileMB: 25, sites: 1, devices: 2, live: true, historyDays: 30 },
		perks: ["1 GB de notas, sincronizadas em tempo real", "2 computadores conectados", "Arquivos de até 25 MB", "Histórico de versões de 1 mês", "1 site no ar pelo Blup, com domínio próprio"],
	},
	{
		name: "max",
		label: "Max",
		priceCents: 2000,
		currency: "usd",
		lookupKey: "blup_max_monthly",
		limits: { storageMB: 5120, maxFileMB: 200, sites: -1, devices: -1, live: true, historyDays: 365 },
		perks: ["5 GB de notas, sincronizadas em tempo real", "Computadores sem limite", "Arquivos de até 200 MB", "Histórico de versões de 12 meses", "Sites sem limite no ar pelo Blup, com domínio próprio"],
	},
];

export const planNamed = (name: string | null | undefined) => PLANS.find((p) => p.name === name) ?? FREE;
