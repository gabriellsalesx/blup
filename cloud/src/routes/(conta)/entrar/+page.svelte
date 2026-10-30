<script lang="ts">
	import { goto } from "$app/navigation";
	import Mail from "@lucide/svelte/icons/mail";
	import KeyRound from "@lucide/svelte/icons/key-round";
	import Button from "#lib/ui/Button.svelte";
	import Field from "#lib/ui/Field.svelte";
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

	async function passkey() {
		error = "";
		const { error: err } = await authClient.signIn.passkey();
		if (err) return (error = cancelled(err) ? "" : message(err));
		goto("/conta", { replaceState: true });
	}
</script>

<section class="w-full max-w-sm">
	<h1 class="text-center text-2xl font-extrabold tracking-tight">Entre na sua conta</h1>

	{#if step === "email"}
		<p class="mt-1 text-center text-sm text-neutral-500">Mandamos um código para o seu e-mail. Primeira vez? A conta é criada na hora.</p>
		<form class="mt-6 grid gap-3" onsubmit={sendCode}>
			<Field label="E-mail" type="email" autocomplete="email webauthn" required bind:value={email} />
			<Button {busy} disabled={!captcha || !email.trim()}>
				{#if !busy}<Mail class="size-4" />{/if}
				Receber código
			</Button>
		</form>
		<div class="my-5 flex items-center gap-3 text-xs text-neutral-400"><span class="h-px flex-1 bg-neutral-200 dark:bg-neutral-800"></span>ou<span class="h-px flex-1 bg-neutral-200 dark:bg-neutral-800"></span></div>
		<Button variant="secondary" class="w-full" type="button" onclick={passkey}>
			<KeyRound class="size-4" />
			Entrar com passkey
		</Button>
	{:else}
		<p class="mt-1 text-center text-sm text-neutral-500">Digite o código que mandamos para <b class="text-neutral-800 dark:text-neutral-200">{email}</b>. Ele vale por 10 minutos.</p>
		<form class="mt-6 grid gap-3" onsubmit={signIn}>
			<Field class="py-3 text-center text-2xl font-bold tracking-[0.5em]" inputmode="numeric" autocomplete="one-time-code" maxlength={6} aria-label="Código" required bind:value={code} />
			<Button {busy} disabled={code.trim().length < 6}>Entrar</Button>
		</form>
		<button class="mt-4 w-full text-center text-sm font-semibold text-blup hover:underline" onclick={() => { step = "email"; code = ""; error = ""; }}>Usar outro e-mail ou pedir outro código</button>
	{/if}

	<!-- Turnstile: outside the steps, so it survives going back. -->
	<div class="mt-3" class:hidden={step !== "email"} {@attach turnstile}></div>

	{#if error}<p class="mt-4 text-sm text-red-600" role="alert">{error}</p>{/if}
</section>
