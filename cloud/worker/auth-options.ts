import type { BetterAuthOptions } from "better-auth";
import { APIError } from "better-auth/api";
import { drizzleAdapter } from "better-auth/adapters/drizzle";
import { emailOTP, captcha } from "better-auth/plugins";
import { passkey } from "@better-auth/passkey";
import { stripe } from "@better-auth/stripe";
import Stripe from "stripe";
import { PLANS } from "./plans";
import { v7 as uuidv7 } from "uuid";

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
	isBanned: (userId: string) => Promise<boolean>;
	sendCode: (email: string, code: string, kind: string) => void;
};

// The OTP send route Turnstile guards (without the /api/auth base path).
export const SEND_CODE_PATH = "/email-otp/send-verification-otp";

// An account made with a passkey has no e-mail yet: Better Auth needs one,
// so it gets an address on a domain that can never receive mail (.invalid,
// RFC 2606) until the person links theirs.
export const NO_EMAIL_DOMAIN = "sem-email.blup.invalid";
export const hasEmail = (email: string) => !email.endsWith(`@${NO_EMAIL_DOMAIN}`);

export function authOptions(db: Parameters<typeof drizzleAdapter>[0], schema: Record<string, unknown>, cfg: AuthConfig) {
	// Fetch, not Node's http: the Worker's runtime.
	const stripeClient = new Stripe(cfg.stripeSecret, { httpClient: Stripe.createFetchHttpClient() });
	return {
		appName: "Blup Cloud",
		baseURL: cfg.appUrl,
		basePath: "/api/auth",
		secret: cfg.secret,
		trustedOrigins: [cfg.appUrl],
		database: drizzleAdapter(db, { provider: "sqlite", schema }),
		// UUIDv7 (RFC 9562): time-ordered, so new rows land at the end of the
		// D1 index, and not guessable. The uuid package keeps a counter within
		// the millisecond (Workers freeze Date.now() during a request) and
		// draws the rest from crypto.getRandomValues.
		advanced: { database: { generateId: () => uuidv7() } },
		// No passwords: codes and passkeys only.
		emailAndPassword: { enabled: false },
		// A blocked account (by the staff) can't sign in.
		user: {
			// The person deletes the account from the panel (a recent sign in is
			// required). A paid subscription ends in Stripe first.
			deleteUser: {
				enabled: true,
				beforeDelete: async (user) => {
					const customer = (user as { stripeCustomerId?: string | null }).stripeCustomerId;
					if (!customer) return;
					for await (const sub of stripeClient.subscriptions.list({ customer, status: "all" })) {
						if (!["canceled", "incomplete_expired"].includes(sub.status)) await stripeClient.subscriptions.cancel(sub.id);
					}
				},
			},
			additionalFields: {
				banned: { type: "boolean", defaultValue: false, input: false },
				banReason: { type: "string", required: false, input: false },
			},
		},
		databaseHooks: {
			session: {
				create: {
					before: async (session) => {
						if (await cfg.isBanned(session.userId)) throw new APIError("FORBIDDEN", { message: "Esta conta está bloqueada. Fale com o suporte.", code: "BANNED" });
					},
				},
			},
		},
		plugins: [
			emailOTP({
				otpLength: 6,
				expiresIn: 600,
				allowedAttempts: 5,
				storeOTP: "hashed",
				// Linking (or changing) the e-mail: a code goes to the new address.
				changeEmail: { enabled: true },
				// Not awaited: waiting would tell whether the e-mail exists.
				sendVerificationOTP: async ({ email, otp, type }) => cfg.sendCode(email, otp, type),
			}),
			passkey({
				rpID: cfg.rpID,
				rpName: "Blup Cloud",
				origin: cfg.appUrl,
				// Passkey-first: an account can start with a passkey alone. The
				// user is created only after the WebAuthn ceremony succeeds.
				registration: {
					requireSession: false,
					resolveUser: async () => ({ id: uuidv7(), name: "Blup Cloud" }),
					afterVerification: async ({ ctx, user }) => {
						// Signed in: the passkey joins that account.
						if (await ctx.context.internalAdapter.findUserById(user.id)) return;
						const created = await ctx.context.internalAdapter.createUser({ email: `${uuidv7()}@${NO_EMAIL_DOMAIN}`, name: "", emailVerified: false }, { method: "passkey" });
						return { userId: created.id };
					},
				},
			}),
			captcha({ provider: "cloudflare-turnstile", secretKey: cfg.turnstileSecret, endpoints: [SEND_CODE_PATH] }),
			stripe({
				stripeClient,
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
