import {type InferResponseType} from "hono/client";
import {pulseApi,parseResponse} from "./api";
// The alert destinations of the account and which project sends to which,
// as the panel shows them: grouped by project, in a fixed order of kinds.
export type Channels=InferResponseType<typeof pulseApi.channels.$get,200>;
export type Channel=Channels["channels"][number];
export type ChannelKind=Channel["type"];
// What a destination's test can send (the server's `testKinds`; the RPC
// types check the request).
export const testKinds=["test","issue","regression","monitor_down","monitor_up"] as const;
export type TestKind=(typeof testKinds)[number];

export const channelKinds=["discord","telegram","slack","webhook"] as const satisfies readonly ChannelKind[];
export const kindName:Record<ChannelKind,string>={discord:"Discord",telegram:"Telegram",slack:"Slack",webhook:"Webhook"};
export const noChannels:Channels={channels:[],bindings:[]};
export const loadChannels=()=>parseResponse(pulseApi.channels.$get());

// The destinations a project sends to, in the order of `channelKinds`.
export function projectChannels(all:Channels,project:string):Channel[]{
 const ids=new Set(all.bindings.filter(b=>b.projectId===project).map(b=>b.channelId));
 return all.channels.filter(c=>ids.has(c.id)).sort((a,b)=>channelKinds.indexOf(a.type)-channelKinds.indexOf(b.type));
}

// The dispatcher sends at each minute: a test made at `at` leaves at the
// next one.
export const roundAfter=(at:number)=>Math.floor(at/60_000)*60_000+60_000;
// "0:42" for a time left in milliseconds.
export function clock(ms:number){const s=Math.max(0,Math.ceil(ms/1000));return `${Math.floor(s/60)}:${String(s%60).padStart(2,"0")}`;}
