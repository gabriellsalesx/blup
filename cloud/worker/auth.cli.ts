// Only for the Better Auth CLI (npm run db:auth-schema): the same options
// with placeholders, without Cloudflare's runtime.
import { betterAuth } from "better-auth";
import { drizzle } from "drizzle-orm/d1";
import { authOptions } from "./auth-options";

export const auth = betterAuth(
	authOptions(drizzle({} as D1Database), {}, { appUrl: "http://localhost", rpID: "localhost", secret: "cli", turnstileSecret: "cli", stripeSecret: "sk_test_cli", stripeWebhookSecret: "whsec_cli", isBanned: async () => false, sendCode: () => {} }),
);
