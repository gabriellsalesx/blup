import {requestLanguage,type Language} from "../shared/language";
import {cloudPaths} from "../shared/panel-paths";
import type { BetterAuthOptions } from "better-auth";
import { APIError, createAuthMiddleware, getSessionFromCtx } from "better-auth/api";
import { z } from "zod";
import { drizzleAdapter } from "better-auth/adapters/drizzle";
import { emailOTP, captcha, deviceAuthorization, bearer } from "better-auth/plugins";
import { passkey } from "@better-auth/passkey";
import { stripe } from "@better-auth/stripe";
import { createStripe } from "./stripe-client";
import { PLANS } from "./plans";
import { v7 as uuidv7 } from "uuid";
import { DEVICE_HEADER, type DeviceGrant } from "./devices";

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
	pulseUrl?:string;
	dealsUrl?:string;
	rpID: string;
	secret: string;
	turnstileSecret: string;
	stripeSecret: string;
	stripeWebhookSecret: string;
	// Why the account can't sign in (null: it can).
	accountBlock: (userId: string) => Promise<AccountBlock | null>;
	// The device sign in (devices.ts): the approval still holding its place
	// (null: not any more) with the account's id of the device, and, once the
	// session is made, the approval released and the device's older sessions
	// removed (how many). rememberDevice keeps the device behind a new code.
	deviceMark?: (userId:string,body:unknown)=>Promise<DeviceGrant|null>;
	finishDevice?: (userId:string,body:unknown,session:{id:string;deviceId?:string|null})=>Promise<number>;
	rememberDevice?: (deviceCode:unknown,device:string|null,expiresIn:unknown)=>Promise<void>;
	sendCode: (email: string, code: string, kind: string,language:Language) => void;
	// The Google Play review account (see reviewLogin).
	review?: ReviewLogin|null;
};

// The store reviewers can't read a mailbox, and the e-mailed code expires,
// so one account (REVIEW_EMAIL, a secret) signs in with a fixed code
// (REVIEW_CODE, a secret). Only that e-mail and only to sign in: the code
// is not sent, and an e-mail change or any other account keeps the random
// code. Both unset (or malformed): no review account.
export type ReviewLogin={email:string;code:string};
export function reviewLogin(email:string|undefined,code:string|undefined):ReviewLogin|null{
	const e=email?.trim().toLowerCase(),c=code?.trim();
	return e&&c&&/^\d{6}$/.test(c)&&z.email().safeParse(e).success?{email:e,code:c}:null;
}
const isReview=(review:ReviewLogin|null|undefined,email:string)=>!!review&&email.trim().toLowerCase()===review.email;

// Blocked by the staff (BANNED) or deleted by its owner and still
// recoverable for 30 days (ACCOUNT_DELETED). Both stop a sign in only after
// the code or the passkey proved who it is, so neither tells whether an
// e-mail has an account.
export type AccountBlock = "BANNED" | "ACCOUNT_DELETED";
const blockRow = z.object({ banned: z.union([z.number(), z.boolean()]), deletedAt: z.number().nullable() });
export async function accountBlock(d1: D1Database, userId: string): Promise<AccountBlock | null> {
	const row = blockRow.safeParse(await d1.prepare("SELECT u.banned AS banned,q.deleted_at AS deletedAt FROM user u LEFT JOIN resource_quota q ON q.user_id=u.id WHERE u.id=?").bind(userId).first());
	// No account (or an unexpected row): refused, as before.
	if (!row.success || row.data.banned) return "BANNED";
	return row.data.deletedAt === null ? null : "ACCOUNT_DELETED";
}
const BLOCK_MESSAGE: Record<AccountBlock, string> = {
	BANNED: "Esta conta está bloqueada. Fale com o suporte.",
	ACCOUNT_DELETED: "Esta conta está em exclusão. Fale com o suporte para recuperar.",
};

// The OTP send route Turnstile guards (without the /api/auth base path).
export const SEND_CODE_PATH = "/email-otp/send-verification-otp";

// An account made with a passkey has no e-mail yet: Better Auth needs one,
// so it gets an address on a domain that can never receive mail (.invalid,
// RFC 2606) until the person links theirs.
export const NO_EMAIL_DOMAIN = "sem-email.blup.invalid";
export const hasEmail = (email: string) => !email.endsWith(`@${NO_EMAIL_DOMAIN}`);

// A new passkey is a lasting way into the account: only a recent sign in
// adds one (a stolen session could otherwise keep the account after every
// session is revoked), and never a Blup Studio token (the Studio has no
// use for it). The same window as deleting the account.
export const RECENT_SIGN_IN_MS = 3600_000;
const PASSKEY_REGISTRATION = new Set(["/passkey/generate-register-options", "/passkey/verify-registration"]);
// The Studio asks for a code with its client ID only: no account pre-bound
// (user_id) by whoever calls this public route.
const deviceCodeBody = z.strictObject({ client_id: z.string().min(1).max(64), scope: z.string().max(256).optional() });
// What /device/code answers (RFC 8628), as far as the device's mark needs.
const deviceCodeAnswer = z.object({ device_code: z.string().min(1).max(512), expires_in: z.number().int().positive() });
const sessionDevice = z.object({ deviceId: z.string().nullish() });
// The profile changes only the name (the panel's field allows 80).
const updateUserBody = z.strictObject({ name: z.string().trim().max(80) });

