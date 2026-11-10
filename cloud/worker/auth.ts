import { deviceMark,finishDevice } from "./devices";
import { env, waitUntil } from "cloudflare:workers";
import { betterAuth } from "better-auth";
import { drizzle } from "drizzle-orm/d1";

import { accountBlock, authOptions, reviewLogin } from "./auth-options";
import { sendCode } from "./email";
import * as schema from "./db/schema";

export const db = drizzle(env.DB, { schema });

export const auth = betterAuth(
	authOptions(db, schema, {
		appUrl: env.APP_URL,
		pulseUrl:env.PULSE_APP_URL,
		dealsUrl:env.DEALS_APP_URL,
		rpID: env.PASSKEY_RP_ID,
		secret: env.BETTER_AUTH_SECRET,
		turnstileSecret: env.TURNSTILE_SECRET_KEY,
		stripeSecret: env.STRIPE_SECRET_KEY,
		stripeWebhookSecret: env.STRIPE_WEBHOOK_SECRET,
		accountBlock:id=>accountBlock(env.DB,id),
		deviceMark:(id,body)=>deviceMark(env.DB,id,body),
		finishDevice:(id,body)=>finishDevice(env.DB,id,body),
		// The e-mail goes out after the answer.
		sendCode: (email, code, kind,language) => waitUntil(sendCode(env, email, code, kind,language)),
		review:reviewLogin(env.REVIEW_EMAIL,env.REVIEW_CODE),
	}),
);

export type Session = typeof auth.$Infer.Session;
