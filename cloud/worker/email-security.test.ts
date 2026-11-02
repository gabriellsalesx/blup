import { describe, expect, it, vi } from "vitest";
import {TestD1} from "./security-test-support";
import { sendCode, sendReminder } from "./email";

describe("mail error privacy",()=>{
 it("does not leak account data or reject the background task when language lookup fails",async()=>{
  const db=new TestD1(),log=vi.spyOn(console,"error").mockImplementation(()=>{}),send=vi.fn();
  const env={DB:db.asD1(),EMAIL:{send}} as unknown as Cloudflare.Env;
  vi.spyOn(db,"prepare").mockImplementation(()=>{throw new Error("fixture@audit.invalid 123456");});
  try{
   await expect(sendCode(env,"fixture@audit.invalid","123456","sign-in","es")).resolves.toBeUndefined();
   expect(send).not.toHaveBeenCalled();expect(log.mock.calls).toEqual([["[blup] mail_language_read_failed"]]);
  }finally{log.mockRestore();db.close();}
 });

	it("does not log provider errors that echo the recipient or OTP",async()=>{
		const db=new TestD1();
		const log=vi.spyOn(console,"error").mockImplementation(()=>{});
		const env={DB:db.asD1(),APP_URL:"https://cloud.audit.invalid",EMAIL_MODE:"send",MAIL_FROM:"Blup <noreply@audit.invalid>",EMAIL:{send:async()=>{throw new Error("recipient fixture@audit.invalid rejected code 123456");}}} as unknown as Cloudflare.Env;
		try {
			await sendCode(env,"fixture@audit.invalid","123456","sign-in");await sendReminder(env,"fixture@audit.invalid");
			expect(log).toHaveBeenCalledTimes(2);
			expect(JSON.stringify(log.mock.calls)).not.toMatch(/fixture@audit\.invalid|123456/);
		} finally {log.mockRestore();db.close();}
	});
});
