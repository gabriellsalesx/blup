<script lang="ts">
  import Sun from "@lucide/svelte/icons/sun";
  import Moon from "@lucide/svelte/icons/moon";
  import Monitor from "@lucide/svelte/icons/monitor";
  import { look, setTheme } from "./theme.svelte";
  import { t } from "./i18n";

  let { big = false }: { big?: boolean } = $props();
  const options = [
    { key: "light", label: t.look2.light, Icon: Sun },
    { key: "dark", label: t.look2.dark, Icon: Moon },
    { key: "system", label: t.look2.system, Icon: Monitor },
  ];
</script>

<div class="picker" class:big role="radiogroup" aria-label={t.look2.label}>
  {#each options as { key, label, Icon } (key)}
    <button role="radio" aria-checked={look.theme === key} class:on={look.theme === key} onclick={() => setTheme(key)} title={label}>
      <Icon size={big ? 18 : 15} />{#if big}<span>{label}</span>{/if}
    </button>
  {/each}
</div>

<style>
  .picker { display: inline-flex; gap: .2rem; padding: .2rem; border: 1px solid var(--line); border-radius: .6rem; background: var(--panel); }
  button { display: inline-flex; align-items: center; gap: .4rem; padding: .35rem .5rem; border-radius: .45rem; color: var(--muted); font-size: .85rem; font-weight: 600; }
  button:hover { color: var(--text); }
  button.on { background: var(--soft); color: var(--text); }
  .big button { padding: .6rem 1rem; }
</style>
