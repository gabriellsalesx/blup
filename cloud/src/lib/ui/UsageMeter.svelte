<script lang="ts">
 // How much of a plan limit is used (Pulse and Deals): a line of text
 // and a bar (red from 90%).
 let {text,used,limit}:{text:string;used:number;limit:number}=$props();
 const share=$derived(limit>0?Math.min(100,used/limit*100):0);
</script>

<div><p class="text-sm text-neutral-500">{text}</p><div class="mt-2 h-2 overflow-hidden rounded-full bg-neutral-200 dark:bg-neutral-800"><div class="h-full rounded-full {share>=90?"bg-red-500":"bg-blup"}" style:width="{share}%"></div></div></div>
