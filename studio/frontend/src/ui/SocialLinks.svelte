<script lang="ts">
  import { tick } from "svelte";
  import X from "@lucide/svelte/icons/x";
  import { NETWORKS, networkOf } from "../social";
  import { t } from "../i18n";

  // A list of profile links (the site's networks, an author's): each with
  // the icon of its network, and a button per network that adds one with
  // the start of the address typed, ready for the rest.
  let { links = $bindable([]), label = "", help = "", add }: { links: string[]; label?: string; help?: string; add: string } = $props();
  let box = $state<HTMLDivElement>();

  async function addNetwork(prefix: string) {
    links.push(prefix);
    await tick();
    const inputs = box?.querySelectorAll<HTMLInputElement>(".link input");
    const el = inputs?.[inputs.length - 1];
    el?.focus();
    el?.setSelectionRange(el.value.length, el.value.length);
  }
</script>

<div class="field" bind:this={box}>{#if label}<span>{label}</span>{/if}
  {#if help}<small>{help}</small>{/if}
  {#each links as _, i (i)}
    {@const net = networkOf(links[i])}
    {@const bare = NETWORKS.some((n) => n.prefix === links[i].trim())}
    <div class="row link">
      <span class="net" title={net.name}>{@html net.icon}</span>
      <input type="text" class:incomplete={bare} bind:value={links[i]} aria-label={net.name} aria-invalid={bare} />
      <button class="icon-btn" title={t.remove} onclick={() => links.splice(i, 1)}><X size={16} /></button>
    </div>
    {#if bare}<small class="bad">{t.look.socialIncomplete}</small>{/if}
  {/each}
  <small>{add}</small>
  <div class="nets">
    {#each NETWORKS as n (n.key)}
      <button class="net-btn" onclick={() => addNetwork(n.prefix)}>{@html n.icon}{n.name}</button>
    {/each}
  </div>
</div>

<style>
  .link .net { display: grid; place-items: center; width: 2.25rem; height: 2.25rem; flex: none; border-radius: .5rem; background: var(--soft); }
  .link input { flex: 1; min-width: 0; }
  .link input.incomplete { border-color: var(--red); }
  .nets { display: flex; flex-wrap: wrap; gap: .35rem; }
  .net-btn { display: inline-flex; align-items: center; gap: .4rem; padding: .35rem .7rem; border: 1px solid var(--line); border-radius: 999px; font-size: .82rem; font-weight: 600; }
  .net-btn:hover { border-color: var(--blue); color: var(--blue); }
</style>
