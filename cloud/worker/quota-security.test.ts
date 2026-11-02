import {beforeEach,afterEach,describe,it,expect,vi} from "vitest";
import {TestD1,seedUser} from "./security-test-support";
import {reserve,release,settle,expireReservations,siteLock,unlockSite} from "./quota";
import {approveDevice,deviceMark,finishDevice} from "./devices";
let d1:TestD1;
beforeEach(()=>{vi.useFakeTimers({toFake:["Date"]});vi.setSystemTime(new Date("2026-11-15T12:00:00Z"));d1=new TestD1();seedUser(d1,"quota-owner");});
afterEach(()=>{d1.close();vi.useRealTimers();});
describe("atomic D1 admission on the real SQLite engine",()=>{
	it("shares storage between existing notes, sites and parallel reservations",async()=>{
		d1.sqlite.exec("INSERT INTO sync_account(user_id,wrapped_key,storage_bytes) VALUES ('quota-owner','{}',4);INSERT INTO site(id,user_id,slug,title,bytes) VALUES ('s','quota-owner','quota-site','x',4)");
		const holds=await Promise.all([reserve(d1.asD1(),"quota-owner","storage",2,10),reserve(d1.asD1(),"quota-owner","storage",2,10)]);
		expect(holds.filter(Boolean)).toHaveLength(1);
	});
	it("refunds failed work and prevents double refunds of another hold",async()=>{
		const a=await reserve(d1.asD1(),"quota-owner","storage",5,10),b=await reserve(d1.asD1(),"quota-owner","storage",5,10);
		expect(a&&b).toBeTruthy();if(!a||!b)return;
		await release(d1.asD1(),a);await release(d1.asD1(),a);
		expect(d1.sqlite.prepare("SELECT storage FROM resource_quota").get()?.storage).toBe(5);
		expect(await reserve(d1.asD1(),"quota-owner","storage",6,10)).toBeNull();
	});
	it("rolls back all usage writes if settlement is repeated or expired",async()=>{
		const hold=await reserve(d1.asD1(),"quota-owner","ai",10,100);expect(hold).toBeTruthy();if(!hold)return;
		const write=()=>d1.asD1().prepare("INSERT INTO ai_usage(user_id,kind,units,cost_micros) VALUES ('quota-owner','translation',1,10)");
		await settle(d1.asD1(),hold,[write()]);
		await expect(settle(d1.asD1(),hold,[write()])).rejects.toThrow();
		expect(d1.sqlite.prepare("SELECT count(*) AS n FROM ai_usage").get()?.n).toBe(1);
		const old=await reserve(d1.asD1(),"quota-owner","ai",10,100);if(!old)throw new Error("hold");
		await expireReservations(d1.asD1(),Date.now()+1_000_000);
		await expect(settle(d1.asD1(),old,[write()])).rejects.toThrow();
		expect(d1.sqlite.prepare("SELECT ai FROM resource_quota").get()?.ai).toBe(0);
	});
	it("limits vaults atomically to fifty",async()=>{
		const holds=await Promise.all(Array.from({length:51},()=>reserve(d1.asD1(),"quota-owner","vaults",1,50)));
		expect(holds.filter(Boolean)).toHaveLength(50);
	});
	it("does not release a replacement publish lease",async()=>{
		const a=await siteLock(d1.asD1(),"site",-1);if(!a)throw new Error("lease");
		const b=await siteLock(d1.asD1(),"site");expect(b).toBeTruthy();
		await unlockSite(d1.asD1(),"site",a);
		expect(await siteLock(d1.asD1(),"site")).toBeNull();
	});
});
describe("device approval reservations",()=>{
	function code(id:string){d1.sqlite.prepare("INSERT INTO device_code(id,device_code,user_code,user_id,expires_at,status,client_id) VALUES (?,?,?,'quota-owner',?,'pending','blup-studio')").run(id,`secret-fixture-${id}`,id.toUpperCase(),Date.now()+900_000);}
	it("reserves the last slot at approval, including simultaneous approvals",async()=>{
		code("codeaa");code("codebb");
		const result=await Promise.all([approveDevice(d1.asD1(),"quota-owner","CODEAA",1),approveDevice(d1.asD1(),"quota-owner","CODEBB",1)]);
		expect(result.sort()).toEqual(["DEVICES_LIMIT","OK"]);
	});
	it("ignores forged user-agent strings and keeps a slot during redemption",async()=>{
		code("codeaa");
		d1.sqlite.prepare("INSERT INTO session(id,token,user_id,expires_at,updated_at,user_agent) VALUES ('web','fixture','quota-owner',?,?,'Blup Studio forged')").run(Date.now()+900_000,Date.now());
		expect(await approveDevice(d1.asD1(),"quota-owner","CODEAA",1)).toBe("OK");
		expect(await deviceMark(d1.asD1(),"quota-owner",{device_code:"secret-fixture-codeaa"})).toBe(true);
		expect(await deviceMark(d1.asD1(),"another-account",{device_code:"secret-fixture-codeaa"})).toBe(false);
		d1.sqlite.prepare("INSERT INTO session(id,token,user_id,expires_at,updated_at,studio_device) VALUES ('device','fixture2','quota-owner',?,?,1)").run(Date.now()+900_000,Date.now());
		await finishDevice(d1.asD1(),"quota-owner",{device_code:"secret-fixture-codeaa"});
		code("codebb");expect(await approveDevice(d1.asD1(),"quota-owner","CODEBB",1)).toBe("DEVICES_LIMIT");
		expect(d1.sqlite.prepare("SELECT devices FROM resource_quota").get()?.devices).toBe(0);
	});
	it("rejects a different owner and repeated approval",async()=>{
		code("codeaa");seedUser(d1,"intruder");
		expect(await approveDevice(d1.asD1(),"intruder","CODEAA",1)).toBe("INVALID");
		expect(await approveDevice(d1.asD1(),"quota-owner","CODEAA",1)).toBe("OK");
		expect(await approveDevice(d1.asD1(),"quota-owner","CODEAA",1)).toBe("INVALID");
	});
});
