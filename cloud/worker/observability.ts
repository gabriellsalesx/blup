// Named imports, not the whole namespace: only what is used goes into the
// bundle (the namespace object pulled in every export of the SDK).
import {captureException,instrumentDurableObjectWithSentry,withSentry,type CloudflareOptions} from "@sentry/cloudflare";
import {pulseIngest} from "./pulse/ingest";
import {BLUP_VERSION} from "../shared/version";

// The Cloud's own errors, sent to Blup Pulse with the official Sentry SDK
// (errors only: Pulse keeps no performance data). The DSN is a secret
// (SENTRY_DSN, from a project in the owner's Pulse account); without it, as
// on this computer, nothing is sent. Personal data is left out here too
// (Pulse strips what is left on arrival).
// https://docs.sentry.io/platforms/javascript/guides/cloudflare/

type Env=Cloudflare.Env;

// `name`: the part of the Cloud the errors come from (its release name).
export function sentryOptions(env:Env,name:"blup-cloud"|"blup-sites"="blup-cloud"):CloudflareOptions{
 if(!env.SENTRY_DSN)return {enabled:false};
 let ingestHost="";
 try{ingestHost=new URL(env.SENTRY_DSN).hostname;}catch{return {enabled:false};}
 // The Cloud's Worker serves the ingest itself: its envelopes go to it in
 // the same request, not out and back through the network (a Worker can't
 // reach its own route).
 const local=ingestHost===env.PULSE_INGEST_HOST;
 return {
  dsn:env.SENTRY_DSN,
  environment:env.SENTRY_ENVIRONMENT||(env.STAFF_HOST?"production":"development"),
  // The project's version; the Cloudflare deployment tells two
  // publications of the same version apart.
  release:`${name}@${BLUP_VERSION}`,
  dist:env.CF_VERSION_METADATA?.id?.slice(0,64),
  // No user, cookies, headers, bodies or query strings.
  dataCollection:{userInfo:false,cookies:false,httpHeaders:false,httpBodies:[],urlQueryParams:false},
  // Errors only: no tracing (no sample rate set).
  sendClientReports:false,
  beforeSend(event){
   // Nothing of the request but its method and path (no query, headers,
   // cookies or body: tokens and personal data live there).
   if(event.request){const url=event.request.url?.split("?")[0];event.request={method:event.request.method,url};}
   delete event.user;
   return event;
  },
  transportOptions:local?{fetch:(input:RequestInfo|URL,init?:RequestInit)=>pulseIngest(new Request(input,init),env)}:undefined,
 };
}

// An error a handler answered itself (a 500 from onError): it reaches Pulse
// too, without changing the answer.
export function reportError(error:unknown){
 try{captureException(error);}catch{/* never break the answer */}
}

// Work that runs past the response (ctx.waitUntil): its failure is reported.
export function watched<T>(work:Promise<T>):Promise<T|undefined>{
 return work.catch(error=>{reportError(error);console.error("[blup] background_failed",error instanceof Error?error.name:"error");return undefined;});
}

export {instrumentDurableObjectWithSentry,withSentry};
