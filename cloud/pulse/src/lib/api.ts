import {hc,parseResponse,type InferResponseType} from "hono/client";
import type {AppType} from "../../../src/lib/rpc";
import {language} from "#lib/i18n.svelte.ts";
export const cloudURL=import.meta.env.DEV&&typeof location!=="undefined"?location.origin:"https://cloud.getblup.io";
// The language goes as Accept-Language (a header browsers allow across
// origins without asking first): with no custom header a GET is a "simple"
// request and needs no preflight (OPTIONS) before it. The Worker reads it
// the same way (requestLanguage).
export const rpc=hc<AppType>(cloudURL,{init:{credentials:"include"},headers:()=>({"accept-language":language()})}).api;
export const pulseApi=rpc.pulse;
export {parseResponse};
export type Me=InferResponseType<typeof rpc.me.$get,200>;
export type PulseUsage=InferResponseType<typeof pulseApi.usage.$get,200>;
export type Project=InferResponseType<typeof pulseApi.projects.$get,200>["projects"][number];
export const getIssues=pulseApi.projects[":project"].issues.$get;
export type Issue=InferResponseType<typeof getIssues,200>["issues"][number];
