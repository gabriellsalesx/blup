import {hc,parseResponse,type InferResponseType} from "hono/client";
import type {AppType} from "../../../src/lib/rpc";
import {language} from "#lib/i18n.svelte.ts";
// The Deals panel talks to the Cloud's Worker (/api/deals), with the same
// Blup session. The types come from the Worker's routes (RPC), so the
// panel follows whatever the server answers.
export const cloudURL=import.meta.env.DEV&&typeof location!=="undefined"?location.origin:"https://cloud.getblup.io";
export const rpc=hc<AppType>(cloudURL,{init:{credentials:"include"},headers:()=>({"x-blup-language":language()})}).api;
export const dealsApi=rpc.deals;
export {parseResponse};
export type Me=InferResponseType<typeof rpc.me.$get,200>;
export type DealsUsage=InferResponseType<typeof dealsApi.usage.$get,200>;
