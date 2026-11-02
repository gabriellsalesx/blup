import { accountAllowed,accountPlan,observePlan } from "./lifecycle";
import { deviceMark,finishDevice } from "./devices";
import { env, waitUntil } from "cloudflare:workers";
import { betterAuth } from "better-auth";
import { drizzle } from "drizzle-orm/d1";

import { authOptions } from "./auth-options";
import { sendCode } from "./email";
import { recordBillingEvent } from "./billing";
import * as schema from "./db/schema";

export const db = drizzle(env.DB, { schema });

export const auth = betterAuth(
	authOptions(db, schema, {
		appUrl: env.APP_URL,
		rpID: env.PASSKEY_RP_ID,
		secret: env.BETTER_AUTH_SECRET,
		turnstileSecret: env.TURNSTILE_SECRET_KEY,
		stripeSecret: env.STRIPE_SECRET_KEY,
		stripeWebhookSecret: env.STRIPE_WEBHOOK_SECRET,
		onStripeEvent: (event, stripe) => recordBillingEvent(event, stripe),
		isBanned:async id=>!await accountAllowed(env.DB,id),
		deviceMark:(id,body)=>deviceMark(env.DB,id,body),
		finishDevice:(id,body)=>finishDevice(env.DB,id,body),
		planChanged:async id=>{const plan=await accountPlan(env,id);if(plan)await observePlan(env.DB,id,plan.name);},
		// The e-mail goes out after the answer.
		sendCode: (email, code, kind) => waitUntil(sendCode(env, email, code, kind)),
	}),
);

export type Session = typeof auth.$Infer.Session;
