// Blup Cloud's plans. Prices live in Stripe and are found by lookup key
// (scripts/stripe-setup.ts checks them), so no Stripe ID is kept here.
// A limit of -1 means no limit.

// Every account can sync up to 50 folders of notes (vaults).
// storageMB: notes in the cloud, old versions included. maxFileMB: larger
// files are not synced (the Studio says so). sites: sites published and
// hosted by Blup (not the vaults).
// customDomain: the sites can have their own domain (else they live at
// name.getblup.site). devices: Blup Studio installs signed in at once.
// live: the Studio sends each note as soon as the person stops typing and
// the other computers hear of it at once; without it, sync is by button.
// historyDays: how long old versions are kept. aiUsage: how much AI use
// (narration, translation) per week, as a multiple of Pro's (0 = none).
// reminderEmails: the calendar's reminders also sent by e-mail, per
// month (0 = none; on the computer they are always free).
export type Limits = {
	storageMB: number;
	maxFileMB: number;
	sites: number;
	customDomain: boolean;
	devices: number;
	live: boolean;
	historyDays: number;
	aiUsage: number;
	reminderEmails: number;
};

export type Plan = {
	name: "free" | "pro" | "max" | "ultra"; // as Better Auth stores it
	label: string;
	lookupKey?: string; // the Stripe monthly price (paid plans only)
	yearlyLookupKey?: string; // the Stripe yearly price
	limits: Limits;
	perks: string[];
};

const FREE: Plan = {
	name: "free",
	label: "Grátis",

	limits: { storageMB: 50, maxFileMB: 10, sites: 0, customDomain: false, devices: 1, live: false, historyDays: 0, aiUsage: 0, reminderEmails: 0 },
	perks: ["50 MB de notas, sincronizadas pelo botão", "1 computador conectado", "Arquivos de até 10 MB", "Criptografia de ponta a ponta"],
};

export const PLANS: Plan[] = [
	FREE,
	{
		name: "pro",
		label: "Pro",

		lookupKey: "blup_pro_mensal",
		yearlyLookupKey: "blup_pro_anual",
		limits: { storageMB: 1024, maxFileMB: 25, sites: 1, customDomain: false, devices: 2, live: true, historyDays: 30, aiUsage: 1, reminderEmails: 0 },
		perks: ["1 GB de notas, sincronizadas em tempo real", "2 computadores conectados", "Arquivos de até 25 MB", "Histórico de versões de 1 mês", "1 site no ar pelo Blup, no endereço seusite.getblup.site"],
	},
	{
		name: "max",
		label: "Max",

		lookupKey: "blup_max_mensal",
		yearlyLookupKey: "blup_max_anual",
		limits: { storageMB: 5120, maxFileMB: 50, sites: 3, customDomain: true, devices: 3, live: true, historyDays: 365, aiUsage: 2, reminderEmails: 50 },
		perks: ["5 GB de notas, sincronizadas em tempo real", "3 computadores conectados", "Arquivos de até 50 MB", "Histórico de versões de 12 meses", "3 sites no ar pelo Blup, com domínio próprio", "50 avisos da agenda por e-mail por mês"],
	},
	{
		name: "ultra",
		label: "Ultra",

		lookupKey: "blup_ultra_mensal",
		yearlyLookupKey: "blup_ultra_anual",
		limits: { storageMB: 10240, maxFileMB: 100, sites: 10, customDomain: true, devices: -1, live: true, historyDays: 365, aiUsage: 5, reminderEmails: 100 },
		perks: ["10 GB de notas, sincronizadas em tempo real", "Computadores sem limite", "Arquivos de até 100 MB", "Histórico de versões de 12 meses", "10 sites no ar pelo Blup, com domínio próprio", "100 avisos da agenda por e-mail por mês", "Suporte prioritário e novidades antes"],
	},
];

// The plan the panel suggests (the middle one).
export const SUGGESTED = "max";

export const planNamed = (name: string | null | undefined) => PLANS.find((p) => p.name === name) ?? FREE;

// The owner of Blup (STAFF_OWNER_EMAIL) has GIFT_PLAN for free. Only a
// verified e-mail equal to the owner's counts; no owner set, no gift.
export const GIFT_PLAN = "ultra";
export function ownerMatches(owner: string | null | undefined, email: string | null | undefined, verified: boolean | null | undefined) {
	const o = owner?.trim().toLowerCase();
	return !!o && !!verified && email?.trim().toLowerCase() === o;
}
