// Run with an isolated installation of the official SDK. No product dependency.
import {createRequire} from "node:module";
import {writeFile} from "node:fs/promises";
import {resolve} from "node:path";
import assert from "node:assert/strict";
const origin=new URL(process.env.PULSE_TEST_ORIGIN??"http://127.0.0.1:18864");
if(origin.protocol!=="http:"||!['localhost','127.0.0.1'].includes(origin.hostname))throw new Error("Local server required");
const headers={authorization:"Bearer pulse-sdk-local-session","content-type":"application/json"};
async function api(path,options={}){const res=await fetch(new URL(`/api/pulse${path}`,origin),{...options,headers});if(!res.ok)throw new Error(`Local API ${res.status}: ${await res.text()}`);return res.json();}
const project=await api('/projects',{method:'POST',body:JSON.stringify({name:'Official SDK local check',platform:'node'})});
const dsn=new URL(project.key.dsn);dsn.protocol='http:';dsn.hostname='localhost';dsn.port=origin.port;
const require=createRequire(resolve(process.env.PULSE_SDK_DIR,'package.json'));
const Sentry=require('@sentry/node');
Sentry.init({dsn:dsn.href,defaultIntegrations:false,sendDefaultPii:false,sendClientReports:false,enableLogs:false,tracesSampleRate:0});
const eventId=Sentry.captureException(new Error('Local SDK exception'),{user:{email:'sentinel@audit.invalid'},extra:{note:'PRIVATE_NOTE_SENTINEL'},tags:{token:'PRIVATE_TOKEN_SENTINEL'}});
Sentry.captureMessage('Local SDK message');
assert.equal(await Sentry.flush(10000),true);
const {issues}=await api(`/projects/${project.id}/issues`);
assert.equal(issues.length,2);
const events=(await Promise.all(issues.map(i=>api(`/projects/${project.id}/issues/${i.id}/events`)))).flatMap(r=>r.events);
assert.ok(events.some(e=>e.id===eventId));
const stored=JSON.stringify(events);assert.ok(!stored.includes('PRIVATE_'));assert.ok(!stored.includes('sentinel@audit.invalid'));
await writeFile(process.env.PULSE_TEST_HANDOFF,JSON.stringify({dsn:dsn.href,project:project.id,origin:origin.href}));
console.log(`Official JavaScript SDK: ${events.length} events persisted, private sentinels removed`);
