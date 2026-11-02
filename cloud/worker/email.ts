import {hasEmail} from "./auth-options";
import {languageOf,languageTag,type Language} from "../shared/language";
import {emailText} from "../shared/email-i18n";
async function savedLanguage(env:Cloudflare.Env,email:string,fallback:Language):Promise<Language|null>{
 try{const user=await env.DB.prepare("SELECT language FROM user WHERE email=?").bind(email.trim().toLowerCase()).first<{language:string}>();
 return languageOf(user?.language)??fallback;}catch{console.error("[blup] mail_language_read_failed");return null;}
}
function escapeHtml(value:string){return value.replace(/[&<>"']/g,c=>({"&":"&amp;","<":"&lt;",">":"&gt;",'"':"&quot;","'":"&#39;"})[c]??c);}
const safeMailCode=(code:unknown)=>typeof code==="string"&&/^[A-Z0-9_]{1,64}$/.test(code)?code:"EMAIL_SEND_FAILED";
export async function sendCode(env:Cloudflare.Env,email:string,code:string,kind:string,fallback:Language="en"){
 if(!hasEmail(email))return;
 const language=await savedLanguage(env,email,fallback);if(!language)return;
 if(env.EMAIL_MODE==="log"){console.log(`[blup] código de ${kind} para ${email}: ${code}`);return;}
 const subject=emailText(language,kind==="email-verification"?"verifySubject":kind==="change-email"?"changeSubject":"signInSubject");
 const lead=emailText(language,kind==="email-verification"?"verifyLead":kind==="change-email"?"changeLead":"signInLead"),note=emailText(language,"codeNote");
 try{await env.EMAIL.send({to:email,from:sender(env.MAIL_FROM),subject,text:`Blup\n\n${lead}\n\n${code}\n\n${note}\n\nBlup · getblup.io`,html:codeHtml(lead,code,`${env.APP_URL}/email/blup.png`,note,language)});}
 catch(e){console.error("[blup] email_send_failed",safeMailCode(typeof e==="object"&&e!==null&&"code" in e?e.code:undefined));}
}
export async function sendReminder(env:Cloudflare.Env,email:string,preferred?:Language){
 if(!hasEmail(email))return false;
 const language=preferred??await savedLanguage(env,email,"en");if(!language)return false;
 if(env.EMAIL_MODE==="log"){console.log("[blup] reminder_scheduled");return true;}
 const lead=emailText(language,"reminderLead"),note=emailText(language,"reminderNote"),off=emailText(language,"reminderOff");
 try{await env.EMAIL.send({to:email,from:sender(env.MAIL_FROM),subject:emailText(language,"reminderSubject"),text:`Blup\n\n${lead}\n\n${note}\n\n${off}\n\nBlup · getblup.io`,html:reminderHtml(lead,note,off,`${env.APP_URL}/email/blup.png`,language)});return true;}
 catch(e){console.error("[blup] reminder_send_failed",safeMailCode(typeof e==="object"&&e!==null&&"code" in e?e.code:undefined));return false;}
}
function sender(from: string) {
	const m = /^\s*(.*?)\s*<([^>]+)>\s*$/.exec(from);
	return m ? { name: m[1], email: m[2] } : from;
}

// The logo is a PNG served by this Worker (static/email/blup.png, the
// wordmark of src/lib/ui/BlupLogo.svelte rendered at 440x172): mail apps
// don't show SVG, and an image embedded in the message shows as an
// attachment in Gmail.
function codeHtml(lead:string,code:string,logo:string,note:string,language:Language) {
	return `<!doctype html><html lang="${languageTag[language]}"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta name="color-scheme" content="light"></head>
<body style="margin:0;padding:32px 16px;background:#f4f6fb;font-family:Inter,-apple-system,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;color:#0e1b45">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0"><tr><td align="center">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="max-width:440px;background:#ffffff;border-radius:16px;border:1px solid #e3e8f2">
<tr><td style="padding:32px 32px 8px"><img src="${escapeHtml(logo)}" width="110" height="43" alt="Blup" style="display:block;border:0"></td></tr>
<tr><td style="padding:16px 32px 0;font-size:15px;line-height:22px">${escapeHtml(lead)}</td></tr>
<tr><td style="padding:16px 32px"><div style="padding:16px;border-radius:12px;background:#eef6fd;text-align:center;font-size:36px;font-weight:800;letter-spacing:8px;color:#0e1b45">${escapeHtml(code)}</div></td></tr>
<tr><td style="padding:0 32px 32px;font-size:13px;line-height:20px;color:#5b6787">${escapeHtml(note)}</td></tr>
</table>
<p style="margin:16px 0 0;font-size:12px;color:#8a94ad">Blup · <a href="https://getblup.io" style="color:#8a94ad">getblup.io</a></p>
</td></tr></table>
</body></html>`;
}

function reminderHtml(lead: string, note: string, off: string, logo:string,language:Language) {
	return `<!doctype html><html lang="${languageTag[language]}"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta name="color-scheme" content="light"></head>
<body style="margin:0;padding:32px 16px;background:#f4f6fb;font-family:Inter,-apple-system,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;color:#0e1b45">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0"><tr><td align="center">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="max-width:440px;background:#ffffff;border-radius:16px;border:1px solid #e3e8f2">
<tr><td style="padding:32px 32px 8px"><img src="${escapeHtml(logo)}" width="110" height="43" alt="Blup" style="display:block;border:0"></td></tr>
<tr><td style="padding:16px 32px 0"><div style="padding:16px;border-radius:12px;background:#eef6fd;font-size:18px;line-height:26px;font-weight:700;color:#0e1b45">${escapeHtml(lead)}</div></td></tr>
<tr><td style="padding:16px 32px 0;font-size:15px;line-height:22px">${escapeHtml(note)}</td></tr>
<tr><td style="padding:16px 32px 32px;font-size:13px;line-height:20px;color:#5b6787">${escapeHtml(off)}</td></tr>
</table>
<p style="margin:16px 0 0;font-size:12px;color:#8a94ad">Blup · <a href="https://getblup.io" style="color:#8a94ad">getblup.io</a></p>
</td></tr></table>
</body></html>`;
}


type DowngradeNotice={sites?:{limit:number;deadline:number};storage?:{usedBytes:number;limitBytes:number;deadline:number};domainDisableAt?:number};
export async function sendDowngrade(env:Cloudflare.Env,email:string,notice:DowngradeNotice={},preferred?:Language){
 if(!hasEmail(email))return true;
 const language=preferred??await savedLanguage(env,email,"en");if(!language)return false;
 const locale=languageTag[language];
 const date=(value:number)=>new Intl.DateTimeFormat(locale,{timeZone:"UTC",dateStyle:"long",timeStyle:"short"}).format(value)+" UTC";
 const mb=(value:number)=>new Intl.NumberFormat(locale,{maximumFractionDigits:2}).format(value/(1024*1024))+" MiB";
 const lead=emailText(language,"planLead"),parts:string[]=[];
 if(notice.sites){const {limit,deadline}=notice.sites;parts.push(limit===0?emailText(language,"sitesOff",{date:date(deadline)}):limit===1?emailText(language,"siteChoose",{date:date(deadline)}):emailText(language,"sitesKeep",{date:date(deadline),count:limit}));}
 if(notice.domainDisableAt!==undefined)parts.push(emailText(language,"domainOff",{date:date(notice.domainDisableAt)}));
 if(notice.storage){const {usedBytes,limitBytes,deadline}=notice.storage;parts.push(emailText(language,"storage",{used:mb(usedBytes),limit:mb(limitBytes),date:date(deadline)}));}
 const note=parts.join(" ")||emailText(language,"planNote"),off=emailText(language,"planOff");
 if(env.EMAIL_MODE==="log"){console.log("[blup] plan_notice_scheduled");return true;}
 try{await env.EMAIL.send({to:email,from:sender(env.MAIL_FROM),subject:emailText(language,"planSubject"),text:`Blup\n\n${lead}\n\n${note}\n\n${off}\n\nBlup · getblup.io`,html:reminderHtml(lead,note,off,`${env.APP_URL}/email/blup.png`,language)});return true;}
 catch{console.error("[blup] plan_notice_send_failed");return false;}
}
