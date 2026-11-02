<script lang="ts">
  import CloudUpload from "@lucide/svelte/icons/cloud-upload";
  import { t } from "./i18n";

  // The question before a note's text first goes to an AI in the cloud:
  // what leaves the computer (the note's text), where it goes, and that
  // it is asked once for each. Cancel sends nothing.
  let { dest, action, onAnswer }: { dest: "openrouter" | "blup"; action: "translate" | "narrate"; onAnswer: (ok: boolean) => void } = $props();
  const a = t.aip;
  const name = $derived(dest === "blup" ? a.whoBlup : a.whoOpenRouter);
  const where = $derived(dest === "blup" ? a.whereBlup : a.whereOpenRouter);
</script>

<svelte:window onkeydown={(e) => e.key === "Escape" && onAnswer(false)} />
<div class="shade" role="presentation" onclick={() => onAnswer(false)}></div>
<div class="dialog" role="alertdialog" aria-modal="true" aria-labelledby="aic-title" aria-describedby="aic-what">
  <header class="head">
    <CloudUpload size={20} />
    <h2 id="aic-title">{a.title}</h2>
  </header>
  <p id="aic-what">{a.what.replace("{action}", action === "narrate" ? a.narrate : a.translate).replace("{where}", where)}</p>
  <ul>
    <li>{a.plain}</li>
    <li>{a.locked}</li>
    <li>{a.local}</li>
  </ul>
  <small class="muted">{a.once.replace("{name}", name)}</small>
  <div class="actions">
    <button class="btn ghost" onclick={() => onAnswer(false)}>{a.cancel}</button>
    <button class="btn" {@attach (el) => el.focus()} onclick={() => onAnswer(true)}>{a.accept}</button>
  </div>
</div>

<style>
  .shade { position: fixed; inset: 0; z-index: 80; background: rgb(0 0 0 / .5); }
  .dialog { position: fixed; z-index: 81; inset: 0; margin: auto; width: min(30rem, calc(100% - 2rem)); height: fit-content; max-height: calc(100% - 2rem); overflow: auto; display: grid; gap: .85rem; padding: 1.25rem; border: 1px solid var(--line); border-radius: .85rem; background: var(--panel); color: var(--text); box-shadow: 0 24px 60px rgb(0 0 0 / .35); }
  .head { display: flex; align-items: center; gap: .6rem; color: var(--blue); }
  .head h2 { flex: 1; font-size: 1.05rem; font-weight: 700; color: var(--text); }
  p { margin: 0; font-size: .92rem; line-height: 1.45; }
  ul { margin: 0; padding-inline-start: 1.1rem; display: grid; gap: .3rem; font-size: .85rem; color: var(--muted); }
  small { font-size: .8rem; line-height: 1.4; }
  .actions { display: flex; justify-content: flex-end; gap: .5rem; flex-wrap: wrap; }
</style>