export const guardAuthRequest = createAuthMiddleware(async (ctx) => {
	if (ctx.path === "/device/code" && !deviceCodeBody.safeParse(ctx.body).success) throw new APIError("BAD_REQUEST", { error: "invalid_request", code: "INVALID" });
	if (ctx.path === "/update-user" && !updateUserBody.safeParse(ctx.body).success) throw new APIError("BAD_REQUEST", { code: "INVALID", message: "Pedido inválido." });
	if (!PASSKEY_REGISTRATION.has(ctx.path)) return;
	const signedIn = await getSessionFromCtx(ctx);
	if (!signedIn) return;
	const created = new Date(signedIn.session.createdAt).getTime();
	const studio = z.object({ studioDevice: z.union([z.literal(true), z.literal(1)]) }).safeParse(signedIn.session).success;
	if (studio || !Number.isFinite(created) || Date.now() - created > RECENT_SIGN_IN_MS) throw new APIError("FORBIDDEN", { code: "SESSION_EXPIRED", message: "Entre de novo para adicionar uma passkey." });
});

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
		// There are no passwords: the e-mail OTP plugin's password reset
		// routes would send codes to any account without Turnstile.
		// The subscription table also holds AppSumo's lifetime rows
		// (appsumo-<user>, no Stripe ids), which the plugin would read as its
		// own: never enable its upgrade/cancel/list/webhook paths, nor call
		// auth.api.upgradeSubscription, cancelSubscription or
		// listActiveSubscriptions on the server (billing-flow.ts does billing).
		disabledPaths: ["/stripe/webhook","/subscription/success","/subscription/upgrade","/subscription/billing-portal","/subscription/cancel","/subscription/restore","/subscription/list","/email-otp/request-password-reset","/forget-password/email-otp","/email-otp/reset-password"],
		secret: cfg.secret,
		trustedOrigins: [cfg.appUrl,...[cfg.pulseUrl,cfg.dealsUrl].filter((url):url is string=>!!url)],
		database: drizzleAdapter(db, { provider: "sqlite", schema }),
		// UUIDv7 (RFC 9562): time-ordered, so new rows land at the end of the
		// D1 index, and not guessable. The uuid package keeps a counter within
		// the millisecond (Workers freeze Date.now() during a request) and
		// draws the rest from crypto.getRandomValues.
		// The client's address is the one Cloudflare sets (X-Forwarded-For
		// would be whatever the client sent): sessions and Turnstile use it.
		// joins: the session and its user in one query (Drizzle's relational
		// query, from the relations in db/auth-schema.ts) instead of two
		// trips to D1 on every signed in request.
		advanced: { database: { generateId: () => uuidv7(), joins: true }, ipAddress: { ipAddressHeaders: ["cf-connecting-ip"] } },
		// No passwords: codes and passkeys only.
		emailAndPassword: { enabled: false },
		// A blocked (by the staff) or deleted account can't sign in.
		user: {
			// The person deletes the account from the panel (a recent sign in is
			// required). A paid subscription ends in Stripe first.
			deleteUser: { enabled: false },
			additionalFields: {
				banned: { type: "boolean", defaultValue: false, input: false },
				banReason: { type: "string", required: false, input: false },
				language:{type:"string",defaultValue:"pt",input:false},
			},
		},
		hooks: {
			before: guardAuthRequest,
			// A new sign-in code keeps the device that asked for it (the mark
			// Blup Studio and Blup Notes send): the code's response has the
			// device_code it is kept by.
			after: createAuthMiddleware(async (ctx) => {
				if (ctx.path !== "/device/code" || !cfg.rememberDevice) return;
				const returned = deviceCodeAnswer.safeParse(ctx.context.returned);
				if (returned.success) await cfg.rememberDevice(returned.data.device_code, ctx.headers?.get(DEVICE_HEADER) ?? null, returned.data.expires_in);
			}),
		},
		session: {additionalFields:{studioDevice:{type:"boolean",defaultValue:false,input:false},deviceId:{type:"string",required:false,input:false}}},
		databaseHooks: {
			user:{create:{before:async(user,ctx)=>({data:{...user,language:requestLanguage(ctx?.headers)}})}},
			session: {
				create: {
					before: async (session,ctx) => {
						const block = await cfg.accountBlock(session.userId);
						if (block) throw new APIError("FORBIDDEN", { message: BLOCK_MESSAGE[block], code: block });
						const device=ctx?.path==="/device/token";
						const grant=device&&cfg.deviceMark?await cfg.deviceMark(session.userId,ctx?.body):null;
						if(device && cfg.deviceMark && !grant)throw new APIError("FORBIDDEN",{code:"DEVICES_LIMIT",message:"Aprovação de aparelho inválida ou expirada."});
						return {data:{...session,studioDevice:device&&!!cfg.deviceMark,deviceId:grant?.deviceId??null}};
					},
					after: async(session,ctx)=>{if(ctx?.path==="/device/token")await cfg.finishDevice?.(session.userId,ctx.body,{id:session.id,deviceId:sessionDevice.safeParse(session).data?.deviceId??null});},
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
				// The review account's fixed code; undefined: a random one.
				generateOTP: ({email,type})=>type==="sign-in"&&isReview(cfg.review,email)?cfg.review!.code:undefined,
				// Not awaited: waiting would tell whether the e-mail exists.
				sendVerificationOTP: async ({ email, otp, type },ctx) => {if(type==="sign-in"&&isReview(cfg.review,email))return;cfg.sendCode(email, otp, type,requestLanguage(ctx?.headers));},
			}),
			passkey({
				rpID: cfg.rpID,
				rpName: "Blup Cloud",
				// Every panel signs in with the same passkeys (rpID is getblup.io).
				origin: [cfg.appUrl, ...[cfg.pulseUrl, cfg.dealsUrl].filter((url): url is string => !!url)],
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
			// The Studio shows a code and opens /device, where the signed in
			// person confirms it is the same code and approves.
			deviceAuthorization({
				verificationUri: `${cfg.appUrl}${cloudPaths.device}`,
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
				},
			}),
		],
	} satisfies BetterAuthOptions;
}
