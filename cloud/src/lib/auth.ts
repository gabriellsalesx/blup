import { createAuthClient } from "better-auth/svelte";
import { emailOTPClient, deviceAuthorizationClient } from "better-auth/client/plugins";
import { passkeyClient } from "@better-auth/passkey/client";
import { stripeClient } from "@better-auth/stripe/client";

// Same origin as the Worker: /api/auth.
export const authClient = createAuthClient({
	basePath: "/api/auth",
	plugins: [emailOTPClient(), passkeyClient(), stripeClient({ subscription: true }), deviceAuthorizationClient()],
});

// Better Auth's errors, in the panel's words.
export function message(err: { code?: string; message?: string } | null | undefined): string {
	switch (err?.code) {
		case "INVALID_OTP":
			return "Código errado. Confira e tente de novo.";
		case "OTP_EXPIRED":
			return "Esse código venceu. Peça um novo.";
		case "TOO_MANY_ATTEMPTS":
			return "Muitas tentativas. Peça um novo código.";
		case "TOO_MANY_REQUESTS":
			return "Muitos pedidos. Espere um minuto.";
		case "EMAIL_REQUIRED":
			return "Vincule um e-mail no Perfil antes de assinar.";
		case "MISSING_RESPONSE":
		case "VERIFICATION_FAILED":
			return "Não deu para confirmar que você não é um robô. Tente de novo.";
	}
	// Stripe's side (customer, checkout, portal): the details stay in the log.
	if (/CUSTOMER|SUBSCRIPTION|CHECKOUT|BILLING/.test(err?.code ?? "")) return "Não deu para abrir o pagamento agora. Tente de novo em instantes.";
	return err?.message || "Algo deu errado. Tente de novo.";
}

// The person closed the passkey prompt: not an error to show.
export const cancelled = (err: object) => "code" in err && err.code === "AUTH_CANCELLED";

// Where to go after signing in: the page that sent the person to /entrar
// (?volta=), only if it is a page of this panel.
export function afterSignIn(): string {
	const back = new URLSearchParams(location.search).get("volta") ?? "";
	if (!back.startsWith("/") || back.startsWith("//") || /[\u0000-\u001f\u007f\\]/.test(back)) return "/conta";
	try {
		const url = new URL(back, location.origin);
		return url.origin === location.origin ? url.pathname + url.search + url.hash : "/conta";
	} catch { return "/conta"; }
}
