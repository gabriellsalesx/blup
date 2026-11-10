import {z} from "zod/mini";
import {t} from "../i18n.svelte";
import type {MessageKey,PlainMessageKey} from "./catalog";
const codes={BILLING_PAYMENT_REQUIRED:"errorPayment",BILLING_QUOTE_EXPIRED:"billingQuoteExpired",BILLING_PENDING:"billingPendingError",BILLING_CHECKOUT_PENDING:"billingPendingError",BILLING_USE_PREVIEW:"billingPendingError",BILLING_UNSUPPORTED:"billingUnsupported",BILLING_PORTAL_UNSAFE:"billingUnsupported",BILLING_NO_CHANGE:"billingNoChange",BILLING_SELECTION_REQUIRED:"billingKeep",BILLING_BUSY:"errorBusy",DOMAIN_EXISTS:"pulseDomainError",DOMAIN_TAKEN:"pulseDomainError",DOMAIN_UNAVAILABLE:"pulseDomainError",DOMAIN_PROVIDER:"pulseDomainError",CHANNEL_UNAVAILABLE:"pulseChannelUnavailable",NO_CHANNELS:"pulseNoChannelsError",LINK_EXPIRED:"pulseLinkExpired",INVALID_TARGET:"pulseInvalidTarget",SLUG_TAKEN:"errorInvalid",INVALID:"errorInvalid",BAD_REQUEST:"errorInvalid",INVALID_OTP:"errorOtp",OTP_EXPIRED:"errorOtpExpired",TOO_MANY_ATTEMPTS:"errorAttempts",TOO_MANY_REQUESTS:"errorRate",EMAIL_REQUIRED:"errorEmailRequired",MISSING_RESPONSE:"errorCaptcha",VERIFICATION_FAILED:"errorCaptcha",PRICING_UNAVAILABLE:"pricingUnavailable",UNAUTHORIZED:"errorSession",SESSION_EXPIRED:"errorSession",FORBIDDEN:"errorForbidden",BANNED:"errorBanned",ACCOUNT_DELETED:"errorAccountDeleted",ORIGIN_NOT_ALLOWED:"errorForbidden",NOT_FOUND:"errorNotFound",QUOTA:"errorQuota",TOO_LARGE:"errorTooLarge",PLAN:"errorPlan",SITES_LIMIT:"errorSiteLimit",SELECTION_CLOSED:"errorSelectionClosed",SYNC_BUSY:"errorBusy",DEPLOY_BUSY:"errorBusy",EMAIL_CHANGE_SUPPORT:"errorEmailSupport",INVALID_EMAIL:"errorInvalid",USER_ALREADY_EXISTS:"errorInvalid",INVALID_EMAIL_OR_PASSWORD:"errorInvalid",PASSKEY_NOT_FOUND:"errorNotFound",AUTH_CANCELLED:"errorGeneric",INVALID_CODE:"redeemInvalid",ALREADY_REDEEMED:"redeemAlready",MAX_LEVEL:"redeemMaxLevel",UNAVAILABLE:"errorUnavailable",LIFETIME_COVERS:"lifetimeCovers"} as const satisfies Record<string,MessageKey>;
// Object.hasOwn proves the key belongs to this closed, typed table.
const known=(code:string):code is keyof typeof codes=>Object.hasOwn(codes,code);
const errorSchema=z.object({code:z.optional(z.string()),statusCode:z.optional(z.number()),detail:z.optional(z.object({data:z.object({code:z.optional(z.string())})}))});
// `own`: a page's words for some codes (the redeem page's INVALID is a
// code, not a form).
export function errorMessage(error:unknown,own:Partial<Record<keyof typeof codes,PlainMessageKey>>={}):string {
 const parsed=errorSchema.safeParse(error),code=parsed.success?parsed.data.code??parsed.data.detail?.data.code:undefined;
 if(code&&known(code))return t(own[code]??codes[code]);
 if(code&&/CUSTOMER|SUBSCRIPTION|CHECKOUT|BILLING/.test(code))return t("errorPayment");
 return t("errorGeneric");
}
