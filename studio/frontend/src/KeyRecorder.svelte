<script lang="ts">
  import { comboOf } from "./keys";
  import { t } from "./i18n";

  // A shortcut field: click, press the keys. Esc cancels; needs a modifier
  // (Ctrl, Alt, Shift or Super) unless free is set.
  let { value, onChange, free = false }: { value: string; onChange: (keys: string) => void; free?: boolean } = $props();
  const k = t.keys;
  let listening = $state(false);
  let hint = $state("");

  function key(e: KeyboardEvent) {
    if (!listening) return;
    e.preventDefault();
    e.stopPropagation();
    if (e.key === "Escape") { listening = false; hint = ""; return; }
    const combo = comboOf(e);
    if (!combo) return;
    if (!free && !/^(Ctrl|Alt|Shift|Super)\+/.test(combo)) { hint = k.needModifier; return; }
    listening = false;
    hint = "";
    onChange(combo);
  }
</script>

<svelte:window onkeydowncapture={key} />

<span class="rec">
  <button class="keys" class:listening onclick={() => { listening = !listening; hint = ""; }} onblur={() => (listening = false)}>
    {#if listening}<span class="muted">{k.press}</span>
    {:else if value}{#each value.split("+") as part, i (i)}<kbd>{part}</kbd>{/each}
    {:else}<span class="muted">{k.none}</span>{/if}
  </button>
  {#if hint}<small class="bad">{hint}</small>{/if}
</span>

<style>
  .rec { display: inline-grid; justify-items: end; gap: .2rem; }
  .keys { display: inline-flex; align-items: center; gap: .25rem; min-width: 9rem; min-height: 2.1rem; justify-content: center; padding: .25rem .6rem; border: 1px solid var(--line); border-radius: .45rem; background: var(--bg); font-size: .82rem; }
  .keys:hover { border-color: var(--blue); }
  .keys.listening { border-color: var(--blue); box-shadow: 0 0 0 2px color-mix(in srgb, var(--blue) 30%, transparent); }
  kbd { padding: .1rem .4rem; border: 1px solid var(--line); border-block-end-width: 2px; border-radius: .3rem; background: var(--soft); font-family: inherit; font-size: .78rem; font-weight: 600; }
  small { font-size: .75rem; }
</style>
