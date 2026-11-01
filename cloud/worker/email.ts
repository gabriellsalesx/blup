import { hasEmail } from "./auth-options";
import { LOGO_PNG } from "./email-logo";

// The sign-in code by e-mail, through Cloudflare's Email Service. With
// EMAIL_MODE=log (development) the code only shows in the terminal.

const subjects: Record<string, string> = {
	"sign-in": "Seu código para entrar no Blup",
	"email-verification": "Confirme seu e-mail no Blup",
	"change-email": "Confirme o e-mail da sua conta do Blup",
};

// What the code is for, under the logo.
const leads: Record<string, string> = {
	"sign-in": "Use este código para entrar no Blup:",
	"email-verification": "Use este código para confirmar seu e-mail no Blup:",
	"change-email": "Use este código para vincular este e-mail à sua conta do Blup:",
};

export async function sendCode(env: Cloudflare.Env, email: string, code: string, kind: string) {
	// An account without e-mail yet (passkey only): nothing to send.
	if (!hasEmail(email)) return;
	if (env.EMAIL_MODE === "log") {
		console.log(`[blup] código de ${kind} para ${email}: ${code}`);
		return;
	}
	const subject = subjects[kind] ?? subjects["sign-in"];
	const lead = leads[kind] ?? leads["sign-in"];
	try {
		await env.EMAIL.send({
			to: email,
			from: sender(env.MAIL_FROM),
			subject,
			text: `Blup\n\n${lead}\n\n${code}\n\nO código vale por 10 minutos. Se não foi você que pediu, ignore este e-mail: ninguém entra sem ele.\n\nBlup · getblup.io`,
			html: codeHtml(lead, code),
			attachments: [{ content: LOGO_PNG, filename: "blup.png", type: "image/png", disposition: "inline", contentId: "blup-logo" }],
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

function codeHtml(lead: string, code: string) {
	return `<!doctype html><html lang="pt-BR"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta name="color-scheme" content="light"></head>
<body style="margin:0;padding:32px 16px;background:#f4f6fb;font-family:Inter,-apple-system,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;color:#0e1b45">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0"><tr><td align="center">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="max-width:440px;background:#ffffff;border-radius:16px;border:1px solid #e3e8f2">
<tr><td style="padding:32px 32px 8px"><img src="cid:blup-logo" width="110" height="43" alt="Blup" style="display:block;border:0"></td></tr>
<tr><td style="padding:16px 32px 0;font-size:15px;line-height:22px">${lead}</td></tr>
<tr><td style="padding:16px 32px"><div style="padding:16px;border-radius:12px;background:#eef6fd;text-align:center;font-size:36px;font-weight:800;letter-spacing:8px;color:#0e1b45">${code}</div></td></tr>
<tr><td style="padding:0 32px 32px;font-size:13px;line-height:20px;color:#5b6787">O código vale por 10 minutos. Se não foi você que pediu, ignore este e-mail: ninguém entra sem ele.</td></tr>
</table>
<p style="margin:16px 0 0;font-size:12px;color:#8a94ad">Blup · <a href="https://getblup.io" style="color:#8a94ad">getblup.io</a></p>
</td></tr></table>
</body></html>`;
}
