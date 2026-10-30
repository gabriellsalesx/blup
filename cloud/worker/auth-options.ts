import type { BetterAuthOptions } from "better-auth";
import { drizzleAdapter } from "better-auth/adapters/drizzle";
import { emailOTP, captcha } from "better-auth/plugins";
import { passkey } from "@better-auth/passkey";
import { stripe } from "@better-auth/stripe";
import Stripe from "stripe";
import { PLANS } from "./plans";

// Blup Cloud's sign in: only a code by e-mail (which also creates the
// account) and passkeys, which a signed in person can add. Shared by the
// Worker (auth.ts) and by the Better Auth CLI that generates the schema
// (auth.cli.ts), so neither imports the other's runtime.

export type AuthConfig = {
	appUrl: string;
	rpID: string;
	secret: string;
	turnstileSecret: string;
	stripeSecret: string;
	stripeWebhookSecret: string;
	sendCode: (email: string, code: string, kind: string) => void;
};

// The OTP send route Turnstile guards (without the /api/auth base path).
export const SEND_CODE_PATH = "/email-otp/send-verification-otp";

export function authOptions(db: Parameters<typeof drizzleAdapter>[0], schema: Record<string, unknown>, cfg: AuthConfig) {
	return {
		appName: "Blup Cloud",
		baseURL: cfg.appUrl,
		basePath: "/api/auth",
		secret: cfg.secret,
		trustedOrigins: [cfg.appUrl],
		database: drizzleAdapter(db, { provider: "sqlite", schema }),
		// No passwords: codes and passkeys only.
		emailAndPassword: { enabled: false },
		plugins: [
			emailOTP({
				otpLength: 6,
				expiresIn: 600,
				allowedAttempts: 5,
				storeOTP: "hashed",
				// Not awaited: waiting would tell whether the e-mail exists.
				sendVerificationOTP: async ({ email, otp, type }) => cfg.sendCode(email, otp, type),
			}),
			passkey({ rpID: cfg.rpID, rpName: "Blup Cloud", origin: cfg.appUrl }),
			captcha({ provider: "cloudflare-turnstile", secretKey: cfg.turnstileSecret, endpoints: [SEND_CODE_PATH] }),
			stripe({
				// Fetch, not Node's http: the Worker's runtime.
				stripeClient: new Stripe(cfg.stripeSecret, { httpClient: Stripe.createFetchHttpClient() }),
				stripeWebhookSecret: cfg.stripeWebhookSecret,
				// The Stripe customer is made at the first checkout, not at sign
				// up: free accounts never reach Stripe.
				createCustomerOnSignUp: false,
				subscription: {
					enabled: true,
					plans: PLANS.filter((p) => p.lookupKey).map((p) => ({ name: p.name, lookupKey: p.lookupKey, limits: p.limits })),
					getCheckoutSessionParams: async () => ({ params: { locale: "pt-BR" } }),
				},
			}),
		],
	} satisfies BetterAuthOptions;
}
