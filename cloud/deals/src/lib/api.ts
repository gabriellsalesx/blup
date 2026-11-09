import {hc,parseResponse,type InferResponseType} from "hono/client";
import type {AppType} from "../../../src/lib/rpc";
import {language} from "#lib/i18n.svelte.ts";
// The Deals panel talks to the Cloud's Worker (/api/deals), with the same
// Blup session. The types come from the Worker's routes (RPC), so the
// panel follows whatever the server answers.
export const cloudURL=import.meta.env.DEV&&typeof location!=="undefined"?location.origin:"https://cloud.getblup.io";
// The language goes as Accept-Language (a header browsers allow across
// origins without asking first): with no custom header a GET is a "simple"
// request and needs no preflight (OPTIONS) before it. The Worker reads it
// the same way (requestLanguage).
export const rpc=hc<AppType>(cloudURL,{init:{credentials:"include"},headers:()=>({"accept-language":language()})}).api;
export const dealsApi=rpc.deals;
export {parseResponse};
export type Me=InferResponseType<typeof rpc.me.$get,200>;
export type DealsUsage=InferResponseType<typeof dealsApi.usage.$get,200>;
export type Channel=InferResponseType<typeof dealsApi.channels.$get,200>["channels"][number];
export type StoreState=InferResponseType<typeof dealsApi.affiliates.$get,200>["stores"][number];
export type Store=StoreState["store"];
export type Preview=InferResponseType<typeof dealsApi.offers.preview.$post,200>;
export type VideoPreview=InferResponseType<typeof dealsApi.offers.video.preview.$post,200>;
export type Post=InferResponseType<typeof dealsApi.offers.$get,200>["posts"][number];
export type PostState=Post["state"];
export type Hunt=InferResponseType<typeof dealsApi.hunts.$get,200>["hunts"][number];
export type HuntKind=Hunt["kind"];
export type Stats=InferResponseType<typeof dealsApi.stats.$get,200>;
export type AIUsage=InferResponseType<typeof dealsApi.ai.$get,200>;
export type Settings=InferResponseType<typeof dealsApi.settings.$get,200>;
