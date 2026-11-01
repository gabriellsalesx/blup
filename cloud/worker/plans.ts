// Blup Cloud's plans. Prices live in Stripe and are found by lookup key
// (scripts/stripe-setup.ts creates them), so no Stripe ID is kept here.
// A limit of -1 means no limit.

// Every plan syncs any number of folders of notes (vaults).
// storageMB: notes in the cloud, old versions included. maxFileMB: larger
// files are not synced (the Studio says so). sites: sites published and
// hosted by Blup (not the vaults).
// customDomain: the sites can have their own domain (else they live at
// name.getblup.site). devices: Blup Studio installs signed in at once.
// live: the Studio sends each note as soon as the person stops typing and
// the other computers hear of it at once; without it, sync is by button.
// historyDays: how long old versions are kept. aiUsage: how much AI use
// (narration, translation) per week, as a multiple of Pro's (0 = none).
export type Limits = {
	storageMB: number;
	maxFileMB: number;
	sites: number;
	customDomain: boolean;
	devices: number;
	live: boolean;
	historyDays: number;
	aiUsage: number;
};

export type Plan = {
	name: string; // as Better Auth stores it (lowercase)
	label: string;
	// In reais: Stripe's Adaptive Pricing shows and charges it in the
	// customer's currency abroad (it only converts from the account's own
	// currency). The yearly price is ten months: two months free.
	priceCents: number; // per month
	yearlyCents: number; // per year
	currency: "brl";
	lookupKey?: string; // the Stripe monthly price (paid plans only)
	yearlyLookupKey?: string; // the Stripe yearly price
	limits: Limits;
	perks: string[];
};

export const FREE: Plan = {
	name: "free",
	label: "Grátis",
	priceCents: 0,
	yearlyCents: 0,
	currency: "brl",
	limits: { storageMB: 50, maxFileMB: 10, sites: 0, customDomain: false, devices: 1, live: false, historyDays: 0, aiUsage: 0 },
	perks: ["50 MB de notas, sincronizadas pelo botão", "1 computador conectado", "Arquivos de até 10 MB", "Criptografia de ponta a ponta"],
};

export const PLANS: Plan[] = [
	FREE,
	{
		name: "pro",
		label: "Pro",
		priceCents: 1490,
		yearlyCents: 14900,
		currency: "brl",
		lookupKey: "blup_pro_mensal",
		yearlyLookupKey: "blup_pro_anual",
		limits: { storageMB: 1024, maxFileMB: 25, sites: 1, customDomain: false, devices: 2, live: true, historyDays: 30, aiUsage: 1 },
		perks: ["1 GB de notas, sincronizadas em tempo real", "2 computadores conectados", "Arquivos de até 25 MB", "Histórico de versões de 1 mês", "1 site no ar pelo Blup, no endereço seusite.getblup.site"],
	},
	{
		name: "max",
		label: "Max",
		priceCents: 2990,
		yearlyCents: 29900,
		currency: "brl",
		lookupKey: "blup_max_mensal",
		yearlyLookupKey: "blup_max_anual",
		limits: { storageMB: 10240, maxFileMB: 200, sites: 5, customDomain: true, devices: 3, live: true, historyDays: 365, aiUsage: 2 },
		perks: ["10 GB de notas, sincronizadas em tempo real", "3 computadores conectados", "Arquivos de até 200 MB", "Histórico de versões de 12 meses", "5 sites no ar pelo Blup, com domínio próprio"],
	},
	{
		name: "ultra",
		label: "Ultra",
		priceCents: 5990,
		yearlyCents: 59900,
		currency: "brl",
		lookupKey: "blup_ultra_mensal",
		yearlyLookupKey: "blup_ultra_anual",
		limits: { storageMB: 51200, maxFileMB: 500, sites: -1, customDomain: true, devices: -1, live: true, historyDays: 365, aiUsage: 5 },
		perks: ["50 GB de notas, sincronizadas em tempo real", "Computadores sem limite", "Arquivos de até 500 MB", "Histórico de versões de 12 meses", "Sites sem limite no ar pelo Blup, com domínio próprio", "Suporte prioritário e novidades antes"],
	},
];

// The plan the panel suggests (the middle one).
export const SUGGESTED = "max";

export const planNamed = (name: string | null | undefined) => PLANS.find((p) => p.name === name) ?? FREE;
