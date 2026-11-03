<script lang="ts">
 import {resolve} from "$app/paths";
 import {goto} from "$app/navigation";
 import {t} from "#lib/i18n.svelte.ts";
 import {returnPath} from "#lib/auth.ts";
 import SignInFrame from "#lib/ui/SignInFrame.svelte";
 import SignInForm from "#lib/ui/SignInForm.svelte";
 import {rpc,parseResponse} from "../../lib/api";
 import {pulseAuth} from "../../lib/auth";
 import {panel} from "../../lib/panel.svelte";
 // The Blup sign-in, in the Pulse colors. Signed in, the panel loads the
 // person again and opens the page that asked (?next=).
 async function done(){panel.ready=false;panel.me=null;panel.signedOut=false;await goto(returnPath(location.search,resolve("/")),{replace:true,refreshAll:true});}
</script>
<svelte:head><title>{t("signIn")} · Blup Pulse</title></svelte:head>
<SignInFrame app="pulse"><SignInForm client={pulseAuth} siteKey={async()=>(await parseResponse(rpc.config.$get())).turnstileSiteKey} ondone={()=>void done()}/></SignInFrame>
