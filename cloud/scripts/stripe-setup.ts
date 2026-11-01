// Creates in Stripe the product and the monthly and yearly prices of each paid plan
// (worker/plans.ts), found again by lookup key. Run it once per Stripe
// mode (test, then live); running it again changes nothing.
//
//   STRIPE_SECRET_KEY=sk_test_... node scripts/stripe-setup.ts
//
// Without the variable, the key comes from .dev.vars (never from git).
import { readFileSync } from "node:fs";
import Stripe from "stripe";
import { PLANS } from "../worker/plans.ts";

function key(): string {
	if (process.env.STRIPE_SECRET_KEY) return process.env.STRIPE_SECRET_KEY;
	try {
		const line = readFileSync(new URL("../.dev.vars", import.meta.url), "utf8").split("\n").find((l) => l.startsWith("STRIPE_SECRET_KEY="));
		return line?.slice("STRIPE_SECRET_KEY=".length).trim() ?? "";
	} catch {
		return "";
	}
}

const secret = key();
if (!secret.startsWith("sk_") || secret.includes("coloque")) {
	console.error("Falta a chave do Stripe: STRIPE_SECRET_KEY no ambiente ou em .dev.vars.");
	process.exit(1);
}
const stripe = new Stripe(secret);
console.log(secret.startsWith("sk_live_") ? "Stripe em PRODUÇÃO" : "Stripe em modo de teste");

// One product per plan, with a monthly and a yearly price.
async function product(plan: (typeof PLANS)[number]) {
	const found = await stripe.products.search({ query: `metadata['blup_plan']:'${plan.name}' AND active:'true'` });
	return found.data[0] ?? (await stripe.products.create({ name: `Blup ${plan.label}`, metadata: { blup_plan: plan.name } }));
}

for (const plan of PLANS) {
	if (!plan.lookupKey || !plan.yearlyLookupKey) continue;
	for (const [lookupKey, cents, interval] of [
		[plan.lookupKey, plan.priceCents, "month"],
		[plan.yearlyLookupKey, plan.yearlyCents, "year"],
	] as const) {
		const found = await stripe.prices.list({ lookup_keys: [lookupKey], active: true, limit: 1 });
		const price = found.data[0];
		if (price) {
			if (price.unit_amount !== cents || price.currency !== plan.currency || price.recurring?.interval !== interval) {
				console.warn(`${lookupKey}: o preço no Stripe (${price.unit_amount} ${price.currency}/${price.recurring?.interval}) é diferente do plano (${cents} ${plan.currency}/${interval}). Crie um preço novo no dashboard e passe o lookup key para ele.`);
			} else console.log(`${lookupKey}: já existe (${price.id})`);
			continue;
		}
		const created = await stripe.prices.create({
			product: (await product(plan)).id,
			currency: plan.currency,
			unit_amount: cents,
			recurring: { interval },
			lookup_key: lookupKey,
			metadata: { blup_plan: plan.name },
		});
		console.log(`${lookupKey}: criado (${created.id})`);
	}
}
