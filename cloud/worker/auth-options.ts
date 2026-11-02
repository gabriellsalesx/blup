import type { BetterAuthOptions } from "better-auth";
import { APIError } from "better-auth/api";
import { drizzleAdapter } from "better-auth/adapters/drizzle";
import { emailOTP, captcha, deviceAuthorization, bearer } from "better-auth/plugins";
import { passkey } from "@better-auth/passkey";
import { stripe } from "@better-auth/stripe";
import { createStripe } from "./stripe-client";
import { PLANS } from "./plans";
import { v7 as uuidv7 } from "uuid";

// Blup Cloud's sign in: only a code by e-mail (which also creates the
// account) and passkeys, which a signed in person can add. Blup Studio
// signs in through the browser (device authorization) and then sends its
// session token as a Bearer header. Shared by the
// Worker (auth.ts) and by the Better Auth CLI that generates the schema
// (auth.cli.ts), so neither imports the other's runtime.

// The only client allowed to sign in through the browser.
const STUDIO_CLIENT_ID = "blup-studio";

type AuthConfig = {
	appUrl: string;
	rpID: string;
	secret: string;
	turnstileSecret: string;
	stripeSecret: string;
	stripeWebhookSecret: string;
	isBanned: (userId: string) => Promise<boolean>;
	deviceMark?: (userId:string,body:unknown)=>Promise<boolean>;
	finishDevice?: (userId:string,body:unknown)=>Promise<void>;
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
	const stripeClient = createStripe(cfg.stripeSecret);
	return {
		appName: "Blup Cloud",
		// SDK errors may include customer fields or SQL bindings.
		logger:{level:"error",log:(_level:unknown,_message:unknown,..._args:unknown[])=>{console.error("[blup] auth_error");}},
		baseURL: cfg.appUrl,
		basePath: "/api/auth",
		// The Worker owns signed webhook reconciliation. The plugin's snapshot
		// handler must never write an older event after that reconciliation.
		disabledPaths: ["/stripe/webhook","/subscription/success"],
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
			deleteUser: { enabled: false },
			additionalFields: {
				banned: { type: "boolean", defaultValue: false, input: false },
				banReason: { type: "string", required: false, input: false },
			},
		},
		session: {additionalFields:{studioDevice:{type:"boolean",defaultValue:false,input:false}}},
		databaseHooks: {
			session: {
				create: {
					before: async (session,ctx) => {
						if (await cfg.isBanned(session.userId)) throw new APIError("FORBIDDEN", { message: "Esta conta está bloqueada. Fale com o suporte.", code: "BANNED" });
						const device=ctx?.path==="/device/token";
						if(device && cfg.deviceMark && !await cfg.deviceMark(session.userId,ctx?.body))throw new APIError("FORBIDDEN",{code:"DEVICES_LIMIT",message:"Aprovação de aparelho inválida ou expirada."});
						return {data:{...session,studioDevice:device&&!!cfg.deviceMark}};
					},
					after: async(session,ctx)=>{if(ctx?.path==="/device/token")await cfg.finishDevice?.(session.userId,ctx.body);},
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
			// The Studio shows a code and opens /aparelho, where the signed in
			// person confirms it is the same code and approves.
			deviceAuthorization({
				verificationUri: `${cfg.appUrl}/aparelho`,
				expiresIn: "15m",
				validateClient: (clientId) => clientId === STUDIO_CLIENT_ID,
			}),
			bearer(),
			captcha({ provider: "cloudflare-turnstile", secretKey: cfg.turnstileSecret, endpoints: [SEND_CODE_PATH] }),
			stripe({
				stripeClient,
				stripeWebhookSecret: cfg.stripeWebhookSecret,
				// The Stripe customer is made at the first checkout, not at sign
				// up: free accounts never reach Stripe.
				createCustomerOnSignUp: false,
				subscription: {
					enabled: true,
					plans: PLANS.filter((p) => p.lookupKey).map((p) => ({ name: p.name, lookupKey: p.lookupKey, annualDiscountLookupKey: p.yearlyLookupKey, limits: p.limits })),
					// Stripe's recommendations: flexible billing mode, and a label to
					// follow this checkout flow in the Dashboard (8 random letters).
					getCheckoutSessionParams: async () => ({
						params: {
							locale: "pt-BR",
							subscription_data: { billing_mode: { type: "flexible" } },
							integration_identifier: "blup_cloud_plans_qmzrtxwa",
						},
					}),
				},
			}),
		],
	} satisfies BetterAuthOptions;
}
