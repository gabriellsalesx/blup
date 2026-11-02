import { describe, expect, it, vi } from "vitest";
import { sendCode, sendReminder } from "./email";

describe("mail error privacy",()=>{
	it("does not log provider errors that echo the recipient or OTP",async()=>{
		const log=vi.spyOn(console,"error").mockImplementation(()=>{});
		const env={APP_URL:"https://cloud.audit.invalid",EMAIL_MODE:"send",MAIL_FROM:"Blup <noreply@audit.invalid>",EMAIL:{send:async()=>{throw new Error("recipient fixture@audit.invalid rejected code 123456");}}} as unknown as Cloudflare.Env;
		try {
			await sendCode(env,"fixture@audit.invalid","123456","sign-in");await sendReminder(env,"fixture@audit.invalid");
			expect(log).toHaveBeenCalledTimes(2);
			expect(JSON.stringify(log.mock.calls)).not.toMatch(/fixture@audit\.invalid|123456/);
		} finally {log.mockRestore();}
	});
});
