import {z} from "zod";
import {t} from "#lib/i18n.svelte.ts";
import {errorMessage} from "#lib/i18n/errors.ts";
import type {PlainMessageKey} from "#lib/i18n/catalog.ts";
// The Deals API answers errors with {code} as Pulse does. The codes whose
// meaning is Deals' own (a plan limit is channels or hunts, not storage)
// get their own words here; the rest go to the panels' shared table.
const own={PLAN:"dealsErrorPlan",QUOTA:"dealsErrorQuota",DAILY_LIMIT:"dealsErrorDaily",FETCH_BLOCKED:"dealsErrorBlocked",FETCH_FAILED:"dealsErrorFetch",DUPLICATE:"dealsErrorDuplicate",CHANNEL_UNAVAILABLE:"dealsErrorBot",NOT_A_VIDEO:"dealsErrorNotVideo",VIDEO_NOT_FOUND:"dealsErrorVideoMissing",VIDEO_ANNOUNCED:"dealsErrorVideoAnnounced",NUUVEM_NOT_CONFIGURED:"dealsErrorNuuvem",CHANNEL_INVALID:"dealsErrorChannelInvalid",CHANNEL_GONE:"dealsErrorChannelGone",CHANNEL_REFUSED:"dealsErrorChannelRefused",MIRROR_EXISTS:"dealsErrorMirrorExists",STORE_NOT_CONFIGURED:"dealsErrorShopKeys",QUEUE_FULL:"dealsErrorQueueFull"} as const satisfies Record<string,PlainMessageKey>;
const isOwn=(code:string):code is keyof typeof own=>Object.hasOwn(own,code);
const errorSchema=z.object({code:z.string().optional(),detail:z.object({data:z.object({code:z.string().optional()})}).optional()});
export function dealsError(error:unknown):string{
 const parsed=errorSchema.safeParse(error),code=parsed.success?parsed.data.code??parsed.data.detail?.data.code:undefined;
 return code&&isOwn(code)?t(own[code]):errorMessage(error);
}
// Why a post or a channel test did not go through (the delivery's code),
// in words.
const delivery={DESTINATION_DISABLED:"dealsDeliveryDisabled",RATE_LIMIT:"dealsDeliveryRate",PROVIDER:"dealsDeliveryProvider",EXPIRED:"dealsDeliveryExpired"} as const satisfies Record<string,PlainMessageKey>;
const isDelivery=(code:string):code is keyof typeof delivery=>Object.hasOwn(delivery,code);
export const failureText=(code:string|null,fallback:PlainMessageKey="dealsNotSent")=>!code?t(fallback):isDelivery(code)?t(delivery[code]):isOwn(code)?t(own[code]):t(fallback);
