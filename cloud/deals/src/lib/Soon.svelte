<script lang="ts">
 import Hourglass from "@lucide/svelte/icons/hourglass";
 import {t} from "#lib/i18n.svelte.ts";
 import Card from "#lib/ui/Card.svelte";
 // The body of a form that is not ready yet (phase F0): one calm line.
 let {lead}:{lead:string}=$props();
</script>
<div class="mt-8"><Card>
 <p class="text-neutral-500">{lead}</p>
 <p class="mt-4 flex items-center gap-2 font-semibold"><Hourglass class="size-4 text-blup"/>{t("dealsSoon")}</p>
</Card></div>
