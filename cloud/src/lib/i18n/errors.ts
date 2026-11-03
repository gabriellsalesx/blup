import {z} from "zod";
import {t} from "../i18n.svelte";
import type {MessageKey} from "./catalog";
const codes={BILLING_PAYMENT_REQUIRED:"errorPayment",BILLING_QUOTE_EXPIRED:"billingQuoteExpired",BILLING_PENDING:"billingPendingError",BILLING_CHECKOUT_PENDING:"billingPendingError",BILLING_USE_PREVIEW:"billingPendingError",BILLING_UNSUPPORTED:"billingUnsupported",BILLING_PORTAL_UNSAFE:"billingUnsupported",BILLING_NO_CHANGE:"billingNoChange",BILLING_SELECTION_REQUIRED:"billingKeep",BILLING_BUSY:"errorBusy",DOMAIN_EXISTS:"pulseDomainError",DOMAIN_TAKEN:"pulseDomainError",DOMAIN_UNAVAILABLE:"pulseDomainError",DOMAIN_PROVIDER:"pulseDomainError",CHANNEL_UNAVAILABLE:"pulseChannelUnavailable",LINK_EXPIRED:"pulseLinkExpired",INVALID_TARGET:"pulseInvalidTarget",SLUG_TAKEN:"errorInvalid",INVALID:"errorInvalid",BAD_REQUEST:"errorInvalid",INVALID_OTP:"errorOtp",OTP_EXPIRED:"errorOtpExpired",TOO_MANY_ATTEMPTS:"errorAttempts",TOO_MANY_REQUESTS:"errorRate",EMAIL_REQUIRED:"errorEmailRequired",MISSING_RESPONSE:"errorCaptcha",VERIFICATION_FAILED:"errorCaptcha",PRICING_UNAVAILABLE:"pricingUnavailable",UNAUTHORIZED:"errorSession",SESSION_EXPIRED:"errorSession",FORBIDDEN:"errorForbidden",BANNED:"errorForbidden",ORIGIN_NOT_ALLOWED:"errorForbidden",NOT_FOUND:"errorNotFound",QUOTA:"errorQuota",TOO_LARGE:"errorTooLarge",PLAN:"errorPlan",SITES_LIMIT:"errorSiteLimit",SELECTION_CLOSED:"errorSelectionClosed",SYNC_BUSY:"errorBusy",DEPLOY_BUSY:"errorBusy",EMAIL_CHANGE_SUPPORT:"errorEmailSupport",INVALID_EMAIL:"errorInvalid",USER_ALREADY_EXISTS:"errorInvalid",INVALID_EMAIL_OR_PASSWORD:"errorInvalid",PASSKEY_NOT_FOUND:"errorNotFound",AUTH_CANCELLED:"errorGeneric"} as const satisfies Record<string,MessageKey>;
// Object.hasOwn proves the key belongs to this closed, typed table.
const known=(code:string):code is keyof typeof codes=>Object.hasOwn(codes,code);
const errorSchema=z.object({code:z.string().optional(),statusCode:z.number().optional(),detail:z.object({data:z.object({code:z.string().optional()})}).optional()});
export function errorMessage(error:unknown):string {
 const parsed=errorSchema.safeParse(error),code=parsed.success?parsed.data.code??parsed.data.detail?.data.code:undefined;
 if(code&&known(code))return t(codes[code]);
 if(code&&/CUSTOMER|SUBSCRIPTION|CHECKOUT|BILLING/.test(code))return t("errorPayment");
 return t("errorGeneric");
}
