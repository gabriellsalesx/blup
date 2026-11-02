<script lang="ts">
 import {t,language} from "#lib/i18n.svelte.ts";
	import Copy from "@lucide/svelte/icons/copy";
	import { accountIdHelp,copyAccountId,shortAccountId } from "#lib/account-id.ts";
	import { toast } from "#lib/toast.svelte.ts";
	let {id,short=false,emphasis=false}:{id:string;short?:boolean;emphasis?:boolean}=$props();
	let copied=$state(false);
	async function copy(){
		if(await copyAccountId(id)){copied=true;toast(()=>t("copied"));}
		else toast(()=>t("copyFailed"),{kind:"error"});
	}
</script>

<span class="inline-flex max-w-full flex-wrap items-center gap-2">
	<code class="select-all break-all {emphasis?'text-base font-bold':'text-xs'}" title={id}>{short?shortAccountId(id):id}</code>
	<button type="button" class="inline-flex shrink-0 items-center gap-1.5 rounded-md border border-neutral-300 px-2 py-1 text-xs font-semibold hover:bg-neutral-100 dark:border-neutral-700 dark:hover:bg-neutral-800" aria-label={t("copyId")}  title={accountIdHelp(language())} onclick={copy}><Copy class="size-3.5" /><span aria-live="polite">{copied?t("copied"):t("copy")}</span></button>
</span>
