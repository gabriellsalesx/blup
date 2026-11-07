import {it,expect,vi} from "vitest";
vi.mock("cloudflare:workers",()=>({env:{},DurableObject:class{},WorkerEntrypoint:class{}}));
import {TestD1} from "./security-test-support";
import {serveHealth} from "./health";
import {sentryOptions} from "./observability";

// What Pulse's uptime monitors check, and where the Cloud's own errors go.
const req=(path:string,method="GET")=>new Request(`https://cloud.audit.invalid${path}`,{method});
it("answers alive and ready, never cached, without details",async()=>{
 const db=new TestD1();const env={DB:db.asD1()};
 const alive=await serveHealth(req("/_blup/health"),env);
 expect(alive?.status).toBe(200);expect(alive?.headers.get("cache-control")).toBe("no-store");expect(await alive!.json()).toEqual({status:"ok"});
 expect((await serveHealth(req("/_blup/ready"),env))?.status).toBe(200);
 expect(await (await serveHealth(req("/_blup/ready","HEAD"),env))!.text()).toBe("");
 expect((await serveHealth(req("/_blup/ready","POST"),env))?.status).toBe(405);
 expect(await serveHealth(req("/health"),env)).toBeNull();
 // A database that fails: not ready (503), still alive.
 const broken={DB:{prepare:()=>({first:async()=>{throw new Error("D1_ERROR: secret detail");}})} as unknown as D1Database};
 const ready=await serveHealth(req("/_blup/ready"),broken);
 expect(ready?.status).toBe(503);expect(await ready!.text()).not.toContain("secret");
 expect((await serveHealth(req("/_blup/health"),broken))?.status).toBe(200);
 db.close();
});
it("sends the Cloud's errors to its own ingest in the same request",()=>{
 const base={PULSE_INGEST_HOST:"ingest.getblup.io",STAFF_HOST:"staff.getblup.io"} as unknown as Cloudflare.Env;
 expect(sentryOptions(base)).toEqual({enabled:false});
 const own=sentryOptions({...base,SENTRY_DSN:"https://abc@ingest.getblup.io/7"});
 expect(own).toMatchObject({dsn:"https://abc@ingest.getblup.io/7",environment:"production",dataCollection:{userInfo:false,cookies:false,httpHeaders:false,urlQueryParams:false}});
 expect(typeof own.transportOptions?.fetch).toBe("function");
 expect(sentryOptions({...base,SENTRY_DSN:"https://abc@other.example/7"}).transportOptions).toBeUndefined();
 // The request kept on an error: only its method and path.
 const event=own.beforeSend!({type:undefined,request:{method:"POST",url:"https://cloud.getblup.io/api/x?token=1",headers:{cookie:"a"},data:"body"},user:{email:"a@b.c"}},{}) as {request:unknown;user?:unknown};
 expect(event.request).toEqual({method:"POST",url:"https://cloud.getblup.io/api/x"});expect(event.user).toBeUndefined();
});
