<script lang="ts">
 import {t,language} from "#lib/i18n.svelte.ts";
	import Mail from "@lucide/svelte/icons/mail";
	import Fingerprint from "@lucide/svelte/icons/fingerprint-pattern";
	import Button from "./Button.svelte";
	import FloatField from "./FloatField.svelte";
	import { cancelled, message, type BlupAuthClient } from "#lib/auth.ts";
	import { loadTurnstile } from "#lib/turnstile.ts";

	// Two ways in: a code sent by e-mail (which also creates the account)
	// or a passkey already added to the account. Both panels sign in here,
	// each with its own client (`client`), the Turnstile key from its
	// Worker (`siteKey`) and where to go next (`ondone`).
	let { client, siteKey, ondone }: { client: BlupAuthClient; siteKey: () => Promise<string>; ondone: () => void } = $props();
	let step = $state<"email" | "code">("email");
	let email = $state("");
	let code = $state("");
	let busy = $state(false);
	let error = $state<(()=>string)|null>(null);
	let captcha = $state("");
	let widgetId = "";

	// The Turnstile widget, rendered into its box once the script loads.
	function turnstile(el: HTMLDivElement) {
        const selectedLanguage=language();
        captcha="";
		let gone = false;
		let ts: Awaited<ReturnType<typeof loadTurnstile>> | undefined;
		Promise.all([siteKey(), loadTurnstile()])
			.then(([key, t]) => {
				if (gone) return;
				ts = t;
				widgetId = t.render(el, {
                    language: selectedLanguage,
					sitekey: key,
					appearance: "interaction-only",
					callback: (token: string) => (captcha = token),
					"expired-callback": () => (captcha = ""),
				});
			})
			.catch(() => (error = ()=>t("captchaLoad")));
		return () => {
			gone = true;
			if (ts && widgetId) ts.remove(widgetId);
		};
	}

	async function sendCode(e?: SubmitEvent) {
		e?.preventDefault();
		error = null;
		busy = true;
		const { error: err } = await client.emailOtp.sendVerificationOtp({
			email: email.trim(),
			type: "sign-in",
			fetchOptions: { headers: { "x-captcha-response": captcha } },
		});
		busy = false;
		// A token works once: the next send needs a new one.
		captcha = "";
		if (widgetId) window.turnstile?.reset(widgetId);
		if (err) return (error = ()=>message(err));
		step = "code";
	}

	async function signIn(e: SubmitEvent) {
		e.preventDefault();
		error = null;
		busy = true;
		const { error: err } = await client.signIn.emailOtp({ email: email.trim(), otp: code.trim() });
		busy = false;
		if (err) return (error = ()=>message(err));
		ondone();
	}

	// A new account with a passkey only (the e-mail can be linked later).
	async function createWithPasskey() {
		error = null;
		const { error: err } = await client.passkey.addPasskey({
			name: "Blup Cloud",
			createSession: true,
			fetchOptions: { headers: { "x-captcha-response": captcha } },
		});
		captcha = "";
		if (widgetId) window.turnstile?.reset(widgetId);
		if (err) return (error = cancelled(err) ? null : ()=>message(err));
		ondone();
	}

	async function passkey() {
		error = null;
		const { error: err } = await client.signIn.passkey();
		if (err) return (error = cancelled(err) ? null : ()=>message(err));
		ondone();
	}
</script>

<section class="w-full">
	{#if step === "email"}
		<h1 class="text-3xl font-extrabold tracking-tight">{t("welcome")}</h1>
		<p class="mt-2 mb-8 text-neutral-500">{t("welcomeLead")}</p>
		<div class="grid gap-3">
			<Button pill class="w-full" type="button" onclick={passkey}>
				<Fingerprint class="size-5" />
				{t("signInPasskey")}
			</Button>
			<Button pill variant="secondary" class="w-full" type="button" disabled={!captcha} onclick={createWithPasskey}>
				<Fingerprint class="size-5" />
				{t("createPasskey")}
			</Button>
		</div>

		<div class="relative my-6 flex items-center justify-center">
			<span class="absolute inset-x-0 border-t border-neutral-200 dark:border-neutral-800"></span>
			<span class="relative bg-white px-4 text-[13px] text-neutral-500 lowercase dark:bg-neutral-900">{t("or")}</span>
		</div>

		<form class="grid gap-4" onsubmit={sendCode}>
			<FloatField id="email" label={t("email")}  surface="bg-white dark:bg-neutral-900" type="email" autocomplete="email webauthn" required bind:value={email} />
			<Button pill class="w-full" {busy} disabled={!captcha || !email.trim()}>{t("signInEmail")}</Button>
		</form>
	{:else}
		<div class="text-center">
			<div class="mb-4 flex justify-center">
				<span class="grid size-12 place-items-center rounded-full bg-neutral-100 dark:bg-neutral-800"><Mail class="size-6" /></span>
			</div>
			<h1 class="mb-2 text-lg font-semibold">{t("checkEmail")}</h1>
			<p class="mb-5 text-sm text-neutral-500">{t("sentCode",{email})} {t("codeExpiry")}</p>
			<form class="grid gap-3" onsubmit={signIn}>
				<FloatField id="code" label={t("code")}  surface="bg-white dark:bg-neutral-900" class="text-center text-2xl font-bold tracking-[0.5em]" inputmode="numeric" autocomplete="one-time-code" maxlength={6} required bind:value={code} />
				<Button pill class="w-full" {busy} disabled={code.trim().length < 6}>{t("signIn")}</Button>
			</form>
			<button class="mt-4 text-sm text-neutral-500 underline hover:text-neutral-900 dark:hover:text-white" onclick={() => { step = "email"; code = ""; error = null; }}>{t("differentEmail")}</button>
		</div>
	{/if}

	<!-- Turnstile: outside the steps, so it survives going back. -->
	<div class="mt-3" class:hidden={step !== "email"} {@attach turnstile}></div>

	{#if error}<p class="mt-4 text-center text-sm text-red-600" role="alert">{error()}</p>{/if}
</section>
