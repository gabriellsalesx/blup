import {describe,expect,it} from "vitest";
import {TestD1,seedUser} from "./security-test-support";
import {cleanExpiredAuth} from "./auth-cleanup";

describe("cleanExpiredAuth", () => {
 it("removes sessions and verifications a day past expiry, and keeps the rest", async () => {
  const d1=new TestD1(),now=Date.UTC(2026,10,7);
  seedUser(d1,"u1");
  const session=d1.sqlite.prepare("INSERT INTO session(id,expires_at,token,created_at,updated_at,user_id) VALUES (?,?,?,?,?,?)");
  session.run("old",now-2*86_400_000,"t-old",0,0,"u1");
  session.run("recent",now-3600_000,"t-recent",0,0,"u1");
  session.run("live",now+86_400_000,"t-live",0,0,"u1");
  const verification=d1.sqlite.prepare("INSERT INTO verification(id,identifier,value,expires_at,created_at,updated_at) VALUES (?,?,?,?,?,?)");
  for(let i=0;i<510;i++)verification.run(`v${i}`,"x","y",now-3*86_400_000,0,0);
  verification.run("fresh","x","y",now+600_000,0,0);
  const device=d1.sqlite.prepare("INSERT INTO device_code(id,device_code,user_code,expires_at,status,client_id) VALUES (?,?,?,?,'pending','blup-studio')");
  device.run("stale","dc-stale","UC-STALE",now-2*86_400_000);device.run("pending","dc-pending","UC-PENDING",now+600_000);
  await cleanExpiredAuth(d1.asD1(),now);
  expect(d1.sqlite.prepare("SELECT id FROM device_code").all().map(r=>r.id)).toEqual(["pending"]);
  expect(d1.sqlite.prepare("SELECT id FROM session ORDER BY id").all().map(r=>r.id)).toEqual(["live","recent"]);
  expect(d1.sqlite.prepare("SELECT id FROM verification").all().map(r=>r.id)).toEqual(["fresh"]);
 });
});
