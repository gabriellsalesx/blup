<script lang="ts">
  import Delete from "@lucide/svelte/icons/delete";
  import X from "@lucide/svelte/icons/x";
  import { NotesService } from "../../bindings/github.com/gabriellsalesx/blup/mobile";
  import { fmt, t } from "../desk/i18n";

  // The keypad of a four-digit PIN: it answers on the fourth digit.
  let { title, hint = "", error = "", onpin, oncancel }: { title: string; hint?: string; error?: string; onpin: (pin: string) => void; oncancel?: () => void } = $props();

  let value = $state("");
  function press(digit: string) {
    NotesService.Haptic();
    value += digit;
    if (value.length < 4) return;
    const pin = value;
    value = "";
    onpin(pin);
  }
</script>

<div class="pad">
  {#if oncancel}<button type="button" class="close" aria-label={t.common.cancel} onclick={oncancel}><X size={24} /></button>{/if}
  <img src="/blups/blup.svg" alt="" width="56" height="56" />
  <h1>{title}</h1>
  {#if hint}<small>{hint}</small>{/if}
  <div class="dots" role="status" aria-label={fmt(t.pin.digits, { n: value.length })}>
    {#each [0, 1, 2, 3] as i (i)}<i class={{ on: i < value.length }}></i>{/each}
  </div>
  <p role="alert">{error}</p>
  <div class="keys">
    {#each ["1", "2", "3", "4", "5", "6", "7", "8", "9"] as digit (digit)}
      <button type="button" onclick={() => press(digit)}>{digit}</button>
    {/each}
    <span></span>
    <button type="button" onclick={() => press("0")}>0</button>
    <button type="button" class="erase" aria-label={t.pin.erase} disabled={!value} onclick={() => { NotesService.Haptic(); value = value.slice(0, -1); }}><Delete size={26} /></button>
  </div>
</div>

<style>
  .pad { position: fixed; inset: 0; z-index: 40; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 1rem; padding: var(--sat) 1rem var(--sab); background: var(--bg); }
  .close { position: absolute; inset-block-start: calc(.5rem + var(--sat)); inset-inline-start: .35rem; display: grid; place-items: center; width: 2.75rem; height: 2.75rem; }
  h1 { font-size: 1.15rem; font-weight: 700; text-align: center; }
  .dots { display: flex; gap: 1rem; }
  .dots i { width: .85rem; height: .85rem; border: 2px solid var(--muted); border-radius: 50%; }
  .dots .on { border-color: var(--blue); background: var(--blue); }
  small { max-width: 19rem; padding-inline: 1rem; font-size: .85rem; line-height: 1.35; color: var(--muted); text-align: center; }
  p { min-height: 1.3rem; font-size: .875rem; font-weight: 600; color: var(--red); text-align: center; }
  .keys { display: grid; grid-template-columns: repeat(3, 4.5rem); gap: .9rem 1.25rem; }
  .keys button { display: grid; place-items: center; height: 4.5rem; border-radius: 50%; background: var(--soft); font-size: 1.5rem; font-weight: 600; }
  .keys button:active { background: var(--line); }
  .keys .erase { background: none; color: var(--muted); }
  .keys .erase:disabled { opacity: .35; }
</style>
