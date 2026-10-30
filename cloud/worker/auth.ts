import { env, waitUntil } from "cloudflare:workers";
import { betterAuth } from "better-auth";
import { drizzle } from "drizzle-orm/d1";
import { authOptions } from "./auth-options";
import { sendCode } from "./email";
import * as schema from "./db/schema";

export const db = drizzle(env.DB, { schema });

export const auth = betterAuth(
	authOptions(db, schema, {
		appUrl: env.APP_URL,
		rpID: env.PASSKEY_RP_ID,
		secret: env.BETTER_AUTH_SECRET,
		turnstileSecret: env.TURNSTILE_SECRET_KEY,
		// The e-mail goes out after the answer.
		sendCode: (email, code, kind) => waitUntil(sendCode(env, email, code, kind)),
	}),
);

export type Session = typeof auth.$Infer.Session;
