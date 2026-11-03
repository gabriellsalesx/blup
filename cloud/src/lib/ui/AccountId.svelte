<script lang="ts">
 import {t,language} from "#lib/i18n.svelte.ts";
	import CopyButton from "./CopyButton.svelte";
	import { accountIdHelp,shortAccountId } from "#lib/account-id.ts";
	let {id,short=false,emphasis=false}:{id:string;short?:boolean;emphasis?:boolean}=$props();
</script>

<span class="inline-flex max-w-full flex-wrap items-center gap-2">
	<code class="select-all break-all {emphasis?'text-base font-bold':'text-xs'}" title={id}>{short?shortAccountId(id):id}</code>
	<CopyButton text={id} size="xs" label={t("copyId")} title={accountIdHelp(language())} />
</span>
