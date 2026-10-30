// Creates in Stripe the product and the monthly price of each paid plan
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

for (const plan of PLANS) {
	if (!plan.lookupKey) continue;
	const found = await stripe.prices.list({ lookup_keys: [plan.lookupKey], active: true, limit: 1 });
	const price = found.data[0];
	if (price) {
		if (price.unit_amount !== plan.priceCents || price.currency !== "brl") {
			console.warn(`${plan.name}: o preço no Stripe (${price.unit_amount} ${price.currency}) é diferente do plano (${plan.priceCents} brl). Crie um preço novo no dashboard e passe o lookup key para ele.`);
		} else console.log(`${plan.name}: já existe (${price.id})`);
		continue;
	}
	const product = await stripe.products.create({ name: `Blup ${plan.label}`, metadata: { blup_plan: plan.name } });
	const created = await stripe.prices.create({
		product: product.id,
		currency: "brl",
		unit_amount: plan.priceCents,
		recurring: { interval: "month" },
		lookup_key: plan.lookupKey,
		metadata: { blup_plan: plan.name },
	});
	console.log(`${plan.name}: criado (${created.id})`);
}
