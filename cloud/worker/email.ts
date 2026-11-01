import { hasEmail } from "./auth-options";

// The sign-in code by e-mail, through Cloudflare's Email Service. With
// EMAIL_MODE=log (development) the code only shows in the terminal.

const subjects: Record<string, string> = {
	"sign-in": "Seu código para entrar no Blup",
	"email-verification": "Confirme seu e-mail no Blup",
	"change-email": "Confirme o e-mail da sua conta do Blup",
};

export async function sendCode(env: Cloudflare.Env, email: string, code: string, kind: string) {
	// An account without e-mail yet (passkey only): nothing to send.
	if (!hasEmail(email)) return;
	if (env.EMAIL_MODE === "log") {
		console.log(`[blup] código de ${kind} para ${email}: ${code}`);
		return;
	}
	const subject = subjects[kind] ?? subjects["sign-in"];
	try {
		await env.EMAIL.send({
			to: email,
			from: sender(env.MAIL_FROM),
			subject,
			text: `${subject}: ${code}\n\nVale por 10 minutos. Se não foi você, ignore este e-mail.`,
			html: codeHtml(subject, code),
		});
	} catch (e) {
		// The person sees no difference (the request already answered); the
		// log keeps the reason.
		const err = e as Error & { code?: string };
		console.error("[blup] e-mail não enviado", err.code ?? "", err.message);
	}
}

// "Blup <no-reply@getblup.io>" as Email Service's { name, email }.
function sender(from: string) {
	const m = /^\s*(.*?)\s*<([^>]+)>\s*$/.exec(from);
	return m ? { name: m[1], email: m[2] } : from;
}

function codeHtml(subject: string, code: string) {
	return `<!doctype html><html lang="pt-BR"><body style="margin:0;padding:32px 16px;background:#f5f5f5;font-family:Inter,system-ui,sans-serif;color:#111">
<div style="max-width:420px;margin:0 auto;padding:32px;background:#fff;border-radius:16px;border:1px solid #e5e5e5">
<p style="margin:0 0 8px;font-size:15px">${subject}</p>
<p style="margin:0 0 24px;font-size:36px;font-weight:800;letter-spacing:8px">${code}</p>
<p style="margin:0;font-size:13px;color:#666">Vale por 10 minutos. Se não foi você, ignore este e-mail.</p>
</div></body></html>`;
}
