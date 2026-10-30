import { createAuthClient } from "better-auth/svelte";
import { emailOTPClient } from "better-auth/client/plugins";
import { passkeyClient } from "@better-auth/passkey/client";

// Same origin as the Worker: /api/auth.
export const authClient = createAuthClient({
	basePath: "/api/auth",
	plugins: [emailOTPClient(), passkeyClient()],
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
		case "MISSING_RESPONSE":
		case "VERIFICATION_FAILED":
			return "Não deu para confirmar que você não é um robô. Tente de novo.";
	}
	return err?.message || "Algo deu errado. Tente de novo.";
}

// The person closed the passkey prompt: not an error to show.
export const cancelled = (err: object) => "code" in err && err.code === "AUTH_CANCELLED";
