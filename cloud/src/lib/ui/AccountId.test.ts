import {describe,expect,it} from "vitest";
import {render} from "svelte/server";
import AccountId from "./AccountId.svelte";
describe("account ID display",()=>{
	it("renders a short public ID and an accessible copy action",()=>{
		const id="01900000-0000-7000-8000-000000000001";
		const {body}=render(AccountId,{props:{id,short:true}});
		expect(body).toContain("01900000…0001");expect(body).toContain('aria-label="Copiar ID da conta"');expect(body).toContain("Em caso de ajuda, informe seu ID.");expect(body).toContain(id);
	});
	it("escapes an unexpected ID instead of injecting HTML",()=>{
		const {body}=render(AccountId,{props:{id:"<script>alert(1)</script>"}});expect(body).not.toContain("<script>");expect(body).toContain("&lt;script");
	});
});
