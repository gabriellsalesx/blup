<script lang="ts">
	import { goto } from "$app/navigation";
	import Mail from "@lucide/svelte/icons/mail";
	import KeyRound from "@lucide/svelte/icons/key-round";
	import UserPlus from "@lucide/svelte/icons/user-plus";
	import Button from "#lib/ui/Button.svelte";
	import FloatField from "#lib/ui/FloatField.svelte";
	import { authClient, cancelled, message } from "#lib/auth.ts";
	import { loadTurnstile } from "#lib/turnstile.ts";

	// Two ways in: a code sent by e-mail (which also creates the account)
	// or a passkey already added to the account.
	let step = $state<"email" | "code">("email");
	let email = $state("");
	let code = $state("");
	let busy = $state(false);
	let error = $state("");
	let captcha = $state("");
	let widgetId = "";

	// The Turnstile widget, rendered into its box once the script loads.
	function turnstile(el: HTMLDivElement) {
		let gone = false;
		let ts: Awaited<ReturnType<typeof loadTurnstile>> | undefined;
		Promise.all([fetch("/api/config").then((r) => r.json() as Promise<{ turnstileSiteKey: string }>), loadTurnstile()])
			.then(([cfg, t]) => {
				if (gone) return;
				ts = t;
				widgetId = t.render(el, {
					sitekey: cfg.turnstileSiteKey,
					appearance: "interaction-only",
					callback: (token: string) => (captcha = token),
					"expired-callback": () => (captcha = ""),
				});
			})
			.catch(() => (error = "Não deu para carregar a verificação. Recarregue a página."));
		return () => {
			gone = true;
			if (ts && widgetId) ts.remove(widgetId);
		};
	}

	async function sendCode(e?: SubmitEvent) {
		e?.preventDefault();
		error = "";
		busy = true;
		const { error: err } = await authClient.emailOtp.sendVerificationOtp({
			email: email.trim(),
			type: "sign-in",
			fetchOptions: { headers: { "x-captcha-response": captcha } },
		});
		busy = false;
		// A token works once: the next send needs a new one.
		captcha = "";
		if (widgetId) window.turnstile?.reset(widgetId);
		if (err) return (error = message(err));
		step = "code";
	}

	async function signIn(e: SubmitEvent) {
		e.preventDefault();
		error = "";
		busy = true;
		const { error: err } = await authClient.signIn.emailOtp({ email: email.trim(), otp: code.trim() });
		busy = false;
		if (err) return (error = message(err));
		goto("/conta", { replaceState: true });
	}

	// A new account with a passkey only (the e-mail can be linked later).
	async function createWithPasskey() {
		error = "";
		const { error: err } = await authClient.passkey.addPasskey({
			name: "Blup Cloud",
			createSession: true,
			fetchOptions: { headers: { "x-captcha-response": captcha } },
		});
		captcha = "";
		if (widgetId) window.turnstile?.reset(widgetId);
		if (err) return (error = cancelled(err) ? "" : message(err));
		goto("/conta", { replaceState: true });
	}

	async function passkey() {
		error = "";
		const { error: err } = await authClient.signIn.passkey();
		if (err) return (error = cancelled(err) ? "" : message(err));
		goto("/conta", { replaceState: true });
	}
</script>

<section class="w-full max-w-[400px]">
	{#if step === "email"}
		<div class="grid gap-3">
			<Button pill class="w-full" type="button" onclick={passkey}>
				<KeyRound class="size-[18px]" />
				Entrar com passkey
			</Button>
			<Button pill variant="secondary" class="w-full" type="button" disabled={!captcha} onclick={createWithPasskey}>
				<UserPlus class="size-[18px]" />
				Criar conta com passkey
			</Button>
		</div>

		<div class="relative my-6 flex items-center justify-center">
			<span class="absolute inset-x-0 border-t border-neutral-200 dark:border-neutral-800"></span>
			<span class="relative bg-neutral-50 px-4 text-[13px] text-neutral-500 lowercase dark:bg-neutral-950">ou</span>
		</div>

		<form class="grid gap-4" onsubmit={sendCode}>
			<FloatField id="email" label="E-mail" type="email" autocomplete="email webauthn" required bind:value={email} />
			<Button pill class="w-full" {busy} disabled={!captcha || !email.trim()}>Entrar com e-mail</Button>
		</form>
	{:else}
		<div class="rounded-xl border border-neutral-200 bg-white p-6 text-center dark:border-neutral-800 dark:bg-neutral-900">
			<div class="mb-4 flex justify-center">
				<span class="grid size-12 place-items-center rounded-full bg-neutral-100 dark:bg-neutral-800"><Mail class="size-6" /></span>
			</div>
			<h1 class="mb-2 text-lg font-semibold">Verifique seu e-mail</h1>
			<p class="mb-5 text-sm text-neutral-500">Enviamos um código para <span class="text-neutral-900 dark:text-white">{email}</span>. Ele vale por 10 minutos. Primeira vez? A conta é criada ao entrar.</p>
			<form class="grid gap-3" onsubmit={signIn}>
				<FloatField id="code" label="Código" surface="bg-white dark:bg-neutral-900" class="text-center text-2xl font-bold tracking-[0.5em]" inputmode="numeric" autocomplete="one-time-code" maxlength={6} required bind:value={code} />
				<Button pill class="w-full" {busy} disabled={code.trim().length < 6}>Entrar</Button>
			</form>
			<button class="mt-4 text-sm text-neutral-500 underline hover:text-neutral-900 dark:hover:text-white" onclick={() => { step = "email"; code = ""; error = ""; }}>Usar um e-mail diferente</button>
		</div>
	{/if}

	<!-- Turnstile: outside the steps, so it survives going back. -->
	<div class="mt-3" class:hidden={step !== "email"} {@attach turnstile}></div>

	{#if error}<p class="mt-4 text-center text-sm text-red-600" role="alert">{error}</p>{/if}
</section>
