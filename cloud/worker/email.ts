import { hasEmail } from "./auth-options";

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
			html: codeHtml(lead, code, `${env.APP_URL}/email/blup.png`),
		});
	} catch (e) {
		// The person sees no difference (the request already answered); the
		// log keeps the reason.
		const code=typeof e==="object"&&e!==null&&"code" in e?e.code:undefined;
		console.error("[blup] e-mail não enviado", safeMailCode(code));
	}
}

// "Blup <no-reply@getblup.io>" as Email Service's { name, email }.
function sender(from: string) {
	const m = /^\s*(.*?)\s*<([^>]+)>\s*$/.exec(from);
	return m ? { name: m[1], email: m[2] } : from;
}

// The logo is a PNG served by this Worker (static/email/blup.png, the
// wordmark of src/lib/ui/BlupLogo.svelte rendered at 440x172): mail apps
// don't show SVG, and an image embedded in the message shows as an
// attachment in Gmail.
function codeHtml(lead: string, code: string, logo: string) {
	return `<!doctype html><html lang="pt-BR"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta name="color-scheme" content="light"></head>
<body style="margin:0;padding:32px 16px;background:#f4f6fb;font-family:Inter,-apple-system,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;color:#0e1b45">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0"><tr><td align="center">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="max-width:440px;background:#ffffff;border-radius:16px;border:1px solid #e3e8f2">
<tr><td style="padding:32px 32px 8px"><img src="${logo}" width="110" height="43" alt="Blup" style="display:block;border:0"></td></tr>
<tr><td style="padding:16px 32px 0;font-size:15px;line-height:22px">${lead}</td></tr>
<tr><td style="padding:16px 32px"><div style="padding:16px;border-radius:12px;background:#eef6fd;text-align:center;font-size:36px;font-weight:800;letter-spacing:8px;color:#0e1b45">${code}</div></td></tr>
<tr><td style="padding:0 32px 32px;font-size:13px;line-height:20px;color:#5b6787">O código vale por 10 minutos. Se não foi você que pediu, ignore este e-mail: ninguém entra sem ele.</td></tr>
</table>
<p style="margin:16px 0 0;font-size:12px;color:#8a94ad">Blup · <a href="https://getblup.io" style="color:#8a94ad">getblup.io</a></p>
</td></tr></table>
</body></html>`;
}

// A reminder of the calendar (Max and Ultra), on purpose without what it
// is about: Blup never knows the event, only that one is due.
export async function sendReminder(env: Cloudflare.Env, email: string) {
	if (!hasEmail(email)) return false;
	if (env.EMAIL_MODE === "log") {
		console.log(`[blup] lembrete da agenda para ${email}`);
		return true;
	}
	const lead = "Você tem um lembrete na sua agenda do Blup.";
	const note = "Abra o Blup Studio para ver o que é. Por segurança, este e-mail não traz os detalhes: eles ficam só no seu computador.";
	const off = "Não quer mais estes avisos? Desligue as notificações em Configurações, no Blup Studio.";
	try {
		await env.EMAIL.send({
			to: email,
			from: sender(env.MAIL_FROM),
			subject: "Você tem um lembrete no Blup",
			text: `Blup\n\n${lead}\n\n${note}\n\n${off}\n\nBlup · getblup.io`,
			html: reminderHtml(lead, note, off, `${env.APP_URL}/email/blup.png`),
		});
		return true;
	} catch (e) {
		const code=typeof e==="object"&&e!==null&&"code" in e?e.code:undefined;
		console.error("[blup] lembrete não enviado", safeMailCode(code));
		return false;
	}
}

// Provider messages can echo recipients and codes. Keep only a stable code.
const safeMailCode = (code: unknown) => typeof code === "string" && /^[A-Z0-9_]{1,64}$/.test(code) ? code : "EMAIL_SEND_FAILED";

// The same card as the code's: the logo, the message and a quiet footer.
function reminderHtml(lead: string, note: string, off: string, logo: string) {
	return `<!doctype html><html lang="pt-BR"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta name="color-scheme" content="light"></head>
<body style="margin:0;padding:32px 16px;background:#f4f6fb;font-family:Inter,-apple-system,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;color:#0e1b45">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0"><tr><td align="center">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="max-width:440px;background:#ffffff;border-radius:16px;border:1px solid #e3e8f2">
<tr><td style="padding:32px 32px 8px"><img src="${logo}" width="110" height="43" alt="Blup" style="display:block;border:0"></td></tr>
<tr><td style="padding:16px 32px 0"><div style="padding:16px;border-radius:12px;background:#eef6fd;font-size:18px;line-height:26px;font-weight:700;color:#0e1b45">${lead}</div></td></tr>
<tr><td style="padding:16px 32px 0;font-size:15px;line-height:22px">${note}</td></tr>
<tr><td style="padding:16px 32px 32px;font-size:13px;line-height:20px;color:#5b6787">${off}</td></tr>
</table>
<p style="margin:16px 0 0;font-size:12px;color:#8a94ad">Blup · <a href="https://getblup.io" style="color:#8a94ad">getblup.io</a></p>
</td></tr></table>
</body></html>`;
}

export async function sendDowngrade(env:Cloudflare.Env,email:string){
 if(!hasEmail(email))return true;
 const lead="Seu site no Blup continua no ar por mais 14 dias.";
 const note="Seu plano passou para o Grátis. Depois desse prazo, seus sites ficam fora do ar. Assinar novamente religa os sites sem precisar publicar de novo.";
 if(env.EMAIL_MODE==="log"){console.log("[blup] aviso de plano agendado");return true;}
 try{await env.EMAIL.send({to:email,from:sender(env.MAIL_FROM),subject:"Sobre seus sites no Blup",text:`Blup\n\n${lead}\n\n${note}\n\nBlup · getblup.io`,html:reminderHtml(lead,note,"Abra sua conta no Blup Cloud para ver seu plano.",`${env.APP_URL}/email/blup.png`)});return true;}
 catch{console.error("[blup] aviso de plano não enviado");return false;}
}
